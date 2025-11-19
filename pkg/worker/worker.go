package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrWorkerStopped is returned when worker is stopped
	ErrWorkerStopped = errors.New("worker is stopped")
	// ErrJobFailed is returned when job execution fails
	ErrJobFailed = errors.New("job failed")
	// ErrInvalidJob is returned when job is invalid
	ErrInvalidJob = errors.New("invalid job")
)

// Job represents a background job
type Job struct {
	// ID uniquely identifies the job
	ID string
	// Type is the job type (e.g., "send_email", "process_order")
	Type string
	// Payload contains job data
	Payload interface{}
	// Priority (higher number = higher priority)
	Priority int
	// MaxRetries is the maximum retry attempts
	MaxRetries int
	// Attempts tracks current attempt count
	Attempts int
	// CreatedAt is when the job was created
	CreatedAt time.Time
	// ScheduledAt is when the job should be executed
	ScheduledAt time.Time
	// Metadata contains additional context
	Metadata map[string]string
}

// JobHandler processes jobs
type JobHandler func(ctx context.Context, job Job) error

// Queue manages job queuing
type Queue interface {
	Enqueue(ctx context.Context, job Job) error
	Dequeue(ctx context.Context) (*Job, error)
	EnqueueBatch(ctx context.Context, jobs []Job) error
	Size(ctx context.Context) (int, error)
	Close() error
}

// Worker processes jobs from a queue
type Worker struct {
	id             string
	queue          Queue
	handlers       map[string]JobHandler
	concurrency    int
	pollInterval   time.Duration
	retryDelay     time.Duration
	stopChan       chan struct{}
	wg             sync.WaitGroup
	mutex          sync.RWMutex
	onJobComplete  func(job Job, err error)
	onJobStart     func(job Job)
}

// Config holds worker configuration
type Config struct {
	// Concurrency is the number of concurrent workers
	Concurrency int
	// PollInterval is how often to poll for jobs
	PollInterval time.Duration
	// RetryDelay is delay between retries
	RetryDelay time.Duration
	// OnJobComplete callback when job completes
	OnJobComplete func(job Job, err error)
	// OnJobStart callback when job starts
	OnJobStart func(job Job)
}

// DefaultConfig returns default worker configuration
func DefaultConfig() Config {
	return Config{
		Concurrency:  5,
		PollInterval: 1 * time.Second,
		RetryDelay:   5 * time.Second,
	}
}

// NewWorker creates a new worker
func NewWorker(id string, queue Queue, config Config) *Worker {
	return &Worker{
		id:            id,
		queue:         queue,
		handlers:      make(map[string]JobHandler),
		concurrency:   config.Concurrency,
		pollInterval:  config.PollInterval,
		retryDelay:    config.RetryDelay,
		stopChan:      make(chan struct{}),
		onJobComplete: config.OnJobComplete,
		onJobStart:    config.OnJobStart,
	}
}

// RegisterHandler registers a job handler
func (w *Worker) RegisterHandler(jobType string, handler JobHandler) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.handlers[jobType] = handler
}

// Start starts the worker pool
func (w *Worker) Start() {
	for i := 0; i < w.concurrency; i++ {
		w.wg.Add(1)
		go w.run(i)
	}
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

// run is the main worker loop
func (w *Worker) run(workerID int) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.processJob(workerID)
		}
	}
}

// processJob processes a single job
func (w *Worker) processJob(workerID int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Dequeue job
	job, err := w.queue.Dequeue(ctx)
	if err != nil {
		return // No jobs available or error
	}

	if job == nil {
		return
	}

	// Find handler
	w.mutex.RLock()
	handler, exists := w.handlers[job.Type]
	w.mutex.RUnlock()

	if !exists {
		fmt.Printf("No handler for job type: %s\n", job.Type)
		return
	}

	// Execute job
	if w.onJobStart != nil {
		w.onJobStart(*job)
	}

	job.Attempts++
	err = handler(ctx, *job)

	// Handle result
	if err != nil {
		if job.Attempts < job.MaxRetries {
			// Retry
			time.Sleep(w.retryDelay)
			if requeueErr := w.queue.Enqueue(ctx, *job); requeueErr != nil {
				fmt.Printf("Failed to requeue job %s: %v\n", job.ID, requeueErr)
			}
		}

		if w.onJobComplete != nil {
			w.onJobComplete(*job, err)
		}
	} else {
		if w.onJobComplete != nil {
			w.onJobComplete(*job, nil)
		}
	}
}

// InMemoryQueue implements Queue using in-memory channel
type InMemoryQueue struct {
	jobs    chan Job
	stopped bool
	mutex   sync.RWMutex
}

// NewInMemoryQueue creates a new in-memory queue
func NewInMemoryQueue(capacity int) *InMemoryQueue {
	return &InMemoryQueue{
		jobs: make(chan Job, capacity),
	}
}

