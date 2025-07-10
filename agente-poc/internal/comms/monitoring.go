package comms

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"agente-poc/internal/logging"
)

// Monitor manages communication monitoring and health checks
type Monitor struct {
	logger      logging.Logger
	metrics     *MonitorMetrics
	healthCheck *HealthCheck
	alertRules  []AlertRule
	alertMutex  sync.RWMutex

	// Monitoring state
	running      bool
	runningMutex sync.RWMutex

	// Context and cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// MonitorMetrics comprehensive metrics for communication monitoring
type MonitorMetrics struct {
	// Connection metrics
	ConnectionUptime   time.Duration
	TotalConnections   int64
	FailedConnections  int64
	ReconnectAttempts  int64
	CurrentConnections int64

	// Request metrics
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TimeoutRequests    int64
	RetryRequests      int64

	// Response metrics
	AverageResponseTime time.Duration
	MinResponseTime     time.Duration
	MaxResponseTime     time.Duration
	ResponseTimes       []time.Duration

	// Error metrics
	TotalErrors          int64
	NetworkErrors        int64
	AuthenticationErrors int64
	ServerErrors         int64
	ClientErrors         int64

	// Data metrics
	TotalBytesSent     int64
	TotalBytesReceived int64
	MessagesPerSecond  float64
	DataTransferRate   float64

	// Queue metrics
	QueueSize        int64
	QueueUtilization float64
	AverageQueueTime time.Duration

	// Performance metrics
	CPUUsage       float64
	MemoryUsage    float64
	GoroutineCount int64

	// Timestamps
	LastUpdated           time.Time
	LastError             time.Time
	LastSuccessfulRequest time.Time
}

// HealthCheck represents system health status
type HealthCheck struct {
	Status          string                     `json:"status"`
	Timestamp       time.Time                  `json:"timestamp"`
	Components      map[string]ComponentHealth `json:"components"`
	OverallHealth   float64                    `json:"overall_health"`
	Issues          []HealthIssue              `json:"issues"`
	Recommendations []string                   `json:"recommendations"`
}

// ComponentHealth represents individual component health
type ComponentHealth struct {
	Status       string        `json:"status"`
	LastCheck    time.Time     `json:"last_check"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorRate    float64       `json:"error_rate"`
	Uptime       time.Duration `json:"uptime"`
	Message      string        `json:"message"`
}

// HealthIssue represents a health issue
type HealthIssue struct {
	Severity    string    `json:"severity"`
	Component   string    `json:"component"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Count       int64     `json:"count"`
}

// AlertRule defines monitoring alert rules
type AlertRule struct {
	ID            string
	Name          string
	Condition     string
	Threshold     float64
	Duration      time.Duration
	Severity      string
	Enabled       bool
	LastTriggered time.Time
	TriggerCount  int64
}

// MonitorConfig configuration for monitoring
type MonitorConfig struct {
	Logger          logging.Logger
	MetricsInterval time.Duration
	HealthInterval  time.Duration
	AlertRules      []AlertRule
}

// NewMonitor creates a new communication monitor
func NewMonitor(config MonitorConfig) *Monitor {
	if config.MetricsInterval == 0 {
		config.MetricsInterval = 30 * time.Second
	}
	if config.HealthInterval == 0 {
		config.HealthInterval = 60 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Monitor{
		logger:  config.Logger,
		metrics: &MonitorMetrics{},
		healthCheck: &HealthCheck{
			Status:     "unknown",
			Components: make(map[string]ComponentHealth),
			Issues:     make([]HealthIssue, 0),
		},
		alertRules: config.AlertRules,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the monitoring system
func (m *Monitor) Start() error {
	m.runningMutex.Lock()
	defer m.runningMutex.Unlock()

	if m.running {
		return fmt.Errorf("monitor already running")
	}

	m.logger.Info("Starting communication monitor")
	m.running = true

	// Start metrics collection
	go m.collectMetrics()

	// Start health checks
	go m.performHealthChecks()

	// Start alert monitoring
	go m.monitorAlerts()

	return nil
}

// Stop stops the monitoring system
func (m *Monitor) Stop() error {
	m.runningMutex.Lock()
	defer m.runningMutex.Unlock()

	if !m.running {
		return nil
	}

	m.logger.Info("Stopping communication monitor")
	m.running = false
	m.cancel()

	return nil
}

// collectMetrics collects and updates metrics
func (m *Monitor) collectMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.updateMetrics()
		}
	}
}

// updateMetrics updates the current metrics
func (m *Monitor) updateMetrics() {
	m.metrics.LastUpdated = time.Now()

	// Calculate derived metrics
	if m.metrics.TotalRequests > 0 {
		m.metrics.MessagesPerSecond = float64(m.metrics.TotalRequests) / time.Since(m.metrics.LastUpdated).Seconds()
	}

	if len(m.metrics.ResponseTimes) > 0 {
		var total time.Duration
		for _, rt := range m.metrics.ResponseTimes {
			total += rt
		}
		m.metrics.AverageResponseTime = total / time.Duration(len(m.metrics.ResponseTimes))
	}

	m.logger.Debug("Metrics updated: %d requests, %f msg/s", m.metrics.TotalRequests, m.metrics.MessagesPerSecond)
}

// performHealthChecks performs comprehensive health checks
func (m *Monitor) performHealthChecks() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkHealth()
		}
	}
}

