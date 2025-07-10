package comms

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"agente-poc/internal/logging"

	"github.com/gorilla/websocket"
)

// WebSocketClient manages WebSocket connections with automatic reconnection
type WebSocketClient struct {
	url       string
	token     string
	machineID string
	conn      *websocket.Conn
	connMutex sync.RWMutex
	logger    logging.Logger

	// System health callback
	systemHealthCallback func() map[string]interface{}

	// Channels
	commandChan chan Command
	messageChan chan WebSocketMessage
	closeChan   chan struct{}

	// Connection state
	connected    bool
	reconnecting bool

	// Configuration
	reconnectDelay time.Duration
	maxReconnects  int
	pingInterval   time.Duration
	pongTimeout    time.Duration

	// Context and cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Metrics
	metrics *WebSocketMetrics

	// Message queue for offline messages
	messageQueue []WebSocketMessage
	queueMutex   sync.Mutex
	maxQueueSize int
}

// WebSocketMetrics tracks WebSocket client metrics
type WebSocketMetrics struct {
	TotalConnections   int64
	SuccessfulConnects int64
	FailedConnects     int64
	Reconnects         int64
	MessagesReceived   int64
	MessagesSent       int64
	PingsSent          int64
	PongsReceived      int64
	LastConnectTime    time.Time
	LastDisconnectTime time.Time
	TotalUptime        time.Duration
	ConnectionErrors   int64
	MessageErrors      int64
}

// WebSocketConfig configuration for WebSocket client
type WebSocketConfig struct {
	URL                  string
	Token                string
	MachineID            string
	ReconnectDelay       time.Duration
	MaxReconnects        int
	PingInterval         time.Duration
	PongTimeout          time.Duration
	MaxQueueSize         int
	Logger               logging.Logger
	SystemHealthCallback func() map[string]interface{}
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(config WebSocketConfig) *WebSocketClient {
	ctx, cancel := context.WithCancel(context.Background())

	return &WebSocketClient{
		url:                  config.URL,
		token:                config.Token,
		machineID:            config.MachineID,
		logger:               config.Logger,
		systemHealthCallback: config.SystemHealthCallback,
		commandChan:          make(chan Command, 100),
		messageChan:          make(chan WebSocketMessage, 100),
		closeChan:            make(chan struct{}),
		reconnectDelay:       config.ReconnectDelay,
		maxReconnects:        config.MaxReconnects,
		pingInterval:         config.PingInterval,
		pongTimeout:          config.PongTimeout,
		ctx:                  ctx,
		cancel:               cancel,
		metrics:              &WebSocketMetrics{},
		messageQueue:         make([]WebSocketMessage, 0),
		maxQueueSize:         config.MaxQueueSize,
	}
}

// Connect establishes WebSocket connection
func (ws *WebSocketClient) Connect() error {
	ws.connMutex.Lock()
	defer ws.connMutex.Unlock()

	if ws.connected {
		return nil
	}

	ws.logger.Info("Connecting to WebSocket server: %s", ws.url)

	// Parse URL
	u, err := url.Parse(ws.url)
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %w", err)
	}

	// Create headers
	headers := make(map[string][]string)
	if ws.token != "" {
		headers["Authorization"] = []string{"Bearer " + ws.token}
	}
	headers["User-Agent"] = []string{"MacOS-Agent/1.0.0"}

	// Establish connection
	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	conn, _, err := dialer.Dial(u.String(), headers)
	if err != nil {
		ws.metrics.FailedConnects++
		ws.metrics.ConnectionErrors++
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	ws.conn = conn
	ws.connected = true
	ws.reconnecting = false
	ws.metrics.TotalConnections++
	ws.metrics.SuccessfulConnects++
	ws.metrics.LastConnectTime = time.Now()

	ws.logger.Info("WebSocket connection established")

	// Start handlers
	go ws.handleMessages()
	go ws.handlePing()

	// Send queued messages
	go ws.sendQueuedMessages()

	return nil
}

// Disconnect closes the WebSocket connection
func (ws *WebSocketClient) Disconnect() error {
	ws.connMutex.Lock()
	defer ws.connMutex.Unlock()

	if !ws.connected {
		return nil
	}

	ws.logger.Info("Disconnecting from WebSocket server")

	// Close connection
	if ws.conn != nil {
		_ = ws.conn.Close()
		ws.conn = nil
	}

	ws.connected = false
	ws.metrics.LastDisconnectTime = time.Now()

	// Signal close
	select {
	case ws.closeChan <- struct{}{}:
	default:
	}

	return nil
}

// Close closes the WebSocket client and cleans up resources
func (ws *WebSocketClient) Close() error {
	ws.cancel()
	return ws.Disconnect()
}

// handleMessages handles incoming WebSocket messages
func (ws *WebSocketClient) handleMessages() {
	defer func() {
		if r := recover(); r != nil {
			ws.logger.Error("WebSocket message handler panic: %v", r)
		}
	}()

	for {
		select {
		case <-ws.ctx.Done():
			return
		case <-ws.closeChan:
			return
		default:
			if !ws.isConnected() {
				time.Sleep(1 * time.Second)
				continue
			}

			ws.connMutex.RLock()
			conn := ws.conn
			connected := ws.connected
			ws.connMutex.RUnlock()

			if !connected || conn == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			// Set read deadline - usar um timeout mais longo
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			// Read message
			_, messageData, err := conn.ReadMessage()
			if err != nil {
				// Verificar se é timeout ou erro de conexão
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					ws.logger.Debug("WebSocket read timeout (normal)")
					continue
				}

				ws.logger.Error("Error reading WebSocket message: %v", err)
				ws.metrics.MessageErrors++

				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					ws.logger.Warning("WebSocket connection closed unexpectedly")
					ws.handleDisconnect()
					return
				}

				// Para outros erros, também desconectar para evitar loops
				ws.logger.Warning("WebSocket read error, disconnecting")
				ws.handleDisconnect()
				return
			}

			ws.metrics.MessagesReceived++

			// Parse message
			var message WebSocketMessage
			if err := json.Unmarshal(messageData, &message); err != nil {
				ws.logger.Error("Error parsing WebSocket message: %v", err)
				ws.metrics.MessageErrors++
				continue
			}

			// Handle message based on type
			switch message.Type {
			case "command":
				ws.handleCommand(message)
			case "ping":
				ws.handlePingMessage(message)
			case "pong":
				ws.handlePongMessage(message)
			default:
				// Forward to message channel
				select {
				case ws.messageChan <- message:
				default:
					ws.logger.Warning("Message channel full, dropping message")
				}
			}
		}
	}
}

