🚀 POC do Agente - macOS
*Proof of Concept rápida para ambiente de desenvolvimento*

## 🎯 Objetivo da POC

Criar uma versão **funcional mínima** do agente para macOS que demonstre:
- ✅ Coleta básica de inventário
- ✅ Comunicação WebSocket + HTTP
- ✅ Execução de comandos remotos
- ✅ Auto-atualização simulada
- ✅ Base para expansão futura

**Tempo estimado**: 1-2 semanas

⸻

## 📁 Estrutura da POC

```text
agente-poc/
├── cmd/
│   └── agente/
│       └── main.go                 # Entrada principal
├── internal/
│   ├── agent/
│   │   ├── agent.go               # Loop principal
│   │   └── config.go              # Configuração básica
│   ├── collector/
│   │   ├── collector.go           # Interface principal
│   │   ├── system.go              # Coleta de sistema (macOS)
│   │   └── types.go               # Estruturas de dados
│   ├── comms/
│   │   ├── manager.go             # Gerenciador de comunicação
│   │   ├── websocket.go           # Cliente WebSocket
│   │   └── http.go                # Cliente HTTP
│   ├── executor/
│   │   ├── executor.go            # Executor de comandos
│   │   └── commands.go            # Comandos permitidos macOS
│   └── logging/
│       └── logger.go              # Logger básico
├── configs/
│   └── config.json                # Configuração da POC
├── go.mod
└── README.md
```

⸻

## 🔧 Implementação por Componente

### **1. Entrada Principal (cmd/agente/main.go)**

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/empresa/agente-poc/internal/agent"
)

func main() {
    // Configurar logging básico
    log.SetFlags(log.LstdFlags | log.Lshortfile)
    
    // Criar contexto com cancelamento
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Inicializar agente
    agent, err := agent.New("configs/config.json")
    if err != nil {
        log.Fatal("Erro ao inicializar agente:", err)
    }
    
    // Iniciar agente em goroutine
    go func() {
        if err := agent.Start(ctx); err != nil {
            log.Fatal("Erro ao iniciar agente:", err)
        }
    }()
    
    // Aguardar sinal de interrupção
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    log.Println("Agente iniciado. Pressione Ctrl+C para parar...")
    <-sigChan
    
    log.Println("Parando agente...")
    cancel()
    
    // Aguardar finalização
    time.Sleep(time.Second)
    log.Println("Agente parado")
}
```

### **2. Agent Loop (internal/agent/agent.go)**

```go
package agent

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "time"

    "github.com/empresa/agente-poc/internal/collector"
    "github.com/empresa/agente-poc/internal/comms"
    "github.com/empresa/agente-poc/internal/executor"
)

type Agent struct {
    config    *Config
    collector *collector.Collector
    comms     *comms.Manager
    executor  *executor.Executor
}

type Config struct {
    MachineID   string `json:"machine_id"`
    BackendURL  string `json:"backend_url"`
    WSUrl       string `json:"websocket_url"`
    Token       string `json:"token"`
    HeartbeatInterval int `json:"heartbeat_interval"`
}

func New(configPath string) (*Agent, error) {
    // Carregar configuração
    config, err := loadConfig(configPath)
    if err != nil {
        return nil, err
    }
    
    // Inicializar componentes
    collector := collector.New()
    comms := comms.New(config.BackendURL, config.WSUrl, config.Token)
    executor := executor.New()
    
    return &Agent{
        config:    config,
        collector: collector,
        comms:     comms,
        executor:  executor,
    }, nil
}

func (a *Agent) Start(ctx context.Context) error {
    log.Println("Iniciando agente...")
    
    // Conectar comunicação
    if err := a.comms.Connect(ctx); err != nil {
        return err
    }
    
    // Loop principal
    heartbeatTicker := time.NewTicker(time.Duration(a.config.HeartbeatInterval) * time.Second)
    inventoryTicker := time.NewTicker(5 * time.Minute)
    
    defer heartbeatTicker.Stop()
    defer inventoryTicker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            log.Println("Contexto cancelado, parando agente...")
            return nil
            
        case <-heartbeatTicker.C:
            a.sendHeartbeat()
            
        case <-inventoryTicker.C:
            a.sendInventory()
            
        case cmd := <-a.comms.CommandChannel():
            a.handleCommand(cmd)
        }
    }
}

func (a *Agent) sendHeartbeat() {
    log.Println("Enviando heartbeat...")
    
    heartbeat := map[string]interface{}{
        "machine_id": a.config.MachineID,
        "timestamp": time.Now().Unix(),
        "status": "online",
    }
    
    a.comms.SendHeartbeat(heartbeat)
}

