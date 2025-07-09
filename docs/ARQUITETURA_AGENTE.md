🧱 Arquitetura do Agente em Go
*Otimizada para empresas pequenas/médias (25-50 máquinas)*

## 📁 Organização de Pastas

```text
agente/
│
├── cmd/                  # Entrada principal do agente
│   └── agente/           # main.go e bootstrapping
│
├── internal/             # Código interno, não exposto como lib
│   ├── agent/            # Ciclo principal do agente (loop, timers, ações)
│   ├── collector/        # Coleta de inventário e métricas
│   ├── executor/         # Execução de scripts e comandos remotos
│   ├── updater/          # Auto-update do agente (CRÍTICO)
│   ├── comms/            # Comunicação híbrida (WebSocket + HTTP)
│   ├── config/           # Leitura/validação da config local
│   ├── logging/          # Logger estruturado (zap/slog)
│   └── service/          # Integração com o sistema (systemd, Windows Service)
│
├── scripts/              # Scripts de instalação e manutenção
│   ├── windows/          # Scripts específicos para Windows
│   │   ├── install.ps1   # Instalação como Windows Service
│   │   ├── uninstall.ps1 # Desinstalação Windows
│   │   └── service.xml   # Configuração do serviço
│   ├── darwin/           # Scripts específicos para macOS
│   │   ├── install.sh    # Instalação como LaunchDaemon
│   │   ├── uninstall.sh  # Desinstalação macOS
│   │   └── com.empresa.agente.plist # LaunchDaemon config
│   └── linux/            # Scripts específicos para Linux
│       ├── install.sh    # Instalação como systemd service
│       ├── uninstall.sh  # Desinstalação Linux
│       └── agente.service # Configuração systemd
│
├── go.mod
└── README.md
```

⸻

## ⚙️ Componentes Técnicos

### 🔁 1. Agent Loop (agent/)

**Ciclo otimizado para rede local:**
```go
// Configuração para pequenas empresas
const (
    HeartbeatInterval = 60 * time.Second  // Reduzido para rede local
    CommandCheck     = 30 * time.Second  // Verificação de comandos
    UpdateCheck      = 6 * time.Hour     // Auto-update check
)
```

**Responsabilidades:**
• Gerencia conexão WebSocket persistente
• Executa coleta de inventário
• Processa comandos em tempo real
• Aplica atualizações automáticas

⸻

### 📥 2. Collector (collector/)

**Coleta otimizada para inventário corporativo:**

```go
type InventoryData struct {
    // Informações do sistema
    OS          string    `json:"os"`
    Hostname    string    `json:"hostname"`
    Uptime      int64     `json:"uptime"`
    LastBoot    time.Time `json:"last_boot"`
    
    // Hardware
    CPU         CPUInfo     `json:"cpu"`
    Memory      MemoryInfo  `json:"memory"`
    Disk        []DiskInfo  `json:"disks"`
    Network     []NetInfo   `json:"network"`
    
    // Software
    Software    []SoftwareInfo `json:"software"`
    Services    []ServiceInfo  `json:"services"`
    Updates     []UpdateInfo   `json:"updates"`
    
    // Segurança
    Antivirus   AntivirusInfo `json:"antivirus"`
    Firewall    FirewallInfo  `json:"firewall"`
    
    // Métricas de performance
    Performance PerformanceInfo `json:"performance"`
}
```

**Bibliotecas utilizadas:**
• `github.com/shirou/gopsutil` — Métricas cross-platform (CPU, RAM, disco, rede)
• `golang.org/x/sys/windows` — APIs Windows específicas (registry, WMI)
• `golang.org/x/sys/unix` — APIs Unix/Linux/macOS específicas
• Comandos nativos por plataforma para informações detalhadas