// handleCommand processes incoming commands
func (ws *WebSocketClient) handleCommand(message WebSocketMessage) {
	ws.logger.Debug("Received command: %s", message.Type)

	// Parse command data
	commandData, ok := message.Data.(map[string]interface{})
	if !ok {
		ws.logger.Error("Invalid command data format")
		return
	}

	// Convert to Command struct
	command := Command{
		ID:        message.ID,
		Type:      getString(commandData, "type"),
		Command:   getString(commandData, "command"),
		Args:      getStringSlice(commandData, "args"),
		Options:   getMap(commandData, "options"),
		Timeout:   getInt(commandData, "timeout"),
		Timestamp: time.Now(),
	}

	// Send to command channel
	select {
	case ws.commandChan <- command:
	default:
		ws.logger.Warning("Command channel full, dropping command")
	}
}

// handlePingMessage handles ping messages
func (ws *WebSocketClient) handlePingMessage(message WebSocketMessage) {
	ws.logger.Debug("Received structured ping")

	// Responder com pong estruturado incluindo dados de sistema
	pongData := map[string]interface{}{
		"machine_id":    ws.getMachineID(),
		"status":        "online",
		"agent_version": "1.0.0",
		"timestamp":     time.Now(),
		"ping_id":       message.ID,
	}

	// Se o ping contiver dados, extrair informações úteis
	if message.Data != nil {
		if pingData, ok := message.Data.(map[string]interface{}); ok {
			if pingSeq, exists := pingData["ping_seq"]; exists {
				pongData["ping_seq"] = pingSeq
			}
		}
	}

	pongMessage := WebSocketMessage{
		Type:      "pong",
		ID:        message.ID,
		Timestamp: time.Now(),
		Data:      pongData,
	}

	if err := ws.SendMessage(pongMessage); err != nil {
		ws.logger.Error("Error sending structured pong: %v", err)
	} else {
		ws.logger.Debug("Structured pong sent in response to ping")
	}
}

// handlePongMessage handles pong messages
func (ws *WebSocketClient) handlePongMessage(message WebSocketMessage) {
	ws.logger.Debug("Received structured pong")
	ws.metrics.PongsReceived++

	// Processar dados estruturados do pong se disponíveis
	if message.Data != nil {
		if pongData, ok := message.Data.(map[string]interface{}); ok {
			if machineID, exists := pongData["machine_id"]; exists {
				ws.logger.Debug("Pong received from machine: %v", machineID)
			}

			if status, exists := pongData["status"]; exists {
				ws.logger.Debug("Remote status: %v", status)
			}
		}
	}
}

// handlePing sends periodic ping messages
func (ws *WebSocketClient) handlePing() {
	ticker := time.NewTicker(ws.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ws.ctx.Done():
			return
		case <-ws.closeChan:
			return
		case <-ticker.C:
			if ws.isConnected() {
				// Criar ping estruturado com dados de sistema
				pingData := map[string]interface{}{
					"machine_id":    ws.getMachineID(),
					"status":        "online",
					"agent_version": "1.0.0",
					"timestamp":     time.Now(),
					"ping_seq":      time.Now().UnixNano(),
				}

				// Adicionar dados de sistema health se callback disponível
				if ws.systemHealthCallback != nil {
					if systemHealth := ws.systemHealthCallback(); systemHealth != nil {
						pingData["system_health"] = systemHealth
					}
				}

				pingMessage := WebSocketMessage{
					Type:      "ping",
					ID:        fmt.Sprintf("ping_%d", time.Now().UnixNano()),
					Timestamp: time.Now(),
					Data:      pingData,
				}

				if err := ws.SendMessage(pingMessage); err != nil {
					ws.logger.Error("Error sending ping: %v", err)
				} else {
					ws.logger.Debug("Structured ping sent with system data")
					ws.metrics.PingsSent++
				}
			}
		}
	}
}