func (a *Agent) sendInventory() {
    log.Println("Coletando inventário...")
    
    inventory := a.collector.CollectAll()
    inventory["machine_id"] = a.config.MachineID
    inventory["timestamp"] = time.Now().Unix()
    
    a.comms.SendInventory(inventory)
}

func (a *Agent) handleCommand(cmd comms.Command) {
    log.Printf("Executando comando: %s", cmd.Name)
    
    result := a.executor.Execute(cmd)
    
    a.comms.SendResult(comms.CommandResult{
        ID:        cmd.ID,
        Output:    result.Output,
        Error:     result.Error,
        Timestamp: time.Now().Unix(),
    })
}

func loadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    var config Config
    if err := json.Unmarshal(data, &config); err != nil {
        return nil, err
    }
    
    return &config, nil
}
```

### **3. Collector para macOS (internal/collector/system.go)**

```go
package collector

import (
    "encoding/json"
    "log"
    "os"
    "os/exec"
    "runtime"
    "strings"
    "time"

    "github.com/shirou/gopsutil/v3/cpu"
    "github.com/shirou/gopsutil/v3/disk"
    "github.com/shirou/gopsutil/v3/host"
    "github.com/shirou/gopsutil/v3/mem"
    "github.com/shirou/gopsutil/v3/net"
)

type Collector struct{}

func New() *Collector {
    return &Collector{}
}

func (c *Collector) CollectAll() map[string]interface{} {
    result := make(map[string]interface{})
    
    // Informações básicas
    result["platform"] = runtime.GOOS
    result["architecture"] = runtime.GOARCH
    result["hostname"], _ = os.Hostname()
    result["timestamp"] = time.Now().Unix()
    
    // Informações do sistema usando gopsutil
    if hostInfo, err := host.Info(); err == nil {
        result["host_info"] = hostInfo
    }
    
    // CPU
    if cpuInfo, err := cpu.Info(); err == nil {
        result["cpu_info"] = cpuInfo
    }
    
    if cpuPercent, err := cpu.Percent(time.Second, false); err == nil {
        result["cpu_usage"] = cpuPercent
    }
    
    // Memória
    if memInfo, err := mem.VirtualMemory(); err == nil {
        result["memory"] = memInfo
    }
    
    // Disco
    if diskInfo, err := disk.Usage("/"); err == nil {
        result["disk"] = diskInfo
    }
    
    // Rede
    if netInfo, err := net.Interfaces(); err == nil {
        result["network"] = netInfo
    }
    
    // Informações específicas do macOS
    result["macos_info"] = c.collectMacOSSpecific()
    
    return result
}

func (c *Collector) collectMacOSSpecific() map[string]interface{} {
    result := make(map[string]interface{})
    
    // System Profiler - Hardware
    if output, err := exec.Command("system_profiler", "SPHardwareDataType", "-json").Output(); err == nil {
        var hardware map[string]interface{}
        if json.Unmarshal(output, &hardware) == nil {
            result["hardware"] = hardware
        }
    }
    
    // System Profiler - Software
    if output, err := exec.Command("system_profiler", "SPSoftwareDataType", "-json").Output(); err == nil {
        var software map[string]interface{}
        if json.Unmarshal(output, &software) == nil {
            result["software"] = software
        }
    }
    
    // Aplicações instaladas
    result["applications"] = c.getInstalledApps()
    
    // Serviços em execução
    result["services"] = c.getRunningServices()
    
    // Informações do sistema
    if output, err := exec.Command("sw_vers").Output(); err == nil {
        result["system_version"] = strings.TrimSpace(string(output))
    }
    
    return result
}

func (c *Collector) getInstalledApps() []map[string]string {
    apps := []map[string]string{}
    
    // Listar aplicações em /Applications
    entries, err := os.ReadDir("/Applications")
    if err != nil {
        log.Printf("Erro ao ler /Applications: %v", err)
        return apps
    }
    
    for _, entry := range entries {
        if strings.HasSuffix(entry.Name(), ".app") {
            app := map[string]string{
                "name": strings.TrimSuffix(entry.Name(), ".app"),
                "path": "/Applications/" + entry.Name(),
            }
            apps = append(apps, app)
        }
    }
    
    return apps
}

func (c *Collector) getRunningServices() []map[string]string {
    services := []map[string]string{}
    
    // Listar serviços com launchctl
    output, err := exec.Command("launchctl", "list").Output()
    if err != nil {
        log.Printf("Erro ao executar launchctl: %v", err)
        return services
    }
    
    lines := strings.Split(string(output), "\n")
    for i, line := range lines {
        if i == 0 || line == "" { // Pular cabeçalho e linhas vazias
            continue
        }
        
        parts := strings.Fields(line)
        if len(parts) >= 3 {
            service := map[string]string{
                "pid":   parts[0],
                "status": parts[1],
                "label": parts[2],
            }
            services = append(services, service)
        }
    }
    
    return services
}
```

### **4. Comunicação (internal/comms/manager.go)**

```go
package comms