**Implementação cross-platform:**
```go
// Coleta específica por plataforma
func (c *Collector) collectPlatformSpecific() PlatformData {
    switch runtime.GOOS {
    case "windows":
        return c.collectWindows()
    case "darwin":
        return c.collectMacOS()
    case "linux":
        return c.collectLinux()
    default:
        return c.collectGeneric()
    }
}

// Exemplo de coleta Windows
func (c *Collector) collectWindows() PlatformData {
    // Registry, WMI, PowerShell
    // Installed programs via registry
    // Windows Update status
    // Domain/Workgroup info
}

// Exemplo de coleta macOS
func (c *Collector) collectMacOS() PlatformData {
    // system_profiler, launchctl
    // Installed apps via /Applications
    // Software Update status
    // System preferences
}

// Exemplo de coleta Linux
func (c *Collector) collectLinux() PlatformData {
    // /proc, /sys, systemctl
    // Package manager info (apt, yum, dnf)
    // Update status
    // Distribution info
}
```

⸻

### 🔧 3. Executor (executor/)

**Execução simplificada para ambiente controlado:**

```go
type CommandExecutor struct {
    allowedCommands map[string]bool
    timeout        time.Duration
    logger         *zap.Logger
}

// Lista de comandos permitidos por plataforma (whitelist)
var AllowedCommands = map[string]map[string]bool{
    "windows": {
        "systeminfo":      true,
        "tasklist":       true,
        "netstat":        true,
        "wmic":           true,
        "powershell":     true,  // Com validação adicional
        "net":            true,  // net start/stop services
        "sc":             true,  // service control
        "ipconfig":       true,
        "whoami":         true,
    },
    "darwin": {
        "system_profiler": true,
        "launchctl":      true,
        "ps":             true,
        "netstat":        true,
        "ifconfig":       true,
        "sw_vers":        true,
        "diskutil":       true,
        "whoami":         true,
    },
    "linux": {
        "systemctl":      true,
        "ps":             true,
        "netstat":        true,
        "lscpu":          true,
        "lsblk":          true,
        "free":           true,
        "uname":          true,
        "whoami":         true,
        "dpkg":           true,  // Para Debian/Ubuntu
        "rpm":            true,  // Para RedHat/CentOS
    },
}

// Executor cross-platform
func (e *CommandExecutor) Execute(cmd Command) CommandResult {
    platform := runtime.GOOS
    if !e.isAllowed(cmd.Name, platform) {
        return CommandResult{Error: "Command not allowed"}
    }
    
    return e.executePlatformCommand(cmd, platform)
}
```

**Características:**
• Timeout configurável (max 5 minutos)
• Logging detalhado de execução
• Validação de comandos (whitelist)
• Sanitização de parâmetros

⸻

### 🔄 4. Auto-Updater (updater/)

**Componente estratégico para crescimento:**

```go
type Updater struct {
    currentVersion string
    updateURL      string
    backupPath     string
    logger         *zap.Logger
}

func (u *Updater) UpdateFlow() error {
    // 1. Detectar plataforma (windows/darwin/linux)
    // 2. Verificar versão disponível para a plataforma
    // 3. Baixar binário correto (agente.exe, agente-darwin, agente-linux)
    // 4. Verificar integridade (SHA256)
    // 5. Backup do binário atual
    // 6. Substituir binário
    // 7. Reiniciar serviço (service/systemctl/launchctl)
    // 8. Verificar funcionamento
    // 9. Rollback se necessário
}

// URLs específicas por plataforma
type UpdateConfig struct {
    Windows string `json:"windows_url"`
    Darwin  string `json:"darwin_url"`
    Linux   string `json:"linux_url"`
}

// Exemplo de implementação
func (u *Updater) getBinaryURL() string {
    switch runtime.GOOS {
    case "windows":
        return u.config.Windows
    case "darwin":
        return u.config.Darwin
    case "linux":
        return u.config.Linux
    default:
        return ""
    }
}
```

**Vantagens para pequenas empresas:**
• Eliminação de visitas técnicas
• Atualizações durante horário comercial
• Rollback automático em caso de falha
• Deploy gradual (máquinas de teste primeiro)

⸻

### 🌐 5. Comms (comms/) - **ARQUITETURA HÍBRIDA**

