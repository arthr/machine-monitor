package comms

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"agente-poc/internal/collector"
	"agente-poc/internal/logging"

	"github.com/gorilla/websocket"
)

// Config contém a configuração do communications manager
type Config struct {
	BackendURL        string
	WebSocketURL      string
	Token             string
	MachineID         string
	RetryInterval     time.Duration
	HeartbeatInterval time.Duration
	Logger            logging.Logger

	// HTTP configuration
	HTTPTimeout    time.Duration
	HTTPMaxRetries int
	HTTPRetryDelay time.Duration
	TLSSkipVerify  bool

	// WebSocket configuration
	WSReconnectDelay time.Duration
	WSMaxReconnects  int
	WSPingInterval   time.Duration
	WSPongTimeout    time.Duration
	WSMaxQueueSize   int
}

// Manager gerencia as comunicações com o backend
type Manager struct {
	config     *Config
	logger     logging.Logger
	httpClient *HTTPClient
	wsClient   *WebSocketClient

	// State management
	running      bool
	runningMutex sync.RWMutex

	// Context and cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Metrics
	metrics *ManagerMetrics

	// Channels
	commandChan chan Command
	resultChan  chan CommandResult

	// Heartbeat control
	lastHeartbeat  time.Time
	heartbeatMutex sync.RWMutex
}

// ManagerMetrics tracks manager-level metrics
type ManagerMetrics struct {
	StartTime         time.Time
	TotalUptime       time.Duration
	HeartbeatsSent    int64
	InventoriesSent   int64
	CommandsReceived  int64
	ResultsSent       int64
	HTTPRequests      int64
	WSMessages        int64
	Errors            int64
	LastError         string
	LastErrorTime     time.Time
	ConnectionStatus  string
	LastInventoryTime time.Time
}