import (
    "bytes"
    "context"
    "encoding/json"
    "log"
    "net/http"
    "time"

    "github.com/gorilla/websocket"
)

type Manager struct {
    httpURL    string
    wsURL      string
    token      string
    httpClient *http.Client
    wsConn     *websocket.Conn
    commandCh  chan Command
}

type Command struct {
    ID        string                 `json:"id"`
    Name      string                 `json:"name"`
    Args      []string               `json:"args"`
    Timestamp int64                  `json:"timestamp"`
}

type CommandResult struct {
    ID        string `json:"id"`
    Output    string `json:"output"`
    Error     string `json:"error"`
    Timestamp int64  `json:"timestamp"`
}

func New(httpURL, wsURL, token string) *Manager {
    return &Manager{
        httpURL:    httpURL,
        wsURL:      wsURL,
        token:      token,
        httpClient: &http.Client{Timeout: 30 * time.Second},
        commandCh:  make(chan Command, 10),
    }
}

func (m *Manager) Connect(ctx context.Context) error {
    // Conectar WebSocket
    go m.connectWebSocket(ctx)
    
    return nil
}

func (m *Manager) connectWebSocket(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            log.Printf("Conectando ao WebSocket: %s", m.wsURL)
            
            // Tentar conectar
            dialer := websocket.Dialer{
                HandshakeTimeout: 10 * time.Second,
            }
            
            headers := http.Header{}
            headers.Set("Authorization", "Bearer "+m.token)
            
            conn, _, err := dialer.Dial(m.wsURL, headers)
            if err != nil {
                log.Printf("Erro ao conectar WebSocket: %v", err)
                time.Sleep(5 * time.Second)
                continue
            }
            
            m.wsConn = conn
            log.Println("WebSocket conectado")
            
            // Processar mensagens
            m.handleWebSocketMessages(ctx)
            
            // Se chegou aqui, conexão foi perdida
            m.wsConn = nil
            log.Println("WebSocket desconectado, tentando reconectar...")
            time.Sleep(5 * time.Second)
        }
    }
}

func (m *Manager) handleWebSocketMessages(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            var cmd Command
            if err := m.wsConn.ReadJSON(&cmd); err != nil {
                log.Printf("Erro ao ler mensagem WebSocket: %v", err)
                return
            }
            
            log.Printf("Comando recebido via WebSocket: %s", cmd.Name)
            
            // Enviar para canal de comandos
            select {
            case m.commandCh <- cmd:
            default:
                log.Println("Canal de comandos cheio, descartando comando")
            }
        }
    }
}

func (m *Manager) SendHeartbeat(data map[string]interface{}) {
    m.sendHTTP("/heartbeat", data)
}

func (m *Manager) SendInventory(data map[string]interface{}) {
    m.sendHTTP("/inventory", data)
}

func (m *Manager) SendResult(result CommandResult) {
    if m.wsConn != nil {
        if err := m.wsConn.WriteJSON(result); err != nil {
            log.Printf("Erro ao enviar resultado via WebSocket: %v", err)
        }
    }
}

func (m *Manager) sendHTTP(endpoint string, data interface{}) {
    jsonData, err := json.Marshal(data)
    if err != nil {
        log.Printf("Erro ao serializar dados: %v", err)
        return
    }
    
    req, err := http.NewRequest("POST", m.httpURL+endpoint, bytes.NewReader(jsonData))
    if err != nil {
        log.Printf("Erro ao criar request: %v", err)
        return
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+m.token)
    
    resp, err := m.httpClient.Do(req)
    if err != nil {
        log.Printf("Erro ao enviar HTTP %s: %v", endpoint, err)
        return
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        log.Printf("HTTP %s retornou status %d", endpoint, resp.StatusCode)
    }
}

func (m *Manager) CommandChannel() <-chan Command {
    return m.commandCh
}
```

### **5. Executor de Comandos (internal/executor/executor.go)**

```go
package executor

import (
    "context"
    "log"
    "os/exec"
    "strings"
    "time"

    "github.com/empresa/agente-poc/internal/comms"
)

type Executor struct {
    allowedCommands map[string]bool
}

type Result struct {
    Output string
    Error  string
}

