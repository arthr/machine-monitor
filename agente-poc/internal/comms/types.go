package comms

import (
	"agente-poc/internal/collector"
	"time"
)

// Command representa um comando recebido do backend
type Command struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Command      string                 `json:"command"`
	Args         []string               `json:"args,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
	Timeout      int                    `json:"timeout,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	RequiresAuth bool                   `json:"requires_auth,omitempty"`
}

// CommandResult representa o resultado da execução de um comando
type CommandResult struct {
	ID            string    `json:"id"`
	CommandID     string    `json:"command_id"`
	Status        string    `json:"status"` // "success", "error", "timeout", "rejected"
	Output        string    `json:"output,omitempty"`
	Error         string    `json:"error,omitempty"`
	ExitCode      int       `json:"exit_code,omitempty"`
	ExecutionTime int64     `json:"execution_time_ms"`
	Timestamp     time.Time `json:"timestamp"`
}

// HeartbeatData representa os dados enviados no heartbeat
type HeartbeatData struct {
	MachineID       string             `json:"machine_id"`
	Timestamp       time.Time          `json:"timestamp"`
	Status          string             `json:"status"` // "online", "offline", "error"
	AgentVersion    string             `json:"agent_version"`
	Uptime          int64              `json:"uptime_seconds"`
	LastInventory   time.Time          `json:"last_inventory,omitempty"`
	SystemHealth    SystemHealthStatus `json:"system_health"`
	PendingCommands int                `json:"pending_commands"`
	ActiveTasks     []string           `json:"active_tasks,omitempty"`
}

// SystemHealthStatus representa o status de saúde do sistema
type SystemHealthStatus struct {
	CPUUsage    float64 `json:"cpu_usage_percent"`
	MemoryUsage float64 `json:"memory_usage_percent"`
	DiskUsage   float64 `json:"disk_usage_percent"`
	Status      string  `json:"status"` // "healthy", "warning", "critical"
}

// InventoryMessage representa uma mensagem de inventário
type InventoryMessage struct {
	Type      string                  `json:"type"`
	MachineID string                  `json:"machine_id"`
	Timestamp time.Time               `json:"timestamp"`
	Data      collector.InventoryData `json:"data"`
	Checksum  string                  `json:"checksum,omitempty"`
}

// WebSocketMessage representa uma mensagem WebSocket genérica
type WebSocketMessage struct {
	Type      string      `json:"type"`
	ID        string      `json:"id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// AuthRequest representa uma requisição de autenticação
type AuthRequest struct {
	MachineID string `json:"machine_id"`
	Token     string `json:"token"`
	Version   string `json:"version"`
}

// AuthResponse representa a resposta de autenticação
type AuthResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message,omitempty"`
	SessionToken string `json:"session_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`
}

// RegistrationRequest representa uma requisição de registro
type RegistrationRequest struct {
	MachineID    string                 `json:"machine_id"`
	Token        string                 `json:"token"`
	SystemInfo   collector.SystemInfo   `json:"system_info"`
	HardwareInfo collector.HardwareInfo `json:"hardware_info"`
	AgentVersion string                 `json:"agent_version"`
	Timestamp    time.Time              `json:"timestamp"`
}

// RegistrationResponse representa a resposta de registro
type RegistrationResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	MachineID string `json:"machine_id,omitempty"`
	Token     string `json:"token,omitempty"`
}

// ErrorResponse representa uma resposta de erro
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// StatusUpdate representa uma atualização de status
type StatusUpdate struct {
	MachineID string    `json:"machine_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// LogMessage representa uma mensagem de log
type LogMessage struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// ConfigUpdate representa uma atualização de configuração
type ConfigUpdate struct {
	MachineID string                 `json:"machine_id"`
	Config    map[string]interface{} `json:"config"`
	Timestamp time.Time              `json:"timestamp"`
}

// FileTransferRequest representa uma requisição de transferência de arquivo
type FileTransferRequest struct {
	ID          string `json:"id"`
	MachineID   string `json:"machine_id"`
	FilePath    string `json:"file_path"`
	Destination string `json:"destination"`
	Action      string `json:"action"` // "upload", "download"
	Checksum    string `json:"checksum,omitempty"`
}

// FileTransferResponse representa a resposta de transferência de arquivo
type FileTransferResponse struct {
	ID       string `json:"id"`
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Checksum string `json:"checksum,omitempty"`
}

// RemoteTaskRequest representa uma requisição de tarefa remota
type RemoteTaskRequest struct {
	ID         string                 `json:"id"`
	MachineID  string                 `json:"machine_id"`
	TaskType   string                 `json:"task_type"`
	Parameters map[string]interface{} `json:"parameters"`
	Schedule   string                 `json:"schedule,omitempty"`
	Priority   int                    `json:"priority,omitempty"`
	Timeout    int                    `json:"timeout,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// RemoteTaskResponse representa a resposta de tarefa remota
type RemoteTaskResponse struct {
	ID        string                 `json:"id"`
	TaskID    string                 `json:"task_id"`
	Status    string                 `json:"status"`
	Result    map[string]interface{} `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Progress  int                    `json:"progress,omitempty"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}