// New cria uma nova instância do communications manager
func New(config *Config) (*Manager, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	// Set defaults
	if config.HTTPTimeout == 0 {
		config.HTTPTimeout = 30 * time.Second
	}
	if config.HTTPMaxRetries == 0 {
		config.HTTPMaxRetries = 3
	}
	if config.HTTPRetryDelay == 0 {
		config.HTTPRetryDelay = 1 * time.Second
	}
	if config.WSReconnectDelay == 0 {
		config.WSReconnectDelay = 5 * time.Second
	}
	if config.WSMaxReconnects == 0 {
		config.WSMaxReconnects = 10
	}
	if config.WSPingInterval == 0 {
		config.WSPingInterval = 30 * time.Second
	}
	if config.WSPongTimeout == 0 {
		config.WSPongTimeout = 10 * time.Second
	}
	if config.WSMaxQueueSize == 0 {
		config.WSMaxQueueSize = 1000
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 30 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create HTTP client
	httpClient := NewHTTPClient(HTTPConfig{
		BaseURL:         config.BackendURL,
		Token:           config.Token,
		UserAgent:       "MacOS-Agent/1.0.0",
		Timeout:         config.HTTPTimeout,
		MaxRetries:      config.HTTPMaxRetries,
		RetryDelay:      config.HTTPRetryDelay,
		TLSSkipVerify:   config.TLSSkipVerify,
		ConnectTimeout:  10 * time.Second,
		IdleTimeout:     90 * time.Second,
		MaxIdleConns:    10,
		MaxConnsPerHost: 10,
		Logger:          config.Logger,
	})

	// Create WebSocket client
	wsClient := NewWebSocketClient(WebSocketConfig{
		URL:            config.WebSocketURL,
		Token:          config.Token,
		ReconnectDelay: config.WSReconnectDelay,
		MaxReconnects:  config.WSMaxReconnects,
		PingInterval:   config.WSPingInterval,
		PongTimeout:    config.WSPongTimeout,
		MaxQueueSize:   config.WSMaxQueueSize,
		Logger:         config.Logger,
	})

	manager := &Manager{
		config:     config,
		logger:     config.Logger,
		httpClient: httpClient,
		wsClient:   wsClient,
		ctx:        ctx,
		cancel:     cancel,
		metrics: &ManagerMetrics{
			StartTime:        time.Now(),
			ConnectionStatus: "disconnected",
		},
		commandChan: make(chan Command, 100),
		resultChan:  make(chan CommandResult, 100),
	}

	return manager, nil
}

// Start inicia o communications manager
func (m *Manager) Start(ctx context.Context) error {
	m.runningMutex.Lock()
	defer m.runningMutex.Unlock()

	if m.running {
		return fmt.Errorf("manager already running")
	}

	m.logger.Info("Starting communications manager...")
	m.running = true
	m.metrics.StartTime = time.Now()

	// Start WebSocket connection
	go m.startWebSocketConnection()

	// Start heartbeat
	go m.startHeartbeat()

	// Start command processing
	go m.processCommands()

	// Start result processing
	go m.processResults()

	// Monitor context cancellation
	go func() {
		select {
		case <-ctx.Done():
			m.logger.Info("Context cancelled, stopping manager")
			_ = m.Stop()
		case <-m.ctx.Done():
			m.logger.Info("Manager context cancelled")
		}
	}()

	// Try to register machine if not already registered
	go func() {
		time.Sleep(2 * time.Second) // Wait for initial connections
		if err := m.RegisterMachine(); err != nil {
			m.logger.Error("Failed to register machine: %v", err)
		}
	}()

	m.logger.Info("Communications manager started successfully")
	return nil
}

// Stop para o communications manager
func (m *Manager) Stop() error {
	m.runningMutex.Lock()
	defer m.runningMutex.Unlock()

	if !m.running {
		return nil
	}

	m.logger.Info("Stopping communications manager...")
	m.running = false

	// Cancel context
	m.cancel()

	// Close WebSocket
	if err := m.wsClient.Close(); err != nil {
		m.logger.Error("Error closing WebSocket client: %v", err)
	}

	// Close HTTP client
	if err := m.httpClient.Close(); err != nil {
		m.logger.Error("Error closing HTTP client: %v", err)
	}

	// Close channels
	close(m.commandChan)
	close(m.resultChan)

	m.logger.Info("Communications manager stopped")
	return nil
}

// startWebSocketConnection manages WebSocket connection
func (m *Manager) startWebSocketConnection() {
	for {
		select {
		case <-m.ctx.Done():
			return
		default:
			if err := m.wsClient.Connect(); err != nil {
				m.logger.Error("Failed to connect WebSocket: %v", err)
				m.metrics.Errors++
				m.metrics.LastError = err.Error()
				m.metrics.LastErrorTime = time.Now()
				m.metrics.ConnectionStatus = "disconnected"

				time.Sleep(m.config.WSReconnectDelay)
				continue
			}

			m.metrics.ConnectionStatus = "connected"
			m.logger.Info("WebSocket connected successfully")

			// Registrar máquina no WebSocket - formato simples esperado pelo backend
			registrationData := map[string]interface{}{
				"machine_id": m.config.MachineID,
			}

			// Serializar e enviar registro
			if regBytes, err := json.Marshal(registrationData); err == nil {
				if err := m.wsClient.conn.WriteMessage(websocket.TextMessage, regBytes); err != nil {
					m.logger.Error("Failed to register WebSocket: %v", err)
				} else {
					m.logger.Info("WebSocket registration sent for machine: %s", m.config.MachineID)
				}
			}

			// Process WebSocket messages
			go m.handleWebSocketMessages()

			// Wait for disconnection
			for m.wsClient.IsConnected() {
				select {
				case <-m.ctx.Done():
					return
				case <-time.After(5 * time.Second):
					// Check connection status
				}
			}

			m.metrics.ConnectionStatus = "disconnected"
			m.logger.Warning("WebSocket disconnected")
		}
	}
}

// handleWebSocketMessages processes incoming WebSocket messages
func (m *Manager) handleWebSocketMessages() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case command := <-m.wsClient.CommandChannel():
			m.logger.Debug("Received command: %s", command.ID)
			m.metrics.CommandsReceived++

			// Forward to command channel
			select {
			case m.commandChan <- command:
			default:
				m.logger.Warning("Command channel full, dropping command")
			}
		case msg := <-m.wsClient.MessageChannel():
			m.logger.Debug("Received WebSocket message: %s", msg.Type)
			m.metrics.WSMessages++

			// Handle different message types
			switch msg.Type {
			case "ping":
				// Already handled by WebSocket client
			case "config_update":
				m.handleConfigUpdate(msg)
			case "status_request":
				m.handleStatusRequest(msg)
			default:
				m.logger.Debug("Unhandled message type: %s", msg.Type)
			}
		}
	}
}

