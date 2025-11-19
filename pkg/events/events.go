package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrPublishFailed is returned when event publication fails
	ErrPublishFailed = errors.New("failed to publish event")
	// ErrSubscribeFailed is returned when event subscription fails
	ErrSubscribeFailed = errors.New("failed to subscribe to events")
	// ErrInvalidEvent is returned when event is invalid
	ErrInvalidEvent = errors.New("invalid event")
)

// Event represents a domain event
type Event struct {
	// ID uniquely identifies the event
	ID string `json:"id"`
	// Type is the event type (e.g., "order.created", "shipment.delivered")
	Type string `json:"type"`
	// Source identifies the service that produced the event
	Source string `json:"source"`
	// Timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`
	// Data contains the event payload
	Data interface{} `json:"data"`
	// Metadata contains additional context
	Metadata map[string]string `json:"metadata,omitempty"`
	// Version for event schema versioning
	Version string `json:"version"`
}

// EventHandler processes events
type EventHandler func(ctx context.Context, event Event) error

// Publisher publishes events
type Publisher interface {
	Publish(ctx context.Context, event Event) error
	PublishBatch(ctx context.Context, events []Event) error
	Close() error
}

// Subscriber subscribes to events
type Subscriber interface {
	Subscribe(ctx context.Context, eventType string, handler EventHandler) error
	SubscribeMultiple(ctx context.Context, eventTypes []string, handler EventHandler) error
	Unsubscribe(ctx context.Context, eventType string) error
	Close() error
}

// EventBus combines Publisher and Subscriber
type EventBus interface {
	Publisher
	Subscriber
}

// Config holds event bus configuration
type Config struct {
	// BrokerURL for message broker (e.g., "amqp://localhost:5672" for RabbitMQ)
	BrokerURL string
	// Exchange name for RabbitMQ
	Exchange string
	// ExchangeType (direct, topic, fanout, headers)
	ExchangeType string
	// Queue name prefix
	QueuePrefix string
	// ConsumerTag identifies the consumer
	ConsumerTag string
	// Durable queues survive broker restart
	Durable bool
	// AutoDelete removes queue when no consumers
	AutoDelete bool
	// PrefetchCount for QoS
	PrefetchCount int
	// RetryAttempts for failed messages
	RetryAttempts int
	// RetryDelay between retry attempts
	RetryDelay time.Duration
	// OnError callback for error handling
	OnError func(err error, event Event)
}

// DefaultConfig returns a default event bus configuration
func DefaultConfig() Config {
	return Config{
		BrokerURL:     "amqp://guest:guest@localhost:5672/",
		Exchange:      "logistics-events",
		ExchangeType:  "topic",
		QueuePrefix:   "logistics",
		ConsumerTag:   "logistics-consumer",
		Durable:       true,
		AutoDelete:    false,
		PrefetchCount: 10,
		RetryAttempts: 3,
		RetryDelay:    2 * time.Second,
	}
}

// InMemoryEventBus implements EventBus for testing/development
type InMemoryEventBus struct {
	handlers map[string][]EventHandler
	mutex    sync.RWMutex
	events   []Event
	closed   bool
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[string][]EventHandler),
		events:   make([]Event, 0),
	}
}

// Publish publishes an event
func (b *InMemoryEventBus) Publish(ctx context.Context, event Event) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.closed {
		return errors.New("event bus is closed")
	}

	// Store event
	b.events = append(b.events, event)

	// Find and execute handlers
	handlers := b.handlers[event.Type]
	handlers = append(handlers, b.handlers["*"]...) // Wildcard handlers

	// Execute handlers asynchronously
	for _, handler := range handlers {
		go func(h EventHandler, e Event) {
			if err := h(ctx, e); err != nil {
				// Log error (in production, use proper logger)
				fmt.Printf("Error handling event %s: %v\n", e.Type, err)
			}
		}(handler, event)
	}

	return nil
}

// PublishBatch publishes multiple events
func (b *InMemoryEventBus) PublishBatch(ctx context.Context, events []Event) error {
	for _, event := range events {
		if err := b.Publish(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe subscribes to events of a specific type
func (b *InMemoryEventBus) Subscribe(ctx context.Context, eventType string, handler EventHandler) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.closed {
		return errors.New("event bus is closed")
	}

	b.handlers[eventType] = append(b.handlers[eventType], handler)
	return nil
}

// SubscribeMultiple subscribes to multiple event types
func (b *InMemoryEventBus) SubscribeMultiple(ctx context.Context, eventTypes []string, handler EventHandler) error {
	for _, eventType := range eventTypes {
		if err := b.Subscribe(ctx, eventType, handler); err != nil {
			return err
		}
	}
	return nil
}

// Unsubscribe removes subscriptions for an event type
func (b *InMemoryEventBus) Unsubscribe(ctx context.Context, eventType string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	delete(b.handlers, eventType)
	return nil
}

// Close closes the event bus
func (b *InMemoryEventBus) Close() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.closed = true
	b.handlers = make(map[string][]EventHandler)
	return nil
}

// GetEvents returns all published events (for testing)
func (b *InMemoryEventBus) GetEvents() []Event {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return append([]Event{}, b.events...)
}

// EventBuilder helps build events
type EventBuilder struct {
	event Event
}

// NewEventBuilder creates a new event builder
func NewEventBuilder(eventType, source string) *EventBuilder {
	return &EventBuilder{
		event: Event{
			ID:        generateEventID(),
			Type:      eventType,
			Source:    source,
			Timestamp: time.Now(),
			Version:   "1.0",
			Metadata:  make(map[string]string),
		},
	}
}

// WithData sets the event data
func (b *EventBuilder) WithData(data interface{}) *EventBuilder {
	b.event.Data = data
	return b
}

// WithMetadata adds metadata
func (b *EventBuilder) WithMetadata(key, value string) *EventBuilder {
	if b.event.Metadata == nil {
		b.event.Metadata = make(map[string]string)
	}
	b.event.Metadata[key] = value
	return b
}

// WithVersion sets the event version
func (b *EventBuilder) WithVersion(version string) *EventBuilder {
	b.event.Version = version
	return b
}

// Build returns the constructed event
func (b *EventBuilder) Build() Event {
	return b.event
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// EventStore stores events for event sourcing
type EventStore interface {
	Save(ctx context.Context, event Event) error
	SaveBatch(ctx context.Context, events []Event) error
	GetByID(ctx context.Context, eventID string) (*Event, error)
	GetByType(ctx context.Context, eventType string, limit int) ([]Event, error)
	GetBySource(ctx context.Context, source string, limit int) ([]Event, error)
	GetSince(ctx context.Context, since time.Time, limit int) ([]Event, error)
}

// InMemoryEventStore implements EventStore for testing
type InMemoryEventStore struct {
	events map[string]Event
	mutex  sync.RWMutex
}

// NewInMemoryEventStore creates a new in-memory event store
func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		events: make(map[string]Event),
	}
}