func New() *Executor {
    // Comandos permitidos para macOS
    allowedCommands := map[string]bool{
        "system_profiler": true,
        "launchctl":      true,
        "ps":             true,
        "netstat":        true,
        "ifconfig":       true,
        "sw_vers":        true,
        "diskutil":       true,
        "whoami":         true,
        "uname":          true,
        "top":            true,
        "df":             true,
        "uptime":         true,
    }
    
    return &Executor{
        allowedCommands: allowedCommands,
    }
}

func (e *Executor) Execute(cmd comms.Command) Result {
    // Verificar se comando é permitido
    if !e.allowedCommands[cmd.Name] {
        return Result{
            Error: "Comando não permitido: " + cmd.Name,
        }
    }
    
    log.Printf("Executando comando: %s %v", cmd.Name, cmd.Args)
    
    // Criar contexto com timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Executar comando
    execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
    output, err := execCmd.CombinedOutput()
    
    result := Result{
        Output: string(output),
    }
    
    if err != nil {
        result.Error = err.Error()
    }
    
    // Limitar tamanho da saída
    if len(result.Output) > 10000 {
        result.Output = result.Output[:10000] + "\n... (truncado)"
    }
    
    return result
}
```

### **6. Configuração da POC (configs/config.json)**

```json
{
  "machine_id": "poc-macos-dev",
  "backend_url": "http://localhost:8080",
  "websocket_url": "ws://localhost:8080/ws",
  "token": "dev-token-123",
  "heartbeat_interval": 30
}
```

### **7. Dependências (go.mod)**

```go
module github.com/empresa/agente-poc

go 1.21

require (
    github.com/gorilla/websocket v1.5.0
    github.com/shirou/gopsutil/v3 v3.23.10
)

require (
    github.com/go-ole/go-ole v1.2.6 // indirect
    github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
    github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
    github.com/shoenig/go-m1cpu v0.1.6 // indirect
    github.com/tklauser/go-sysconf v0.3.12 // indirect
    github.com/tklauser/numcpus v0.6.1 // indirect
    github.com/yusufpapurcu/wmi v1.2.3 // indirect
    golang.org/x/sys v0.13.0 // indirect
)
```

⸻

## 🚀 Como Executar a POC

### **1. Inicialização**
```bash
# Clonar/criar projeto
mkdir agente-poc
cd agente-poc

# Inicializar módulo Go
go mod init github.com/empresa/agente-poc

# Instalar dependências
go get github.com/gorilla/websocket
go get github.com/shirou/gopsutil/v3

# Criar estrutura de pastas
mkdir -p cmd/agente internal/{agent,collector,comms,executor,logging} configs
```

### **2. Execução**
```bash
# Compilar
go build -o agente ./cmd/agente

# Executar
./agente

# Ou executar diretamente
go run ./cmd/agente
```

### **3. Testes Manuais**
```bash
# Verificar se está coletando dados
curl -H "Authorization: Bearer dev-token-123" http://localhost:8080/machines

# Enviar comando via WebSocket (usando ferramenta como wscat)
wscat -c ws://localhost:8080/ws -H "Authorization: Bearer dev-token-123"
```

⸻

## 🎯 Funcionalidades da POC

### **✅ Implementadas**
- Coleta de inventário macOS (hardware, software, serviços)
- Comunicação WebSocket + HTTP
- Execução de comandos remotos
- Heartbeat automático
- Logging básico

### **🔄 Próximos Passos**
- Auto-updater simulado
- Persistência de configuração
- Melhor tratamento de erros
- Testes automatizados
- Integração com backend

⸻

## 🐛 Debugging

### **Logs Importantes**
```bash
# Monitorar logs
tail -f /var/log/agente-poc.log

# Verificar conexões
netstat -an | grep 8080

# Verificar processos
ps aux | grep agente
```

### **Problemas Comuns**
- **WebSocket não conecta**: Verificar se backend está rodando
- **Comando não executa**: Verificar whitelist de comandos
- **Alta CPU**: Reduzir frequência de coleta
- **Dados incompletos**: Verificar permissões do usuário

⸻

## 📈 Métricas da POC

### **Performance Esperada**
- **CPU**: < 1% em idle
- **RAM**: < 20MB
- **Startup**: < 2 segundos
- **Coleta completa**: < 5 segundos

### **Testes de Stress**
```bash
# Múltiplos comandos simultâneos
for i in {1..10}; do echo '{"id":"test-'$i'","name":"ps","args":["aux"]}' | wscat -c ws://localhost:8080/ws & done

# Coleta contínua
while true; do curl -s http://localhost:8080/inventory > /dev/null; sleep 1; done
```

Esta POC fornece uma **base sólida** para desenvolvimento e testes do agente completo! 