package communications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"machine-monitor-agent/internal/types"
)

// HTTPClient cliente HTTP para comunicação com o backend
type HTTPClient struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

// NewHTTPClient cria um novo cliente HTTP
func NewHTTPClient(baseURL, apiKey string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// RegisterMachine registra a máquina no backend
func (h *HTTPClient) RegisterMachine(ctx context.Context, machineID string, inventory *types.Inventory) error {
	url := fmt.Sprintf("%s/api/agentes/%s", h.baseURL, machineID)

	payload := map[string]interface{}{
		"machine_id": machineID,
		"inventory":  inventory,
	}

	return h.makeRequest(ctx, "POST", url, payload, nil)
}

// SendHeartbeat envia heartbeat para o backend
func (h *HTTPClient) SendHeartbeat(ctx context.Context, heartbeat *types.HeartbeatData) error {
	url := fmt.Sprintf("%s/api/agentes/%s/heartbeat", h.baseURL, heartbeat.MachineID)
	return h.makeRequest(ctx, "POST", url, heartbeat, nil)
}

// SendInventory envia inventário para o backend
func (h *HTTPClient) SendInventory(ctx context.Context, inventory *types.Inventory) error {
	url := fmt.Sprintf("%s/api/agentes/%s/inventory", h.baseURL, inventory.MachineID)
	return h.makeRequest(ctx, "POST", url, inventory, nil)
}

// SendCommandResult envia resultado de comando para o backend
func (h *HTTPClient) SendCommandResult(ctx context.Context, machineID string, result *types.CommandResult) error {
	url := fmt.Sprintf("%s/api/agentes/%s/commands/%s/result", h.baseURL, machineID, result.ID)
	return h.makeRequest(ctx, "POST", url, result, nil)
}

// GetCommands obtém comandos pendentes do backend
func (h *HTTPClient) GetCommands(ctx context.Context, machineID string) ([]types.Command, error) {
	url := fmt.Sprintf("%s/api/agentes/%s/commands", h.baseURL, machineID)

	var commands []types.Command
	err := h.makeRequest(ctx, "GET", url, nil, &commands)
	if err != nil {
		return nil, err
	}

	return commands, nil
}

// makeRequest faz uma requisição HTTP
func (h *HTTPClient) makeRequest(ctx context.Context, method, url string, payload interface{}, result interface{}) error {
	var body io.Reader

	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("erro ao serializar payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("erro ao criar requisição: %w", err)
	}

	// Adiciona headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Machine-Monitor-Agent/1.0.0")

	if h.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.apiKey)
	}

	// Faz a requisição
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao fazer requisição: %w", err)
	}
	defer resp.Body.Close()

	// Lê o corpo da resposta
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("erro ao ler resposta: %w", err)
	}

	// Verifica status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("erro HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Deserializa resultado se fornecido
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("erro ao deserializar resposta: %w", err)
		}
	}

	return nil
}

// Ping testa conectividade com o backend
func (h *HTTPClient) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/ping", h.baseURL)
	return h.makeRequest(ctx, "GET", url, nil, nil)
}