// Enqueue adds a job to the queue
func (q *InMemoryQueue) Enqueue(ctx context.Context, job Job) error {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	if q.stopped {
		return ErrWorkerStopped
	}

	select {
	case q.jobs <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Dequeue removes a job from the queue
func (q *InMemoryQueue) Dequeue(ctx context.Context) (*Job, error) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	if q.stopped {
		return nil, ErrWorkerStopped
	}

	select {
	case job := <-q.jobs:
		return &job, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, nil // No jobs available
	}
}

// EnqueueBatch adds multiple jobs
func (q *InMemoryQueue) EnqueueBatch(ctx context.Context, jobs []Job) error {
	for _, job := range jobs {
		if err := q.Enqueue(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

// Size returns the queue size
func (q *InMemoryQueue) Size(ctx context.Context) (int, error) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	return len(q.jobs), nil
}

// Close closes the queue
func (q *InMemoryQueue) Close() error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if !q.stopped {
		q.stopped = true
		close(q.jobs)
	}

	return nil
}

// JobBuilder helps build jobs
type JobBuilder struct {
	job Job
}

// NewJobBuilder creates a new job builder
func NewJobBuilder(jobType string) *JobBuilder {
	return &JobBuilder{
		job: Job{
			ID:          generateJobID(),
			Type:        jobType,
			Priority:    0,
			MaxRetries:  3,
			Attempts:    0,
			CreatedAt:   time.Now(),
			ScheduledAt: time.Now(),
			Metadata:    make(map[string]string),
		},
	}
}

// WithPayload sets the job payload
func (b *JobBuilder) WithPayload(payload interface{}) *JobBuilder {
	b.job.Payload = payload
	return b
}

// WithPriority sets the job priority
func (b *JobBuilder) WithPriority(priority int) *JobBuilder {
	b.job.Priority = priority
	return b
}

// WithMaxRetries sets max retry attempts
func (b *JobBuilder) WithMaxRetries(maxRetries int) *JobBuilder {
	b.job.MaxRetries = maxRetries
	return b
}

// WithScheduledAt schedules the job for later
func (b *JobBuilder) WithScheduledAt(scheduledAt time.Time) *JobBuilder {
	b.job.ScheduledAt = scheduledAt
	return b
}

// WithDelay schedules the job after a delay
func (b *JobBuilder) WithDelay(delay time.Duration) *JobBuilder {
	b.job.ScheduledAt = time.Now().Add(delay)
	return b
}

// WithMetadata adds metadata
func (b *JobBuilder) WithMetadata(key, value string) *JobBuilder {
	if b.job.Metadata == nil {
		b.job.Metadata = make(map[string]string)
	}
	b.job.Metadata[key] = value
	return b
}

// Build returns the constructed job
func (b *JobBuilder) Build() Job {
	return b.job
}

// generateJobID generates a unique job ID
func generateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}

// Common job types
const (
	JobSendEmail              = "send_email"
	JobSendSMS                = "send_sms"
	JobProcessPayment         = "process_payment"
	JobGenerateReport         = "generate_report"
	JobSyncInventory          = "sync_inventory"
	JobCalculateRouteMetrics  = "calculate_route_metrics"
	JobSendNotification       = "send_notification"
	JobCleanupOldData         = "cleanup_old_data"
)

// Example job payloads

// SendEmailPayload represents email job data
type SendEmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	From    string `json:"from,omitempty"`
}

// SendSMSPayload represents SMS job data
type SendSMSPayload struct {
	PhoneNumber string `json:"phone_number"`
	Message     string `json:"message"`
}

// ProcessPaymentPayload represents payment job data
type ProcessPaymentPayload struct {
	OrderID     string  `json:"order_id"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	PaymentMethod string `json:"payment_method"`
}

// Example usage:
//
// // Create queue and worker
// queue := worker.NewInMemoryQueue(1000)
// defer queue.Close()
//
// w := worker.NewWorker("worker-1", queue, worker.Config{
//     Concurrency:  10,
//     PollInterval: 1 * time.Second,
//     RetryDelay:   5 * time.Second,
//     OnJobStart: func(job worker.Job) {
//         log.Printf("Starting job %s (type: %s)", job.ID, job.Type)
//     },
//     OnJobComplete: func(job worker.Job, err error) {
//         if err != nil {
//             log.Printf("Job %s failed: %v", job.ID, err)
//         } else {
//             log.Printf("Job %s completed successfully", job.ID)
//         }
//     },
// })
//
// // Register handlers
// w.RegisterHandler(worker.JobSendEmail, func(ctx context.Context, job worker.Job) error {
//     payload := job.Payload.(worker.SendEmailPayload)
//     return emailService.Send(ctx, payload.To, payload.Subject, payload.Body)
// })
//
// w.RegisterHandler(worker.JobSendSMS, func(ctx context.Context, job worker.Job) error {
//     payload := job.Payload.(worker.SendSMSPayload)
//     return smsService.Send(ctx, payload.PhoneNumber, payload.Message)
// })
//
// // Start worker
// w.Start()
// defer w.Stop()
//
// // Enqueue jobs
// emailJob := worker.NewJobBuilder(worker.JobSendEmail).
//     WithPayload(worker.SendEmailPayload{
//         To:      "customer@example.com",
//         Subject: "Order Confirmation",
//         Body:    "Your order has been confirmed",
//     }).
//     WithPriority(1).
//     WithMaxRetries(3).
//     Build()
//
// if err := queue.Enqueue(context.Background(), emailJob); err != nil {
//     log.Printf("Failed to enqueue job: %v", err)
// }
//
// // Schedule delayed job
// reminderJob := worker.NewJobBuilder(worker.JobSendEmail).
//     WithPayload(worker.SendEmailPayload{
//         To:      "customer@example.com",
//         Subject: "Shipment Reminder",
//         Body:    "Your shipment is arriving today",
//     }).
//     WithDelay(24 * time.Hour).
//     Build()
//
// queue.Enqueue(context.Background(), reminderJob)
