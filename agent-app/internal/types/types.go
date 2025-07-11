package types

import (
	"time"
)

// Config representa a configuração do agente
type Config struct {
	Server   ServerConfig   `json:"server"`
	Agent    AgentConfig    `json:"agent"`
	Logging  LoggingConfig  `json:"logging"`
	UI       UIConfig       `json:"ui"`
	Security SecurityConfig `json:"security"`
}

// ServerConfig configurações do servidor backend
type ServerConfig struct {
	BaseURL    string `json:"base_url"`
	HTTPPort   int    `json:"http_port"`
	WSPort     int    `json:"ws_port"`
	UseHTTPS   bool   `json:"use_https"`
	Timeout    int    `json:"timeout"`
	MaxRetries int    `json:"max_retries"`
	RetryDelay int    `json:"retry_delay"`
}

// AgentConfig configurações do agente
type AgentConfig struct {
	MachineID         string `json:"machine_id"`
	Name              string `json:"name"`
	Version           string `json:"version"`
	HeartbeatInterval int    `json:"heartbeat_interval"`
	InventoryInterval int    `json:"inventory_interval"`
	MaxConcurrency    int    `json:"max_concurrency"`
	DataCacheTTL      int    `json:"data_cache_ttl"`
}

// LoggingConfig configurações de logging
type LoggingConfig struct {
	Level    string `json:"level"`
	File     string `json:"file"`
	MaxSize  int    `json:"max_size"`
	MaxAge   int    `json:"max_age"`
	Compress bool   `json:"compress"`
}

// UIConfig configurações da interface
type UIConfig struct {
	ShowTrayIcon bool   `json:"show_tray_icon"`
	WebUIPort    int    `json:"webui_port"`
	Theme        string `json:"theme"`
	AutoStart    bool   `json:"auto_start"`
}

// SecurityConfig configurações de segurança
type SecurityConfig struct {
	APIKey          string   `json:"api_key"`
	EnableTLS       bool     `json:"enable_tls"`
	CertFile        string   `json:"cert_file"`
	KeyFile         string   `json:"key_file"`
	ValidateCerts   bool     `json:"validate_certs"`
	AllowedCommands []string `json:"allowed_commands"`
}

// SystemInfo informações do sistema
type SystemInfo struct {
	OS        string    `json:"os"`
	Platform  string    `json:"platform"`
	Hostname  string    `json:"hostname"`
	Uptime    uint64    `json:"uptime"`
	BootTime  uint64    `json:"boot_time"`
	Procs     uint64    `json:"procs"`
	Users     []User    `json:"users"`
	Timestamp time.Time `json:"timestamp"`
}

// HardwareInfo informações de hardware
type HardwareInfo struct {
	CPU       CPUInfo       `json:"cpu"`
	Memory    MemoryInfo    `json:"memory"`
	Disk      []DiskInfo    `json:"disk"`
	Network   []NetworkInfo `json:"network"`
	Timestamp time.Time     `json:"timestamp"`
}

// CPUInfo informações da CPU
type CPUInfo struct {
	ModelName   string    `json:"model_name"`
	Cores       int32     `json:"cores"`
	Threads     int32     `json:"threads"`
	Frequency   float64   `json:"frequency"`
	Usage       float64   `json:"usage"`
	Temperature float64   `json:"temperature"`
	Timestamp   time.Time `json:"timestamp"`
}

// MemoryInfo informações de memória
type MemoryInfo struct {
	Total       uint64    `json:"total"`
	Available   uint64    `json:"available"`
	Used        uint64    `json:"used"`
	UsedPercent float64   `json:"used_percent"`
	Free        uint64    `json:"free"`
	Timestamp   time.Time `json:"timestamp"`
}

// DiskInfo informações de disco
type DiskInfo struct {
	Device      string    `json:"device"`
	Mountpoint  string    `json:"mountpoint"`
	Fstype      string    `json:"fstype"`
	Total       uint64    `json:"total"`
	Used        uint64    `json:"used"`
	Free        uint64    `json:"free"`
	UsedPercent float64   `json:"used_percent"`
	Timestamp   time.Time `json:"timestamp"`
}

// NetworkInfo informações de rede
type NetworkInfo struct {
	Name         string    `json:"name"`
	HardwareAddr string    `json:"hardware_addr"`
	Flags        []string  `json:"flags"`
	Addrs        []string  `json:"addrs"`
	BytesSent    uint64    `json:"bytes_sent"`
	BytesRecv    uint64    `json:"bytes_recv"`
	PacketsSent  uint64    `json:"packets_sent"`
	PacketsRecv  uint64    `json:"packets_recv"`
	Timestamp    time.Time `json:"timestamp"`
}

// User informações de usuário
type User struct {
	Username  string    `json:"username"`
	Terminal  string    `json:"terminal"`
	Host      string    `json:"host"`
	Started   int64     `json:"started"`
	Timestamp time.Time `json:"timestamp"`
}

// Inventory inventário completo da máquina
type Inventory struct {
	MachineID string       `json:"machine_id"`
	System    SystemInfo   `json:"system"`
	Hardware  HardwareInfo `json:"hardware"`
	Timestamp time.Time    `json:"timestamp"`
}

// Command comando recebido do servidor
type Command struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Timeout   int               `json:"timeout"`
	Metadata  map[string]string `json:"metadata"`
	Timestamp time.Time         `json:"timestamp"`
}

// CommandResult resultado da execução do comando
type CommandResult struct {
	ID        string    `json:"id"`
	Success   bool      `json:"success"`
	Output    string    `json:"output"`
	Error     string    `json:"error"`
	ExitCode  int       `json:"exit_code"`
	Duration  int64     `json:"duration"`
	Timestamp time.Time `json:"timestamp"`
}

// HeartbeatData dados do heartbeat
type HeartbeatData struct {
	MachineID string    `json:"machine_id"`
	Status    string    `json:"status"`
	Uptime    uint64    `json:"uptime"`
	CPUUsage  float64   `json:"cpu_usage"`
	MemUsage  float64   `json:"mem_usage"`
	Timestamp time.Time `json:"timestamp"`
}

// AgentStatus status do agente
type AgentStatus struct {
	State         string        `json:"state"`
	LastHeartbeat time.Time     `json:"last_heartbeat"`
	LastInventory time.Time     `json:"last_inventory"`
	CommandsRun   int64         `json:"commands_run"`
	Errors        int64         `json:"errors"`
	Uptime        time.Duration `json:"uptime"`
}

// Estados possíveis do agente
const (
	StateStarting = "starting"
	StateRunning  = "running"
	StateStopping = "stopping"
	StateStopped  = "stopped"
	StateError    = "error"
)

// Tipos de comando
const (
	CommandTypeShell   = "shell"
	CommandTypeInfo    = "info"
	CommandTypePing    = "ping"
	CommandTypeRestart = "restart"
)

// Níveis de log
const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)