// checkHealth performs health checks on all components
func (m *Monitor) checkHealth() {
	m.healthCheck.Timestamp = time.Now()
	m.healthCheck.Issues = m.healthCheck.Issues[:0]
	m.healthCheck.Recommendations = m.healthCheck.Recommendations[:0]

	// Check HTTP client health
	m.checkHTTPHealth()

	// Check WebSocket health
	m.checkWebSocketHealth()

	// Check queue health
	m.checkQueueHealth()

	// Check system resources
	m.checkSystemResources()

	// Calculate overall health
	m.calculateOverallHealth()

	// Generate recommendations
	m.generateRecommendations()

	m.logger.Debug("Health check completed: %s (%.1f%%)", m.healthCheck.Status, m.healthCheck.OverallHealth*100)
}

// checkHTTPHealth checks HTTP client health
func (m *Monitor) checkHTTPHealth() {
	health := ComponentHealth{
		Status:    "healthy",
		LastCheck: time.Now(),
		Uptime:    time.Since(m.metrics.LastUpdated),
	}

	// Check error rate
	if m.metrics.TotalRequests > 0 {
		errorRate := float64(m.metrics.FailedRequests) / float64(m.metrics.TotalRequests)
		health.ErrorRate = errorRate

		if errorRate > 0.1 { // 10% error rate
			health.Status = "unhealthy"
			health.Message = fmt.Sprintf("High error rate: %.2f%%", errorRate*100)
			m.addHealthIssue("warning", "http", health.Message)
		} else if errorRate > 0.05 { // 5% error rate
			health.Status = "degraded"
			health.Message = fmt.Sprintf("Elevated error rate: %.2f%%", errorRate*100)
		}
	}

	// Check response time
	if m.metrics.AverageResponseTime > 5*time.Second {
		health.Status = "degraded"
		health.Message = fmt.Sprintf("Slow response time: %v", m.metrics.AverageResponseTime)
		m.addHealthIssue("warning", "http", health.Message)
	}

	m.healthCheck.Components["http"] = health
}

// checkWebSocketHealth checks WebSocket health
func (m *Monitor) checkWebSocketHealth() {
	health := ComponentHealth{
		Status:    "healthy",
		LastCheck: time.Now(),
		Uptime:    time.Since(m.metrics.LastUpdated),
	}

	// Check connection status
	if m.metrics.CurrentConnections == 0 {
		health.Status = "unhealthy"
		health.Message = "No active connections"
		m.addHealthIssue("critical", "websocket", health.Message)
	}

	// Check reconnection attempts
	if m.metrics.ReconnectAttempts > 10 {
		health.Status = "degraded"
		health.Message = fmt.Sprintf("High reconnection attempts: %d", m.metrics.ReconnectAttempts)
		m.addHealthIssue("warning", "websocket", health.Message)
	}

	m.healthCheck.Components["websocket"] = health
}

// checkQueueHealth checks message queue health
func (m *Monitor) checkQueueHealth() {
	health := ComponentHealth{
		Status:    "healthy",
		LastCheck: time.Now(),
		Uptime:    time.Since(m.metrics.LastUpdated),
	}

	// Check queue utilization
	if m.metrics.QueueUtilization > 0.9 {
		health.Status = "unhealthy"
		health.Message = fmt.Sprintf("Queue nearly full: %.1f%%", m.metrics.QueueUtilization*100)
		m.addHealthIssue("critical", "queue", health.Message)
	} else if m.metrics.QueueUtilization > 0.7 {
		health.Status = "degraded"
		health.Message = fmt.Sprintf("Queue utilization high: %.1f%%", m.metrics.QueueUtilization*100)
		m.addHealthIssue("warning", "queue", health.Message)
	}

	m.healthCheck.Components["queue"] = health
}