// Save saves an event
func (s *InMemoryEventStore) Save(ctx context.Context, event Event) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.events[event.ID] = event
	return nil
}

// SaveBatch saves multiple events
func (s *InMemoryEventStore) SaveBatch(ctx context.Context, events []Event) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, event := range events {
		s.events[event.ID] = event
	}
	return nil
}

// GetByID retrieves an event by ID
func (s *InMemoryEventStore) GetByID(ctx context.Context, eventID string) (*Event, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	event, exists := s.events[eventID]
	if !exists {
		return nil, errors.New("event not found")
	}

	return &event, nil
}

// GetByType retrieves events by type
func (s *InMemoryEventStore) GetByType(ctx context.Context, eventType string, limit int) ([]Event, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var results []Event
	for _, event := range s.events {
		if event.Type == eventType {
			results = append(results, event)
			if len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// GetBySource retrieves events by source
func (s *InMemoryEventStore) GetBySource(ctx context.Context, source string, limit int) ([]Event, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var results []Event
	for _, event := range s.events {
		if event.Source == source {
			results = append(results, event)
			if len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// GetSince retrieves events since a timestamp
func (s *InMemoryEventStore) GetSince(ctx context.Context, since time.Time, limit int) ([]Event, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var results []Event
	for _, event := range s.events {
		if event.Timestamp.After(since) {
			results = append(results, event)
			if len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// Common event types
const (
	// Order events
	EventOrderCreated   = "order.created"
	EventOrderUpdated   = "order.updated"
	EventOrderCancelled = "order.cancelled"
	EventOrderCompleted = "order.completed"

	// Shipment events
	EventShipmentCreated   = "shipment.created"
	EventShipmentPickedUp  = "shipment.picked_up"
	EventShipmentInTransit = "shipment.in_transit"
	EventShipmentDelivered = "shipment.delivered"

	// Inventory events
	EventInventoryReserved = "inventory.reserved"
	EventInventoryReleased = "inventory.released"
	EventInventoryUpdated  = "inventory.updated"
	EventLowStockAlert     = "inventory.low_stock"

	// Driver events
	EventDriverAssigned   = "driver.assigned"
	EventDriverAvailable  = "driver.available"
	EventDriverOffline    = "driver.offline"
	EventLocationUpdated  = "driver.location_updated"

	// Notification events
	EventNotificationSent   = "notification.sent"
	EventNotificationFailed = "notification.failed"
)

// Example event data structures

// OrderCreatedData represents order.created event data
type OrderCreatedData struct {
	OrderID       string    `json:"order_id"`
	CustomerID    string    `json:"customer_id"`
	TotalAmount   float64   `json:"total_amount"`
	ItemCount     int       `json:"item_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// ShipmentDeliveredData represents shipment.delivered event data
type ShipmentDeliveredData struct {
	ShipmentID     string    `json:"shipment_id"`
	TrackingNumber string    `json:"tracking_number"`
	DriverID       string    `json:"driver_id"`
	DeliveredAt    time.Time `json:"delivered_at"`
	Signature      string    `json:"signature,omitempty"`
}

// InventoryReservedData represents inventory.reserved event data
type InventoryReservedData struct {
	ProductID      string `json:"product_id"`
	Quantity       int    `json:"quantity"`
	ReservationID  string `json:"reservation_id"`
	OrderID        string `json:"order_id"`
}

// Example usage:
//
// // Initialize event bus
// eventBus := events.NewInMemoryEventBus()
// defer eventBus.Close()
//
// // Subscribe to events
// eventBus.Subscribe(context.Background(), events.EventOrderCreated, func(ctx context.Context, event events.Event) error {
//     var data events.OrderCreatedData
//     if err := json.Unmarshal(event.Data.([]byte), &data); err != nil {
//         return err
//     }
//     log.Printf("Order created: %s with total %f", data.OrderID, data.TotalAmount)
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
//     WithMetadata("user_id", "user-789").
//     Build()
//
// if err := eventBus.Publish(context.Background(), event); err != nil {
//     log.Printf("Failed to publish event: %v", err)
// }

// SerializeEventData serializes event data to JSON bytes
func SerializeEventData(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// DeserializeEventData deserializes JSON bytes to event data
func DeserializeEventData(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
