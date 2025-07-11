package communications

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"machine-monitor-agent/internal/types"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// WSClient cliente WebSocket para comunicação em tempo real
type WSClient struct {
	conn      *websocket.Conn
	url       string
	apiKey    string
	machineID string
	mu        sync.RWMutex
	connected bool
	reconnect bool

	// Canais para comunicação
	commandChan chan types.Command
	resultChan  chan types.CommandResult
	closeChan   chan struct{}

	// Configurações
	reconnectInterval time.Duration
	pingInterval      time.Duration
	writeTimeout      time.Duration
	readTimeout       time.Duration
}

// NewWSClient cria um novo cliente WebSocket
func NewWSClient(baseURL, apiKey, machineID string) *WSClient {
	// Converte HTTP URL para WebSocket URL
	wsURL := baseURL
	if baseURL[:4] == "http" {
		wsURL = "ws" + baseURL[4:] + "/agent-ws"
	}

	return &WSClient{
		url:               wsURL,
		apiKey:            apiKey,
		machineID:         machineID,
		commandChan:       make(chan types.Command, 100),
		resultChan:        make(chan types.CommandResult, 100),
		closeChan:         make(chan struct{}),
		reconnectInterval: 5 * time.Second,
		pingInterval:      30 * time.Second,
		writeTimeout:      10 * time.Second,
		readTimeout:       60 * time.Second,
	}
}

// Connect conecta ao WebSocket
func (w *WSClient) Connect(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.connected {
		return nil
	}

	// Prepara headers
	headers := make(map[string][]string)
	if w.apiKey != "" {
		headers["Authorization"] = []string{"Bearer " + w.apiKey}
	}

	// Conecta ao WebSocket
	u, err := url.Parse(w.url)
	if err != nil {
		return fmt.Errorf("erro ao fazer parse da URL: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), headers)
	if err != nil {
		return fmt.Errorf("erro ao conectar WebSocket: %w", err)
	}

	w.conn = conn
	w.connected = true
	w.reconnect = true

	// Inicia goroutines para leitura e escrita
	go w.readLoop()
	go w.writeLoop()
	go w.pingLoop()

	log.Info().Str("url", w.url).Msg("WebSocket conectado")
	return nil
}

// Disconnect desconecta do WebSocket
func (w *WSClient) Disconnect() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.connected {
		return nil
	}

	w.reconnect = false
	w.connected = false

	// Fecha canais
	close(w.closeChan)

	// Fecha conexão
	if w.conn != nil {
		w.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		w.conn.Close()
	}

	log.Info().Msg("WebSocket desconectado")
	return nil
}

// IsConnected verifica se está conectado
func (w *WSClient) IsConnected() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.connected
}

// GetCommandChannel retorna o canal de comandos
func (w *WSClient) GetCommandChannel() <-chan types.Command {
	return w.commandChan
}

// SendResult envia resultado de comando
func (w *WSClient) SendResult(result types.CommandResult) error {
	select {
	case w.resultChan <- result:
		return nil
	default:
		return fmt.Errorf("canal de resultados cheio")
	}
}

// readLoop loop de leitura do WebSocket
func (w *WSClient) readLoop() {
	defer func() {
		w.mu.Lock()
		w.connected = false
		w.mu.Unlock()

		if w.reconnect {
			go w.reconnectLoop()
		}
	}()

	for {
		select {
		case <-w.closeChan:
			return
		default:
		}

		// Define timeout de leitura
		w.conn.SetReadDeadline(time.Now().Add(w.readTimeout))

		messageType, data, err := w.conn.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("Erro ao ler mensagem WebSocket")
			return
		}

		if messageType == websocket.TextMessage {
			w.handleMessage(data)
		}
	}
}

// writeLoop loop de escrita do WebSocket
func (w *WSClient) writeLoop() {
	ticker := time.NewTicker(w.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.closeChan:
			return
		case result := <-w.resultChan:
			w.sendMessage("command_result", result)
		case <-ticker.C:
			// Ping é enviado pelo pingLoop
		}
	}
}

// pingLoop loop de ping do WebSocket
func (w *WSClient) pingLoop() {
	ticker := time.NewTicker(w.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.closeChan:
			return
		case <-ticker.C:
			if err := w.ping(); err != nil {
				log.Error().Err(err).Msg("Erro ao enviar ping")
				return
			}
		}
	}
}

// ping envia ping para o servidor
func (w *WSClient) ping() error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.connected || w.conn == nil {
		return fmt.Errorf("WebSocket não conectado")
	}

	w.conn.SetWriteDeadline(time.Now().Add(w.writeTimeout))
	return w.conn.WriteMessage(websocket.PingMessage, nil)
}

// handleMessage trata mensagens recebidas
func (w *WSClient) handleMessage(data []byte) {
	var msg struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		log.Error().Err(err).Msg("Erro ao deserializar mensagem")
		return
	}

	switch msg.Type {
	case "command":
		var command types.Command
		if err := json.Unmarshal(msg.Data, &command); err != nil {
			log.Error().Err(err).Msg("Erro ao deserializar comando")
			return
		}

		select {
		case w.commandChan <- command:
			log.Info().Str("command_id", command.ID).Str("type", command.Type).Msg("Comando recebido")
		default:
			log.Warn().Str("command_id", command.ID).Msg("Canal de comandos cheio, comando ignorado")
		}

	case "ping":
		w.sendMessage("pong", map[string]interface{}{
			"machine_id": w.machineID,
			"timestamp":  time.Now(),
		})

	default:
		log.Warn().Str("type", msg.Type).Msg("Tipo de mensagem desconhecido")
	}
}

// sendMessage envia mensagem para o servidor
func (w *WSClient) sendMessage(msgType string, data interface{}) error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.connected || w.conn == nil {
		return fmt.Errorf("WebSocket não conectado")
	}

	msg := map[string]interface{}{
		"type": msgType,
		"data": data,
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("erro ao serializar mensagem: %w", err)
	}

	w.conn.SetWriteDeadline(time.Now().Add(w.writeTimeout))
	return w.conn.WriteMessage(websocket.TextMessage, jsonData)
}

// reconnectLoop loop de reconexão
func (w *WSClient) reconnectLoop() {
	for w.reconnect {
		log.Info().Msg("Tentando reconectar WebSocket...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := w.Connect(ctx); err != nil {
			log.Error().Err(err).Msg("Erro ao reconectar WebSocket")
			cancel()
			time.Sleep(w.reconnectInterval)
			continue
		}
		cancel()

		break
	}
}