**Comunicação otimizada para rede local:**

```go
type CommsManager struct {
    websocket *WebSocketComms  // Tempo real
    http      *HTTPComms       // Operações bulk
    fallback  *FallbackComms   // Recuperação
}

// WebSocket - Tempo real
type WebSocketComms struct {
    conn         *websocket.Conn
    commandChan  chan Command
    alertChan    chan Alert
    pingInterval time.Duration
}

// HTTP - Operações bulk
type HTTPComms struct {
    client      *http.Client
    retryCount  int
    timeout     time.Duration
}
```

**Fluxo de comunicação:**
```text
┌─────────────┐    WebSocket     ┌─────────────┐
│   Agente    │ ←──────────────→ │   Backend   │
│             │  • Comandos      │   (Local)   │
│             │  • Alertas       │             │
│             │  • Monitoramento │             │
│             │                  │             │
│             │    HTTP/REST     │             │
│             │ ←──────────────→ │             │
│             │  • Heartbeat     │             │
│             │  • Inventário    │             │
│             │  • Logs          │             │
└─────────────┘                  └─────────────┘
```

**Casos de uso:**
• **WebSocket**: Comandos urgentes, alertas críticos, monitoramento ativo
• **HTTP**: Inventário completo, logs históricos, heartbeat de fallback

⸻

### ⚙️ 6. Serviço do SO (service/)

**Instalação como serviço nativo:**

```go
// Usando github.com/kardianos/service
type ServiceConfig struct {
    Name        string
    DisplayName string
    Description string
    Executable  string
    Arguments   []string
    Dependencies []string
}

// Configurações específicas
var Config = ServiceConfig{
    Name:        "MachineMonitorAgent",
    DisplayName: "Agente de Monitoramento",
    Description: "Agente para controle de inventário corporativo",
    Dependencies: []string{"Tcpip", "Dnscache"},
}
```

**Características:**
• Início automático com o sistema
• Recuperação automática em caso de falha
• Logs integrados com Event Viewer (Windows)
• Integração com systemd/launchd

⸻

### 🧾 7. Configuração (config/)

**Configuração otimizada para pequenas empresas:**

```json
{
  "machine_id": "auto-generated-uuid",
  "platform": "auto-detected",
  "backend": {
    "websocket_url": "ws://backend.local:8080/ws",
    "http_url": "http://backend.local:8080",
    "token": "machine-auth-token"
  },
  "intervals": {
    "heartbeat": 60,
    "inventory": 300,
    "update_check": 21600
  },
  "security": {
    "allowed_commands": {
      "windows": ["systeminfo", "tasklist", "netstat", "wmic"],
      "darwin": ["system_profiler", "launchctl", "ps", "netstat"],
      "linux": ["systemctl", "ps", "netstat", "lscpu"]
    },
    "execution_timeout": 300,
    "log_level": "info"
  },
  "local_storage": {
    "max_log_size": "100MB",
    "backup_retention": "7d",
    "cache_offline": true,
    "paths": {
      "windows": {
        "log_dir": "%ProgramData%\\MachineMonitor\\logs",
        "config_dir": "%ProgramData%\\MachineMonitor\\config",
        "temp_dir": "%TEMP%\\MachineMonitor"
      },
      "darwin": {
        "log_dir": "/var/log/machinemonitor",
        "config_dir": "/etc/machinemonitor",
        "temp_dir": "/tmp/machinemonitor"
      },
      "linux": {
        "log_dir": "/var/log/machinemonitor",
        "config_dir": "/etc/machinemonitor",
        "temp_dir": "/tmp/machinemonitor"
      }
    }
  },
  "update": {
    "urls": {
      "windows": "https://backend.local:8080/updates/windows/agente.exe",
      "darwin": "https://backend.local:8080/updates/darwin/agente",
      "linux": "https://backend.local:8080/updates/linux/agente"
    }
  }
}
```

⸻

### 📜 8. Logging (logging/)

**Logging estruturado para troubleshooting:**

