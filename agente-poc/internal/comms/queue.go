package comms

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"agente-poc/internal/logging"
)

// MessageQueue manages offline message queuing with persistence
type MessageQueue struct {
	messages    []QueuedMessage
	mutex       sync.RWMutex
	logger      logging.Logger
	maxSize     int
	persistPath string
	metrics     *QueueMetrics
}

// QueuedMessage represents a queued message with metadata
type QueuedMessage struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    int                    `json:"priority"`
	Timestamp   time.Time              `json:"timestamp"`
	ExpiresAt   time.Time              `json:"expires_at"`
	Retries     int                    `json:"retries"`
	MaxRetries  int                    `json:"max_retries"`
	Data        map[string]interface{} `json:"data"`
	Endpoint    string                 `json:"endpoint"`
	Method      string                 `json:"method"`
	Headers     map[string]string      `json:"headers"`
	LastError   string                 `json:"last_error,omitempty"`
	LastAttempt time.Time              `json:"last_attempt,omitempty"`
}

// QueueMetrics tracks queue performance metrics
type QueueMetrics struct {
	TotalMessages      int64
	ProcessedMessages  int64
	FailedMessages     int64
	ExpiredMessages    int64
	QueueSize          int64
	MaxQueueSize       int64
	TotalRetries       int64
	LastProcessTime    time.Time
	AverageProcessTime time.Duration
	PersistErrors      int64
	LoadErrors         int64
}

// QueueConfig configuration for message queue
type QueueConfig struct {
	MaxSize     int
	PersistPath string
	Logger      logging.Logger
}

// NewMessageQueue creates a new message queue
func NewMessageQueue(config QueueConfig) (*MessageQueue, error) {
	if config.Logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	if config.MaxSize <= 0 {
		config.MaxSize = 10000
	}

	if config.PersistPath == "" {
		config.PersistPath = "/tmp/agent_queue.json"
	}

	queue := &MessageQueue{
		messages:    make([]QueuedMessage, 0),
		logger:      config.Logger,
		maxSize:     config.MaxSize,
		persistPath: config.PersistPath,
		metrics:     &QueueMetrics{MaxQueueSize: int64(config.MaxSize)},
	}

	// Try to load existing messages
	if err := queue.loadFromDisk(); err != nil {
		queue.logger.Warning("Failed to load queue from disk: %v", err)
		queue.metrics.LoadErrors++
	}

	return queue, nil
}

