package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Config representa a configuração do agente
type Config struct {
	MachineID          string        `json:"machine_id"`
	BackendURL         string        `json:"backend_url"`
	WebSocketURL       string        `json:"websocket_url"`
	Token              string        `json:"token"`
	HeartbeatInterval  time.Duration `json:"heartbeat_interval"`
	CollectionInterval time.Duration `json:"collection_interval"`
	InventoryInterval  time.Duration `json:"inventory_interval"`
	CommandTimeout     time.Duration `json:"command_timeout"`
	RetryInterval      time.Duration `json:"retry_interval"`
	ReconnectInterval  time.Duration `json:"reconnect_interval"`
	MaxRetries         int           `json:"max_retries"`
	LogLevel           string        `json:"log_level"`
	Debug              bool          `json:"debug"`
}

// configJSON é usado para deserialização JSON com segundos
type configJSON struct {
	MachineID          string `json:"machine_id"`
	BackendURL         string `json:"backend_url"`
	WebSocketURL       string `json:"websocket_url"`
	Token              string `json:"token"`
	HeartbeatInterval  int    `json:"heartbeat_interval"`
	CollectionInterval int    `json:"collection_interval"`
	InventoryInterval  int    `json:"inventory_interval"`
	CommandTimeout     int    `json:"command_timeout"`
	RetryInterval      int    `json:"retry_interval"`
	ReconnectInterval  int    `json:"reconnect_interval"`
	MaxRetries         int    `json:"max_retries"`
	LogLevel           string `json:"log_level"`
	Debug              bool   `json:"debug"`
}

// LoadConfig carrega a configuração de um arquivo JSON
func LoadConfig(path string) (*Config, error) {
	// Ler arquivo de configuração
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo de configuração %s: %w", path, err)
	}

	// Deserializar JSON em struct temporária
	var tempConfig configJSON
	if err := json.Unmarshal(data, &tempConfig); err != nil {
		return nil, fmt.Errorf("erro ao deserializar configuração: %w", err)
	}

	// Converter para Config com time.Duration
	config := Config{
		MachineID:          tempConfig.MachineID,
		BackendURL:         tempConfig.BackendURL,
		WebSocketURL:       tempConfig.WebSocketURL,
		Token:              tempConfig.Token,
		HeartbeatInterval:  time.Duration(tempConfig.HeartbeatInterval) * time.Second,
		CollectionInterval: time.Duration(tempConfig.CollectionInterval) * time.Second,
		InventoryInterval:  time.Duration(tempConfig.InventoryInterval) * time.Second,
		CommandTimeout:     time.Duration(tempConfig.CommandTimeout) * time.Second,
		RetryInterval:      time.Duration(tempConfig.RetryInterval) * time.Second,
		ReconnectInterval:  time.Duration(tempConfig.ReconnectInterval) * time.Second,
		MaxRetries:         tempConfig.MaxRetries,
		LogLevel:           tempConfig.LogLevel,
		Debug:              tempConfig.Debug,
	}

	// Validar configuração
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuração inválida: %w", err)
	}

	// Aplicar valores padrão
	config.ApplyDefaults()

	return &config, nil
}

// Validate valida os campos obrigatórios da configuração
func (c *Config) Validate() error {
	var errors []string

	if c.MachineID == "" {
		errors = append(errors, "machine_id é obrigatório")
	}

	if c.BackendURL == "" {
		errors = append(errors, "backend_url é obrigatório")
	}

	if c.WebSocketURL == "" {
		errors = append(errors, "websocket_url é obrigatório")
	}

	if c.Token == "" {
		errors = append(errors, "token é obrigatório")
	}

	if c.HeartbeatInterval <= 0 {
		errors = append(errors, "heartbeat_interval deve ser maior que 0")
	}

	if len(errors) > 0 {
		return fmt.Errorf("erros de validação: %s", strings.Join(errors, ", "))
	}

	return nil
}

// ApplyDefaults aplica valores padrão para campos opcionais
func (c *Config) ApplyDefaults() {
	if c.CollectionInterval <= 0 {
		c.CollectionInterval = 60 * time.Second // 1 minuto
	}

	if c.InventoryInterval <= 0 {
		c.InventoryInterval = 300 * time.Second // 5 minutos
	}

	if c.CommandTimeout <= 0 {
		c.CommandTimeout = 30 * time.Second // 30 segundos
	}

	if c.RetryInterval <= 0 {
		c.RetryInterval = 5 * time.Second // 5 segundos
	}

	if c.ReconnectInterval <= 0 {
		c.ReconnectInterval = 5 * time.Second // 5 segundos
	}

	if c.MaxRetries <= 0 {
		c.MaxRetries = 3
	}

	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
}

// String retorna uma representação string da configuração (sem token)
func (c *Config) String() string {
	safeConfig := *c
	safeConfig.Token = "***" // Ocultar token nos logs

	data, _ := json.MarshalIndent(safeConfig, "", "  ")
	return string(data)
}