```go
type Logger struct {
    *zap.Logger
    file   *os.File
    remote RemoteLogger
}

// Níveis de log
const (
    DEBUG = "debug"  // Desenvolvimento
    INFO  = "info"   // Operações normais
    WARN  = "warn"   // Situações de atenção
    ERROR = "error"  // Erros críticos
)
```

**Características:**
• Rotação automática de logs
• Envio remoto opcional
• Estrutura JSON para parsing
• Correlação por machine_id

⸻

## 🔐 Segurança Adequada para Rede Local

### **Simplificada mas Eficaz:**
```go
type SecurityConfig struct {
    // Autenticação simples por token
    MachineToken string `json:"machine_token"`
    
    // Comunicação segura
    TLSEnabled   bool   `json:"tls_enabled"`
    
    // Validação de comandos
    CommandWhitelist []string `json:"command_whitelist"`
    
    // Timeout de execução
    MaxExecutionTime int `json:"max_execution_time"`
}
```

**Características:**
• HTTP/HTTPS conforme configuração
• Token de autenticação por máquina
• Whitelist de comandos permitidos
• Timeout para execução segura
• Logs de auditoria completos

⸻

## 🚀 Ciclo de Execução Otimizado

```go
func (a *Agent) Run() {
    // Inicialização
    a.comms.ConnectWebSocket()
    a.comms.StartHTTPHeartbeat()
    
    // Loop principal
    for {
        select {
        case <-time.After(60 * time.Second):
            // Heartbeat HTTP (fallback)
            a.comms.SendHeartbeat()
            
        case <-time.After(300 * time.Second):
            // Inventário completo
            inventory := a.collector.CollectFull()
            a.comms.SendInventory(inventory)
            
        case cmd := <-a.comms.CommandChannel():
            // Comando em tempo real via WebSocket
            result := a.executor.Execute(cmd)
            a.comms.SendResult(cmd.ID, result)
            
        case <-time.After(6 * time.Hour):
            // Verificação de atualização
            if a.updater.CheckUpdate() {
                a.updater.Apply()
            }
        }
    }
}
```

⸻

## 🎯 Benefícios para Pequenas Empresas

### **✅ Controle em Tempo Real**
• Comandos executados instantaneamente
• Alertas críticos imediatos
• Monitoramento proativo

### **✅ Baixo Custo de Manutenção**
• Auto-atualização elimina visitas técnicas
• Configuração centralizada
• Troubleshooting remoto

### **✅ Escalabilidade Natural**
• Arquitetura suporta crescimento para 200+ máquinas
• Componentes modulares
• Fácil adição de funcionalidades

### **✅ ROI Rápido**
• Implementação em 4-6 semanas
• Economia de tempo significativa
• Visibilidade completa do parque

⸻

## 📊 Recursos Necessários

### **Por Agente:**
• CPU: < 2% (idle), < 5% (coleta)
• RAM: < 30MB baseline
• Disco: < 50MB (binário + logs)
• Rede: < 10KB/min (operação normal)

### **Backend (25-50 máquinas, multi-plataforma):**
• Servidor: 2 cores CPU, 4GB RAM
• Banco: PostgreSQL simples
• Armazenamento: < 8GB/ano (incluindo binários de múltiplas plataformas)
• Largura de banda: Mínima (rede local)
• Espaço para binários: ~150MB (50MB × 3 plataformas)

⸻

## 🎉 Plano de Implementação

### **Fase 1 - Core Multi-Plataforma (3-4 semanas)**
✅ Agent loop + WebSocket + HTTP
✅ Collector básico com detecção de plataforma
✅ Auto-updater cross-platform
✅ Instalação como serviço (Windows/macOS/Linux)

### **Fase 2 - Funcionalidades Específicas (2-3 semanas)**
✅ Executor + Comandos específicos por plataforma
✅ Inventário completo cross-platform
✅ Alertas e monitoramento
✅ Interface de gerenciamento unificada