// checkSystemResources checks system resource usage
func (m *Monitor) checkSystemResources() {
	health := ComponentHealth{
		Status:    "healthy",
		LastCheck: time.Now(),
		Uptime:    time.Since(m.metrics.LastUpdated),
	}

	// Check memory usage
	if m.metrics.MemoryUsage > 0.9 {
		health.Status = "unhealthy"
		health.Message = fmt.Sprintf("High memory usage: %.1f%%", m.metrics.MemoryUsage*100)
		m.addHealthIssue("critical", "system", health.Message)
	} else if m.metrics.MemoryUsage > 0.7 {
		health.Status = "degraded"
		health.Message = fmt.Sprintf("Elevated memory usage: %.1f%%", m.metrics.MemoryUsage*100)
	}

	// Check CPU usage
	if m.metrics.CPUUsage > 0.8 {
		health.Status = "unhealthy"
		health.Message = fmt.Sprintf("High CPU usage: %.1f%%", m.metrics.CPUUsage*100)
		m.addHealthIssue("warning", "system", health.Message)
	}

	m.healthCheck.Components["system"] = health
}

// calculateOverallHealth calculates overall system health
func (m *Monitor) calculateOverallHealth() {
	totalWeight := 0.0
	weightedHealth := 0.0

	for component, health := range m.healthCheck.Components {
		weight := 1.0
		if component == "http" || component == "websocket" {
			weight = 2.0 // Critical components
		}

		var healthScore float64
		switch health.Status {
		case "healthy":
			healthScore = 1.0
		case "degraded":
			healthScore = 0.7
		case "unhealthy":
			healthScore = 0.3
		default:
			healthScore = 0.0
		}

		totalWeight += weight
		weightedHealth += healthScore * weight
	}

	if totalWeight > 0 {
		m.healthCheck.OverallHealth = weightedHealth / totalWeight
	} else {
		m.healthCheck.OverallHealth = 0.0
	}

	// Set overall status
	if m.healthCheck.OverallHealth >= 0.8 {
		m.healthCheck.Status = "healthy"
	} else if m.healthCheck.OverallHealth >= 0.5 {
		m.healthCheck.Status = "degraded"
	} else {
		m.healthCheck.Status = "unhealthy"
	}
}

// generateRecommendations generates health recommendations
func (m *Monitor) generateRecommendations() {
	if m.metrics.FailedRequests > 0 {
		m.healthCheck.Recommendations = append(m.healthCheck.Recommendations, "Review network connectivity and backend availability")
	}

	if m.metrics.AverageResponseTime > 3*time.Second {
		m.healthCheck.Recommendations = append(m.healthCheck.Recommendations, "Consider optimizing request timeout settings")
	}

	if m.metrics.QueueUtilization > 0.5 {
		m.healthCheck.Recommendations = append(m.healthCheck.Recommendations, "Monitor queue size and consider increasing processing capacity")
	}

	if m.metrics.ReconnectAttempts > 5 {
		m.healthCheck.Recommendations = append(m.healthCheck.Recommendations, "Investigate WebSocket connection stability")
	}
}

// addHealthIssue adds a health issue
func (m *Monitor) addHealthIssue(severity, component, description string) {
	issue := HealthIssue{
		Severity:    severity,
		Component:   component,
		Description: description,
		Timestamp:   time.Now(),
		Count:       1,
	}

	// Check if issue already exists
	for i, existingIssue := range m.healthCheck.Issues {
		if existingIssue.Component == component && existingIssue.Description == description {
			m.healthCheck.Issues[i].Count++
			m.healthCheck.Issues[i].Timestamp = time.Now()
			return
		}
	}

	m.healthCheck.Issues = append(m.healthCheck.Issues, issue)
}

// monitorAlerts monitors and triggers alerts
func (m *Monitor) monitorAlerts() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkAlerts()
		}
	}
}

