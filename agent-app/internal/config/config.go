package config

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"machine-monitor-agent/internal/types"
)

// LoadConfig carrega a configuração do arquivo JSON
func LoadConfig(configPath string) (*types.Config, error) {
	// Se o caminho não for fornecido, usa o padrão
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// Lê o arquivo de configuração
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo de configuração: %w", err)
	}

	// Faz o parse do JSON
	var config types.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("erro ao fazer parse da configuração: %w", err)
	}

	// Valida e completa a configuração
	if err := validateAndCompleteConfig(&config); err != nil {
		return nil, fmt.Errorf("erro na validação da configuração: %w", err)
	}

	return &config, nil
}

// SaveConfig salva a configuração no arquivo JSON
func SaveConfig(config *types.Config, configPath string) error {
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// Cria o diretório se não existir
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório de configuração: %w", err)
	}

	// Converte para JSON com indentação
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao converter configuração para JSON: %w", err)
	}

	// Escreve o arquivo
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("erro ao salvar arquivo de configuração: %w", err)
	}

	return nil
}

// getDefaultConfigPath retorna o caminho padrão do arquivo de configuração
func getDefaultConfigPath() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "MachineMonitor", "config.json")
	case "darwin":
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, "Library", "Application Support", "MachineMonitor", "config.json")
	default: // Linux e outros
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, ".config", "machine-monitor", "config.json")
	}
}

// validateAndCompleteConfig valida e completa a configuração com valores padrão
func validateAndCompleteConfig(config *types.Config) error {
	// Gera Machine ID se não existir
	if config.Agent.MachineID == "" {
		machineID, err := generateMachineID()
		if err != nil {
			return fmt.Errorf("erro ao gerar Machine ID: %w", err)
		}
		config.Agent.MachineID = machineID
	}

	// Valida configurações do servidor
	if config.Server.BaseURL == "" {
		config.Server.BaseURL = "http://localhost:3000"
	}
	if config.Server.HTTPPort == 0 {
		config.Server.HTTPPort = 3000
	}
	if config.Server.WSPort == 0 {
		config.Server.WSPort = 3001
	}
	if config.Server.Timeout == 0 {
		config.Server.Timeout = 30
	}
	if config.Server.MaxRetries == 0 {
		config.Server.MaxRetries = 3
	}
	if config.Server.RetryDelay == 0 {
		config.Server.RetryDelay = 5
	}

	// Valida configurações do agente
	if config.Agent.Name == "" {
		config.Agent.Name = "Machine Monitor Agent"
	}
	if config.Agent.Version == "" {
		config.Agent.Version = "1.0.0"
	}
	if config.Agent.HeartbeatInterval == 0 {
		config.Agent.HeartbeatInterval = 30
	}
	if config.Agent.InventoryInterval == 0 {
		config.Agent.InventoryInterval = 300
	}
	if config.Agent.MaxConcurrency == 0 {
		config.Agent.MaxConcurrency = 5
	}
	if config.Agent.DataCacheTTL == 0 {
		config.Agent.DataCacheTTL = 300
	}

	// Valida configurações de logging
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.File == "" {
		config.Logging.File = getDefaultLogPath()
	}
	if config.Logging.MaxSize == 0 {
		config.Logging.MaxSize = 100
	}
	if config.Logging.MaxAge == 0 {
		config.Logging.MaxAge = 7
	}

	// Valida configurações da UI
	if config.UI.WebUIPort == 0 {
		config.UI.WebUIPort = 8080
	}
	if config.UI.Theme == "" {
		config.UI.Theme = "dark"
	}

	// Valida configurações de segurança
	if len(config.Security.AllowedCommands) == 0 {
		config.Security.AllowedCommands = []string{"ping", "info", "restart"}
	}

	return nil
}

// generateMachineID gera um ID único para a máquina
func generateMachineID() (string, error) {
	// Gera 16 bytes aleatórios
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Converte para string hexadecimal
	return fmt.Sprintf("%x", bytes), nil
}

// getDefaultLogPath retorna o caminho padrão para os logs
func getDefaultLogPath() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "MachineMonitor", "logs", "agent.log")
	case "darwin":
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, "Library", "Logs", "MachineMonitor", "agent.log")
	default: // Linux e outros
		return "/var/log/machine-monitor/agent.log"
	}
}

// GetDataDirectory retorna o diretório de dados do aplicativo
func GetDataDirectory() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "MachineMonitor")
	case "darwin":
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, "Library", "Application Support", "MachineMonitor")
	default: // Linux e outros
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, ".local", "share", "machine-monitor")
	}
}

// EnsureDirectories garante que os diretórios necessários existam
func EnsureDirectories(config *types.Config) error {
	directories := []string{
		filepath.Dir(config.Logging.File),
		GetDataDirectory(),
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("erro ao criar diretório %s: %w", dir, err)
		}
	}

	return nil
}