// SendHeartbeat envia heartbeat para o backend
func (m *Manager) SendHeartbeat() error {
	m.heartbeatMutex.Lock()
	defer m.heartbeatMutex.Unlock()

	m.logger.Debug("Sending heartbeat for machine: %s", m.config.MachineID)

	// Get system health info
	healthStatus := m.getSystemHealth()

	heartbeat := map[string]interface{}{
		"machine_id":       m.config.MachineID,
		"timestamp":        time.Now(),
		"status":           "online",
		"agent_version":    "1.0.0",
		"uptime_seconds":   int64(time.Since(m.metrics.StartTime).Seconds()),
		"last_inventory":   m.metrics.LastInventoryTime,
		"system_health":    healthStatus,
		"pending_commands": len(m.commandChan),
		"active_tasks":     []string{}, // TODO: Get from task manager
	}

	// Send via HTTP
	ctx, cancel := context.WithTimeout(m.ctx, m.config.HTTPTimeout)
	defer cancel()

	if err := m.httpClient.POST(ctx, "/heartbeat", heartbeat, nil); err != nil {
		m.metrics.Errors++
		m.metrics.LastError = err.Error()
		m.metrics.LastErrorTime = time.Now()
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	m.metrics.HeartbeatsSent++
	m.metrics.HTTPRequests++
	m.lastHeartbeat = time.Now()

	m.logger.Debug("Heartbeat sent successfully")
	return nil
}

// SendInventory envia dados de inventário para o backend
func (m *Manager) SendInventory(data *collector.InventoryData) error {
	m.logger.WithField("machine_id", data.MachineID).Debug("Sending inventory data...")

	// Calculate checksum
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal inventory data: %w", err)
	}

	hash := sha256.Sum256(dataBytes)
	checksum := hex.EncodeToString(hash[:])

	// Create inventory message in the format expected by backend
	inventoryMsg := map[string]interface{}{
		"machine_id": data.MachineID,
		"type":       "inventory",
		"timestamp":  time.Now(),
		"data":       data,
		"checksum":   checksum,
	}

	// Send via HTTP
	ctx, cancel := context.WithTimeout(m.ctx, m.config.HTTPTimeout)
	defer cancel()

	if err := m.httpClient.POST(ctx, "/inventory", inventoryMsg, nil); err != nil {
		m.metrics.Errors++
		m.metrics.LastError = err.Error()
		m.metrics.LastErrorTime = time.Now()
		return fmt.Errorf("failed to send inventory: %w", err)
	}

	m.metrics.InventoriesSent++
	m.metrics.HTTPRequests++
	m.metrics.LastInventoryTime = time.Now()

	m.logger.Debug("Inventory sent successfully")
	return nil
}

// SendCommandResult envia resultado de comando para o backend
func (m *Manager) SendCommandResult(result *CommandResult) error {
	m.logger.WithField("command_id", result.CommandID).Debug("Sending command result...")

	// Send via WebSocket if connected, otherwise HTTP
	if m.wsClient.IsConnected() {
		message := WebSocketMessage{
			Type:      "command_result",
			ID:        result.ID,
			Timestamp: time.Now(),
			Data:      result,
		}

		if err := m.wsClient.SendMessage(message); err != nil {
			m.logger.Warning("Failed to send via WebSocket, trying HTTP: %v", err)
			return m.sendResultViaHTTP(result)
		}

		m.metrics.ResultsSent++
		m.metrics.WSMessages++
	} else {
		return m.sendResultViaHTTP(result)
	}

	m.logger.Debug("Command result sent successfully")
	return nil
}

// sendResultViaHTTP sends command result via HTTP fallback
func (m *Manager) sendResultViaHTTP(result *CommandResult) error {
	ctx, cancel := context.WithTimeout(m.ctx, m.config.HTTPTimeout)
	defer cancel()

	if err := m.httpClient.POST(ctx, "/commands/result", result, nil); err != nil {
		m.metrics.Errors++
		m.metrics.LastError = err.Error()
		m.metrics.LastErrorTime = time.Now()
		return fmt.Errorf("failed to send command result via HTTP: %w", err)
	}

	m.metrics.ResultsSent++
	m.metrics.HTTPRequests++
	return nil
}