// checkAlerts checks all alert rules
func (m *Monitor) checkAlerts() {
	m.alertMutex.Lock()
	defer m.alertMutex.Unlock()

	for i, rule := range m.alertRules {
		if !rule.Enabled {
			continue
		}

		triggered := m.evaluateAlertRule(rule)
		if triggered {
			m.alertRules[i].LastTriggered = time.Now()
			m.alertRules[i].TriggerCount++
			m.triggerAlert(rule)
		}
	}
}

// evaluateAlertRule evaluates an alert rule
func (m *Monitor) evaluateAlertRule(rule AlertRule) bool {
	switch rule.Condition {
	case "error_rate":
		if m.metrics.TotalRequests > 0 {
			errorRate := float64(m.metrics.FailedRequests) / float64(m.metrics.TotalRequests)
			return errorRate > rule.Threshold
		}
	case "response_time":
		return m.metrics.AverageResponseTime.Seconds() > rule.Threshold
	case "queue_utilization":
		return m.metrics.QueueUtilization > rule.Threshold
	case "memory_usage":
		return m.metrics.MemoryUsage > rule.Threshold
	case "cpu_usage":
		return m.metrics.CPUUsage > rule.Threshold
	}

	return false
}

// triggerAlert triggers an alert
func (m *Monitor) triggerAlert(rule AlertRule) {
	m.logger.Warning("Alert triggered: %s (%s)", rule.Name, rule.Severity)

	// In a real implementation, this would send notifications
	// via email, Slack, PagerDuty, etc.
}

// RecordRequest records a request for metrics
func (m *Monitor) RecordRequest(duration time.Duration, success bool) {
	m.metrics.TotalRequests++
	m.metrics.LastSuccessfulRequest = time.Now()

	if success {
		m.metrics.SuccessfulRequests++
	} else {
		m.metrics.FailedRequests++
		m.metrics.LastError = time.Now()
	}

	// Record response time
	m.metrics.ResponseTimes = append(m.metrics.ResponseTimes, duration)
	if len(m.metrics.ResponseTimes) > 100 {
		m.metrics.ResponseTimes = m.metrics.ResponseTimes[1:]
	}

	// Update min/max response times
	if m.metrics.MinResponseTime == 0 || duration < m.metrics.MinResponseTime {
		m.metrics.MinResponseTime = duration
	}
	if duration > m.metrics.MaxResponseTime {
		m.metrics.MaxResponseTime = duration
	}
}

// RecordError records an error for metrics
func (m *Monitor) RecordError(errorType string) {
	m.metrics.TotalErrors++
	m.metrics.LastError = time.Now()

	switch errorType {
	case "network":
		m.metrics.NetworkErrors++
	case "authentication":
		m.metrics.AuthenticationErrors++
	case "server":
		m.metrics.ServerErrors++
	case "client":
		m.metrics.ClientErrors++
	}
}

// RecordConnection records connection metrics
func (m *Monitor) RecordConnection(success bool) {
	m.metrics.TotalConnections++
	if success {
		m.metrics.CurrentConnections++
	} else {
		m.metrics.FailedConnections++
	}
}

// RecordDisconnection records disconnection
func (m *Monitor) RecordDisconnection() {
	if m.metrics.CurrentConnections > 0 {
		m.metrics.CurrentConnections--
	}
}

// RecordReconnect records reconnection attempt
func (m *Monitor) RecordReconnect() {
	m.metrics.ReconnectAttempts++
}

// RecordDataTransfer records data transfer metrics
func (m *Monitor) RecordDataTransfer(sent, received int64) {
	m.metrics.TotalBytesSent += sent
	m.metrics.TotalBytesReceived += received
}

// GetMetrics returns current metrics
func (m *Monitor) GetMetrics() MonitorMetrics {
	return *m.metrics
}

// GetHealthCheck returns current health check
func (m *Monitor) GetHealthCheck() HealthCheck {
	return *m.healthCheck
}

// GetMetricsJSON returns metrics as JSON
func (m *Monitor) GetMetricsJSON() ([]byte, error) {
	return json.MarshalIndent(m.metrics, "", "  ")
}

// GetHealthJSON returns health check as JSON
func (m *Monitor) GetHealthJSON() ([]byte, error) {
	return json.MarshalIndent(m.healthCheck, "", "  ")
}

// IsHealthy returns if the system is healthy
func (m *Monitor) IsHealthy() bool {
	return m.healthCheck.Status == "healthy"
}

// GetOverallHealth returns overall health score
func (m *Monitor) GetOverallHealth() float64 {
	return m.healthCheck.OverallHealth
}