// handleDisconnect handles connection loss and triggers reconnection
func (ws *WebSocketClient) handleDisconnect() {
	ws.connMutex.Lock()
	ws.connected = false
	ws.connMutex.Unlock()

	if ws.reconnecting {
		return
	}

	ws.reconnecting = true
	ws.logger.Info("Starting reconnection process")

	go func() {
		for attempt := 0; attempt < ws.maxReconnects; attempt++ {
			select {
			case <-ws.ctx.Done():
				return
			default:
				ws.logger.Info("Reconnection attempt %d/%d", attempt+1, ws.maxReconnects)

				if err := ws.Connect(); err != nil {
					ws.logger.Error("Reconnection attempt %d failed: %v", attempt+1, err)
					ws.metrics.Reconnects++

					if attempt < ws.maxReconnects-1 {
						time.Sleep(ws.reconnectDelay * time.Duration(attempt+1))
					}
				} else {
					ws.logger.Info("Reconnection successful")
					return
				}
			}
		}

		ws.logger.Error("Max reconnection attempts exceeded")
		ws.reconnecting = false
	}()
}

// SendMessage sends a message via WebSocket
func (ws *WebSocketClient) SendMessage(message WebSocketMessage) error {
	if !ws.isConnected() {
		// Queue message if not connected
		ws.queueMessage(message)
		return fmt.Errorf("not connected, message queued")
	}

	ws.connMutex.RLock()
	defer ws.connMutex.RUnlock()

	// Set write deadline
	ws.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

	// Send message
	if err := ws.conn.WriteJSON(message); err != nil {
		ws.metrics.MessageErrors++
		return fmt.Errorf("failed to send message: %w", err)
	}

	ws.metrics.MessagesSent++
	return nil
}

// queueMessage adds a message to the offline queue
func (ws *WebSocketClient) queueMessage(message WebSocketMessage) {
	ws.queueMutex.Lock()
	defer ws.queueMutex.Unlock()

	if len(ws.messageQueue) >= ws.maxQueueSize {
		// Remove oldest message
		ws.messageQueue = ws.messageQueue[1:]
	}

	ws.messageQueue = append(ws.messageQueue, message)
}

// sendQueuedMessages sends all queued messages
func (ws *WebSocketClient) sendQueuedMessages() {
	ws.queueMutex.Lock()
	defer ws.queueMutex.Unlock()

	for _, message := range ws.messageQueue {
		if err := ws.SendMessage(message); err != nil {
			ws.logger.Error("Failed to send queued message: %v", err)
		}
	}

	ws.messageQueue = ws.messageQueue[:0]
}

// CommandChannel returns the command channel
func (ws *WebSocketClient) CommandChannel() <-chan Command {
	return ws.commandChan
}

// MessageChannel returns the message channel
func (ws *WebSocketClient) MessageChannel() <-chan WebSocketMessage {
	return ws.messageChan
}

// isConnected checks if the WebSocket is connected
func (ws *WebSocketClient) isConnected() bool {
	ws.connMutex.RLock()
	defer ws.connMutex.RUnlock()
	return ws.connected
}

// IsConnected returns connection status
func (ws *WebSocketClient) IsConnected() bool {
	return ws.isConnected()
}

// GetMetrics returns WebSocket metrics
func (ws *WebSocketClient) GetMetrics() WebSocketMetrics {
	return *ws.metrics
}

// ResetMetrics resets WebSocket metrics
func (ws *WebSocketClient) ResetMetrics() {
	ws.metrics = &WebSocketMetrics{}
}

// UpdateMachineID atualiza o machine_id do WebSocket client
func (ws *WebSocketClient) UpdateMachineID(machineID string) {
	if machineID != "" && machineID != ws.machineID {
		ws.machineID = machineID
		ws.logger.Debug("WebSocket machine_id updated to: %s", machineID)
	}
}

// getMachineID returns the machine ID
func (ws *WebSocketClient) getMachineID() string {
	return ws.machineID
}

// Helper functions for parsing command data
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getStringSlice(data map[string]interface{}, key string) []string {
	if val, ok := data[key]; ok {
		if slice, ok := val.([]interface{}); ok {
			result := make([]string, len(slice))
			for i, v := range slice {
				if str, ok := v.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}
	return nil
}

func getMap(data map[string]interface{}, key string) map[string]interface{} {
	if val, ok := data[key]; ok {
		if m, ok := val.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

func getInt(data map[string]interface{}, key string) int {
	if val, ok := data[key]; ok {
		if i, ok := val.(float64); ok {
			return int(i)
		}
		if i, ok := val.(int); ok {
			return i
		}
	}
	return 0
}
