package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

// RabbitMQEventBus implements EventBus using RabbitMQ
type RabbitMQEventBus struct {
	conn          *amqp.Connection
	channel       *amqp.Channel
	config        Config
	handlers      map[string][]EventHandler
	mutex         sync.RWMutex
	subscriptions map[string]<-chan amqp.Delivery
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewRabbitMQEventBus creates a new RabbitMQ event bus
func NewRabbitMQEventBus(config Config) (*RabbitMQEventBus, error) {
	conn, err := amqp.Dial(config.BrokerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange
	err = channel.ExchangeDeclare(
		config.Exchange,     // name
		config.ExchangeType, // type
		config.Durable,      // durable
		config.AutoDelete,   // auto-deleted
		false,               // internal
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Set QoS
	if err := channel.Qos(config.PrefetchCount, 0, false); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	bus := &RabbitMQEventBus{
		conn:          conn,
		channel:       channel,
		config:        config,
		handlers:      make(map[string][]EventHandler),
		subscriptions: make(map[string]<-chan amqp.Delivery),
		stopChan:      make(chan struct{}),
	}

	// Monitor connection
	go bus.monitorConnection()

	return bus, nil
}

// Publish publishes an event to RabbitMQ
func (b *RabbitMQEventBus) Publish(ctx context.Context, event Event) error {
	// Serialize event
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	// Determine routing key (event type)
	routingKey := event.Type

	// Publish message
	err = b.channel.Publish(
		b.config.Exchange, // exchange
		routingKey,        // routing key
		false,             // mandatory
		false,             // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // persistent
			Timestamp:    event.Timestamp,
			MessageId:    event.ID,
			Type:         event.Type,
			Headers: amqp.Table{
				"source":  event.Source,
				"version": event.Version,
			},
		},
	)

	if err != nil {
		return fmt.Errorf("%w: %v", ErrPublishFailed, err)
	}

	return nil
}

// PublishBatch publishes multiple events
func (b *RabbitMQEventBus) PublishBatch(ctx context.Context, events []Event) error {
	for _, event := range events {
		if err := b.Publish(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe subscribes to events of a specific type
func (b *RabbitMQEventBus) Subscribe(ctx context.Context, eventType string, handler EventHandler) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Add handler
	b.handlers[eventType] = append(b.handlers[eventType], handler)

	// If already subscribed, don't create new queue
	if _, exists := b.subscriptions[eventType]; exists {
		return nil
	}

	// Declare queue
	queueName := fmt.Sprintf("%s.%s", b.config.QueuePrefix, eventType)
	queue, err := b.channel.QueueDeclare(
		queueName,           // name
		b.config.Durable,    // durable
		b.config.AutoDelete, // auto-delete
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return fmt.Errorf("%w: failed to declare queue: %v", ErrSubscribeFailed, err)
	}

	// Bind queue to exchange
	err = b.channel.QueueBind(
		queue.Name,        // queue name
		eventType,         // routing key
		b.config.Exchange, // exchange
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		return fmt.Errorf("%w: failed to bind queue: %v", ErrSubscribeFailed, err)
	}

	// Start consuming
	deliveries, err := b.channel.Consume(
		queue.Name,            // queue
		b.config.ConsumerTag,  // consumer
		false,                 // auto-ack
		false,                 // exclusive
		false,                 // no-local
		false,                 // no-wait
		nil,                   // args
	)
	if err != nil {
		return fmt.Errorf("%w: failed to start consuming: %v", ErrSubscribeFailed, err)
	}

	b.subscriptions[eventType] = deliveries

	// Start message handler
	b.wg.Add(1)
	go b.handleMessages(eventType, deliveries)

	return nil
}

// SubscribeMultiple subscribes to multiple event types
func (b *RabbitMQEventBus) SubscribeMultiple(ctx context.Context, eventTypes []string, handler EventHandler) error {
	for _, eventType := range eventTypes {
		if err := b.Subscribe(ctx, eventType, handler); err != nil {
			return err
		}
	}
	return nil
}

// Unsubscribe removes subscriptions for an event type
func (b *RabbitMQEventBus) Unsubscribe(ctx context.Context, eventType string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	delete(b.handlers, eventType)
	delete(b.subscriptions, eventType)

	return nil
}

// Close closes the RabbitMQ connection
func (b *RabbitMQEventBus) Close() error {
	close(b.stopChan)
	b.wg.Wait()

	if b.channel != nil {
		b.channel.Close()
	}

	if b.conn != nil {
		b.conn.Close()
	}

	return nil
}

// handleMessages processes incoming messages
func (b *RabbitMQEventBus) handleMessages(eventType string, deliveries <-chan amqp.Delivery) {
	defer b.wg.Done()

	for {
		select {
		case <-b.stopChan:
			return
		case delivery, ok := <-deliveries:
			if !ok {
				return
			}

			// Parse event
			var event Event
			if err := json.Unmarshal(delivery.Body, &event); err != nil {
				if b.config.OnError != nil {
					b.config.OnError(err, event)
				}
				delivery.Nack(false, false) // Don't requeue
				continue
			}

			// Execute handlers
			b.mutex.RLock()
			handlers := b.handlers[eventType]
			b.mutex.RUnlock()

			success := true
			for _, handler := range handlers {
				if err := b.executeWithRetry(handler, event); err != nil {
					success = false
					if b.config.OnError != nil {
						b.config.OnError(err, event)
					}
				}
			}

			// Acknowledge or reject
			if success {
				delivery.Ack(false)
			} else {
				delivery.Nack(false, true) // Requeue for retry
			}
		}
	}
}

// executeWithRetry executes a handler with retry logic
func (b *RabbitMQEventBus) executeWithRetry(handler EventHandler, event Event) error {
	var lastErr error

	for attempt := 0; attempt <= b.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(b.config.RetryDelay)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := handler(ctx, event)
		cancel()

		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("failed after %d attempts: %w", b.config.RetryAttempts, lastErr)
}

// monitorConnection monitors RabbitMQ connection and attempts reconnection
func (b *RabbitMQEventBus) monitorConnection() {
	for {
		select {
		case <-b.stopChan:
			return
		case err := <-b.conn.NotifyClose(make(chan *amqp.Error)):
			if err != nil {
				fmt.Printf("RabbitMQ connection closed: %v\n", err)
				// In production, implement reconnection logic
			}
		}
	}
}

// Example usage with RabbitMQ:
//
// // Initialize RabbitMQ event bus
// eventBus, err := events.NewRabbitMQEventBus(events.Config{
//     BrokerURL:     "amqp://guest:guest@localhost:5672/",
//     Exchange:      "logistics-events",
//     ExchangeType:  "topic",
//     QueuePrefix:   "logistics",
//     ConsumerTag:   "order-service",
//     Durable:       true,
//     PrefetchCount: 10,
//     RetryAttempts: 3,
//     RetryDelay:    2 * time.Second,
//     OnError: func(err error, event events.Event) {
//         log.Printf("Event error: %v, Event: %+v", err, event)
//     },
// })
// if err != nil {
//     log.Fatal(err)
// }
// defer eventBus.Close()
//
// // Subscribe to order events
// err = eventBus.Subscribe(context.Background(), "order.*", func(ctx context.Context, event events.Event) error {
//     log.Printf("Received event: %s from %s", event.Type, event.Source)
//
//     switch event.Type {
//     case events.EventOrderCreated:
//         var data events.OrderCreatedData
//         if err := events.DeserializeEventData(event.Data.([]byte), &data); err != nil {
//             return err
//         }
//         // Process order created event
//         return processOrderCreated(ctx, data)
//
//     case events.EventOrderCompleted:
//         // Process order completed event
//         return processOrderCompleted(ctx, event)
//     }
//
//     return nil
// })
//
// // Publish event
// event := events.NewEventBuilder(events.EventOrderCreated, "order-service").
//     WithData(events.OrderCreatedData{
//         OrderID:     "order-123",
//         CustomerID:  "customer-456",
//         TotalAmount: 99.99,
//         ItemCount:   3,
//         CreatedAt:   time.Now(),
//     }).
//     WithMetadata("correlation_id", "req-789").
//     Build()
//
// if err := eventBus.Publish(context.Background(), event); err != nil {
//     log.Printf("Failed to publish event: %v", err)
// }