// Enqueue adds a message to the queue
func (q *MessageQueue) Enqueue(message QueuedMessage) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// Check if queue is full
	if len(q.messages) >= q.maxSize {
		// Remove oldest low-priority message
		q.removeOldestLowPriority()
	}

	// Set defaults
	if message.ID == "" {
		message.ID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}
	if message.ExpiresAt.IsZero() {
		message.ExpiresAt = time.Now().Add(24 * time.Hour)
	}
	if message.MaxRetries == 0 {
		message.MaxRetries = 3
	}

	// Insert in priority order
	inserted := false
	for i, existing := range q.messages {
		if message.Priority > existing.Priority {
			q.messages = append(q.messages[:i], append([]QueuedMessage{message}, q.messages[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		q.messages = append(q.messages, message)
	}

	q.metrics.TotalMessages++
	q.metrics.QueueSize = int64(len(q.messages))

	q.logger.Debug("Message enqueued: %s (priority: %d)", message.ID, message.Priority)

	// Persist to disk
	if err := q.saveToDisk(); err != nil {
		q.logger.Error("Failed to persist queue to disk: %v", err)
		q.metrics.PersistErrors++
	}

	return nil
}

// Dequeue removes and returns the highest priority message
func (q *MessageQueue) Dequeue() (*QueuedMessage, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if len(q.messages) == 0 {
		return nil, fmt.Errorf("queue is empty")
	}

	// Remove expired messages first
	q.removeExpiredMessages()

	if len(q.messages) == 0 {
		return nil, fmt.Errorf("no valid messages in queue")
	}

	// Get highest priority message
	message := q.messages[0]
	q.messages = q.messages[1:]

	q.metrics.QueueSize = int64(len(q.messages))
	q.metrics.LastProcessTime = time.Now()

	q.logger.Debug("Message dequeued: %s", message.ID)

	return &message, nil
}

// Peek returns the highest priority message without removing it
func (q *MessageQueue) Peek() (*QueuedMessage, error) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	if len(q.messages) == 0 {
		return nil, fmt.Errorf("queue is empty")
	}

	return &q.messages[0], nil
}

// Requeue adds a message back to the queue with updated retry count
func (q *MessageQueue) Requeue(message QueuedMessage, err error) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	message.Retries++
	message.LastError = err.Error()
	message.LastAttempt = time.Now()

	if message.Retries >= message.MaxRetries {
		q.logger.Warning("Message %s exceeded max retries, dropping", message.ID)
		q.metrics.FailedMessages++
		return fmt.Errorf("message exceeded max retries")
	}

	// Calculate backoff delay
	delay := time.Duration(message.Retries) * time.Second
	message.Timestamp = time.Now().Add(delay)

	// Insert back with updated timestamp
	inserted := false
	for i, existing := range q.messages {
		if message.Priority > existing.Priority ||
			(message.Priority == existing.Priority && message.Timestamp.Before(existing.Timestamp)) {
			q.messages = append(q.messages[:i], append([]QueuedMessage{message}, q.messages[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		q.messages = append(q.messages, message)
	}

	q.metrics.TotalRetries++
	q.metrics.QueueSize = int64(len(q.messages))

	q.logger.Debug("Message requeued: %s (retry: %d/%d)", message.ID, message.Retries, message.MaxRetries)

	return nil
}

// MarkProcessed marks a message as successfully processed
func (q *MessageQueue) MarkProcessed(messageID string) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.metrics.ProcessedMessages++
	q.logger.Debug("Message marked as processed: %s", messageID)

	// Persist updated state
	if err := q.saveToDisk(); err != nil {
		q.logger.Error("Failed to persist queue to disk: %v", err)
		q.metrics.PersistErrors++
	}
}

// Size returns the current queue size
func (q *MessageQueue) Size() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return len(q.messages)
}

// Clear removes all messages from the queue
func (q *MessageQueue) Clear() error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.messages = q.messages[:0]
	q.metrics.QueueSize = 0

	q.logger.Info("Queue cleared")

	return q.saveToDisk()
}

// GetMetrics returns queue metrics
func (q *MessageQueue) GetMetrics() QueueMetrics {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return *q.metrics
}

// removeExpiredMessages removes expired messages from the queue
func (q *MessageQueue) removeExpiredMessages() {
	now := time.Now()
	validMessages := make([]QueuedMessage, 0, len(q.messages))

	for _, message := range q.messages {
		if now.Before(message.ExpiresAt) {
			validMessages = append(validMessages, message)
		} else {
			q.metrics.ExpiredMessages++
		}
	}

	q.messages = validMessages
}

// removeOldestLowPriority removes the oldest low-priority message
func (q *MessageQueue) removeOldestLowPriority() {
	if len(q.messages) == 0 {
		return
	}

	// Find the oldest message with the lowest priority
	oldestIndex := 0
	for i, message := range q.messages {
		if message.Priority < q.messages[oldestIndex].Priority ||
			(message.Priority == q.messages[oldestIndex].Priority &&
				message.Timestamp.Before(q.messages[oldestIndex].Timestamp)) {
			oldestIndex = i
		}
	}

	// Remove the message
	q.messages = append(q.messages[:oldestIndex], q.messages[oldestIndex+1:]...)
	q.logger.Debug("Removed oldest low-priority message to make space")
}

// saveToDisk persists the queue to disk
func (q *MessageQueue) saveToDisk() error {
	if q.persistPath == "" {
		return nil
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(q.persistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal queue data
	data, err := json.Marshal(q.messages)
	if err != nil {
		return fmt.Errorf("failed to marshal queue data: %w", err)
	}

	// Write to temporary file first
	tempPath := q.persistPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, q.persistPath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// loadFromDisk loads the queue from disk
func (q *MessageQueue) loadFromDisk() error {
	if q.persistPath == "" {
		return nil
	}

	// Check if file exists
	if _, err := os.Stat(q.persistPath); os.IsNotExist(err) {
		return nil // File doesn't exist, start with empty queue
	}

	// Read file
	data, err := os.ReadFile(q.persistPath)
	if err != nil {
		return fmt.Errorf("failed to read queue file: %w", err)
	}

	// Unmarshal data
	var messages []QueuedMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("failed to unmarshal queue data: %w", err)
	}

	q.messages = messages
	q.metrics.QueueSize = int64(len(q.messages))

	q.logger.Info("Loaded %d messages from disk", len(q.messages))

	// Remove expired messages
	q.removeExpiredMessages()

	return nil
}

// CreateHeartbeatMessage creates a heartbeat message for the queue
func CreateHeartbeatMessage(data HeartbeatData) QueuedMessage {
	return QueuedMessage{
		Type:     "heartbeat",
		Priority: 5, // Medium priority
		Data: map[string]interface{}{
			"machine_id":    data.MachineID,
			"timestamp":     data.Timestamp,
			"status":        data.Status,
			"agent_version": data.AgentVersion,
			"uptime":        data.Uptime,
			"system_health": data.SystemHealth,
		},
		Endpoint:   "/heartbeat",
		Method:     "POST",
		MaxRetries: 3,
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	}
}

// CreateInventoryMessage creates an inventory message for the queue
func CreateInventoryMessage(data InventoryMessage) QueuedMessage {
	return QueuedMessage{
		Type:     "inventory",
		Priority: 8, // High priority
		Data: map[string]interface{}{
			"machine_id": data.MachineID,
			"timestamp":  data.Timestamp,
			"data":       data.Data,
			"checksum":   data.Checksum,
		},
		Endpoint:   "/inventory",
		Method:     "POST",
		MaxRetries: 5,
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}
}

// CreateCommandResultMessage creates a command result message for the queue
func CreateCommandResultMessage(result CommandResult) QueuedMessage {
	return QueuedMessage{
		Type:     "command_result",
		Priority: 9, // Very high priority
		Data: map[string]interface{}{
			"id":             result.ID,
			"command_id":     result.CommandID,
			"status":         result.Status,
			"output":         result.Output,
			"error":          result.Error,
			"exit_code":      result.ExitCode,
			"execution_time": result.ExecutionTime,
			"timestamp":      result.Timestamp,
		},
		Endpoint:   "/commands/result",
		Method:     "POST",
		MaxRetries: 3,
		ExpiresAt:  time.Now().Add(30 * time.Minute),
	}
}