// RegisterMachine registra a máquina no backend
func (m *Manager) RegisterMachine() error {
	m.logger.WithField("machine_id", m.config.MachineID).Info("Registering machine...")

	// Create registration request
	regRequest := RegistrationRequest{
		MachineID:    m.config.MachineID,
		Token:        m.config.Token,
		AgentVersion: "1.0.0",
		Timestamp:    time.Now(),
		// TODO: Add system info and hardware info
	}

	// Send via HTTP
	ctx, cancel := context.WithTimeout(m.ctx, m.config.HTTPTimeout)
	defer cancel()

	var response RegistrationResponse
	if err := m.httpClient.POST(ctx, "/machines/register", regRequest, &response); err != nil {
		m.metrics.Errors++
		m.metrics.LastError = err.Error()
		m.metrics.LastErrorTime = time.Now()
		return fmt.Errorf("failed to register machine: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("machine registration failed: %s", response.Message)
	}

	m.metrics.HTTPRequests++
	m.logger.Info("Machine registered successfully")
	return nil
}

// CommandChannel returns the command channel
func (m *Manager) CommandChannel() <-chan Command {
	return m.commandChan
}

// SendResult sends a command result
func (m *Manager) SendResult(result *CommandResult) error {
	select {
	case m.resultChan <- *result:
		return nil
	default:
		return fmt.Errorf("result channel full")
	}
}

// startHeartbeat starts the heartbeat routine
func (m *Manager) startHeartbeat() {
	ticker := time.NewTicker(m.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if err := m.SendHeartbeat(); err != nil {
				m.logger.Error("Failed to send heartbeat: %v", err)
			}
		}
	}
}

// processCommands processes incoming commands
func (m *Manager) processCommands() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case command := <-m.commandChan:
			m.logger.Debug("Processing command: %s", command.ID)
			// Commands are forwarded to the command executor
			// This is just a passthrough for now
		}
	}
}

// processResults processes command results
func (m *Manager) processResults() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case result := <-m.resultChan:
			if err := m.SendCommandResult(&result); err != nil {
				m.logger.Error("Failed to send command result: %v", err)
			}
		}
	}
}

// getSystemHealth returns current system health status
func (m *Manager) getSystemHealth() map[string]interface{} {
	// TODO: Get real system health data from collector
	// Por agora, simular alguns dados básicos
	return map[string]interface{}{
		"cpu_usage_percent":    25.5, // Simular 25.5% CPU
		"memory_usage_percent": 68.3, // Simular 68.3% RAM
		"disk_usage_percent":   45.2, // Simular 45.2% disco
		"status":               "healthy",
	}
}

// handleConfigUpdate handles configuration updates
func (m *Manager) handleConfigUpdate(msg WebSocketMessage) {
	m.logger.Info("Received configuration update")
	// TODO: Implement configuration update
}

// handleStatusRequest handles status requests
func (m *Manager) handleStatusRequest(msg WebSocketMessage) {
	m.logger.Debug("Received status request")

	status := StatusUpdate{
		MachineID: m.config.MachineID,
		Status:    m.metrics.ConnectionStatus,
		Message:   fmt.Sprintf("Uptime: %v", time.Since(m.metrics.StartTime)),
		Timestamp: time.Now(),
	}

	response := WebSocketMessage{
		Type:      "status_response",
		ID:        msg.ID,
		Timestamp: time.Now(),
		Data:      status,
	}

	_ = m.wsClient.SendMessage(response)
}

// GetMetrics returns manager metrics
func (m *Manager) GetMetrics() ManagerMetrics {
	m.runningMutex.RLock()
	defer m.runningMutex.RUnlock()

	metrics := *m.metrics
	if m.running {
		metrics.TotalUptime = time.Since(m.metrics.StartTime)
	}

	return metrics
}

// IsRunning returns if the manager is running
func (m *Manager) IsRunning() bool {
	m.runningMutex.RLock()
	defer m.runningMutex.RUnlock()
	return m.running
}

// IsConnected returns if the manager is connected
func (m *Manager) IsConnected() bool {
	return m.wsClient.IsConnected() || m.httpClient.IsHealthy()
}