### **Fase 3 - Testes e Refinamentos (2-3 semanas)**
✅ Testes em ambiente misto (Windows/Mac/Linux)
✅ Otimizações de performance por plataforma
✅ Scripts de deployment automatizado
✅ Documentação e treinamento para múltiplas plataformas

**Total:** 7-10 semanas para implementação completa (incluindo suporte cross-platform)

⸻

## 🖥️ Suporte Cross-Platform (Windows, macOS, Linux)

### **Detecção Automática de Plataforma**
```go
func detectPlatform() PlatformInfo {
    return PlatformInfo{
        OS:      runtime.GOOS,
        Arch:    runtime.GOARCH,
        Version: getOSVersion(),
    }
}
```

### **Diferenças por Plataforma**

#### **🪟 Windows**
```powershell
# Coleta específica
- Registry (HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall)
- WMI queries (Win32_ComputerSystem, Win32_OperatingSystem)
- PowerShell cmdlets (Get-Service, Get-Process)
- Windows Update API

# Instalação
- Windows Service via sc.exe
- Event Viewer integration
- %ProgramData% para dados
```

#### **🍎 macOS**
```bash
# Coleta específica
- system_profiler SPHardwareDataType
- /Applications para software instalado
- launchctl para serviços
- softwareupdate -l para updates

# Instalação
- LaunchDaemon (/Library/LaunchDaemons/)
- /var/log para logs
- sudo privileges necessários
```

#### **🐧 Linux**
```bash
# Coleta específica
- /proc/cpuinfo, /proc/meminfo
- Package managers (apt, yum, dnf, pacman)
- systemctl para serviços
- /etc/os-release para distro info

# Instalação
- systemd service (/etc/systemd/system/)
- /var/log para logs
- sudo privileges necessários
```

### **Compilação Multi-Plataforma**
```bash
# Build para todas as plataformas
go build -ldflags="-s -w" -o releases/windows/agente.exe ./cmd/agente
GOOS=darwin go build -ldflags="-s -w" -o releases/darwin/agente ./cmd/agente
GOOS=linux go build -ldflags="-s -w" -o releases/linux/agente ./cmd/agente
```

### **Deployment Unificado**
```text
releases/
├── windows/
│   ├── agente.exe
│   ├── install.ps1
│   └── config.json
├── darwin/
│   ├── agente
│   ├── install.sh
│   └── config.json
└── linux/
    ├── agente
    ├── install.sh
    └── config.json
```

### **Vantagens do Suporte Multi-Plataforma**
✅ **Flexibilidade**: Suporta diferentes preferências de usuário
✅ **Unified Management**: Um único backend para todas as plataformas
✅ **Consistent Experience**: Mesma funcionalidade em todos os SOs
✅ **Cost Effective**: Reduz complexidade de gerenciamento
✅ **Future Proof**: Pronto para mudanças no parque tecnológico

### **Considerações Especiais**

| **Aspecto** | **Windows** | **macOS** | **Linux** |
|-------------|-------------|-----------|-----------|
| **Serviços** | Windows Service | LaunchDaemon | systemd |
| **Logs** | Event Viewer | syslog | journald |
| **Configs** | %ProgramData% | /etc | /etc |
| **Privilégios** | SYSTEM/Admin | sudo | sudo |
| **Software** | Registry | /Applications | Package Manager |
| **Updates** | Windows Update | softwareupdate | apt/yum/dnf |
| **Instalação** | MSI/EXE | .pkg/.dmg | .deb/.rpm |
| **Firewall** | Windows Firewall | pfctl | iptables/firewalld |

### **Complexidades Adicionais**
- **Permissões**: Cada OS tem diferentes requisitos de privilégios
- **Paths**: Caminhos específicos para logs, config e temp
- **Services**: Diferentes sistemas de gerenciamento de serviços
- **Updates**: Diferentes métodos de distribuição de binários
- **Security**: Considerações específicas de cada plataforma
- **Testing**: Necessidade de testes em ambiente misto
- **Deployment**: Scripts específicos para cada plataforma
