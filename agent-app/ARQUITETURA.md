# Arquitetura do Agente macOS

## Visão Geral da Arquitetura

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              AGENTE macOS                                   │
│                              (Versão 1.0.0)                                │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                            PONTO DE ENTRADA                                 │
│                         cmd/agente/main.go                                  │
│                                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐             │
│  │   Configuração  │  │   Logging       │  │   Sinais OS     │             │
│  │   - Flags CLI   │  │   - Níveis      │  │   - SIGINT      │             │
│  │   - Config JSON │  │   - Estruturado │  │   - SIGTERM     │             │
│  │   - Validação   │  │   - Rotação     │  │   - Shutdown    │             │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘             │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          NÚCLEO DO AGENTE                                   │
│                        internal/agent/agent.go                              │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                      GERENCIAMENTO DE ESTADO                         │ │
│  │                                                                       │ │
│  │  Estados: Starting → Running → Stopping → Stopped                    │ │
│  │                             ↓                                        │ │
│  │                           Error                                       │ │
│  │                                                                       │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐      │ │
│  │  │    Métricas     │  │  Circuit Breaker │  │    Retry Config │      │ │
│  │  │  - Heartbeats   │  │  - Falhas       │  │  - Max Retries  │      │ │
│  │  │  - Comandos     │  │  - Timeout      │  │  - Backoff      │      │ │
│  │  │  - Inventários  │  │  - Estado       │  │  - Jitter       │      │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘      │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          LOOPS DE EXECUÇÃO                                  │
│                        (5 Goroutines Paralelas)                            │
│                                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐             │
│  │   Collector     │  │  Communications │  │   Main Loop     │             │
│  │   - Timer       │  │   - Start/Stop  │  │   - Health      │             │
│  │   - Inventory   │  │   - Connection  │  │   - Monitoring  │             │
│  │   - Async       │  │   - Errors      │  │   - Metrics     │             │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘             │
│                                                                             │
│  ┌─────────────────┐  ┌─────────────────┐                                  │
│  │ Command Proc.   │  │ Error Handler   │                                  │
│  │ - Canal CMD     │  │ - Canal Errors  │                                  │
│  │ - Execução      │  │ - Tratamento    │                                  │
│  │ - Resultados    │  │ - Recuperação   │                                  │
│  └─────────────────┘  └─────────────────┘                                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Componentes Principais

### 1. Sistema de Coleta de Dados (Collector)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    SYSTEM COLLECTOR                                         │
│                  internal/collector/                                        │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                         COLETA PARALELA                                 │ │
│  │                                                                         │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐        │ │
│  │  │  System Info    │  │  Hardware Info  │  │  Software Info  │        │ │
│  │  │  - OS Version   │  │  - CPU Details  │  │  - Installed    │        │ │
│  │  │  - Hostname     │  │  - Memory       │  │  - Processes    │        │ │
│  │  │  - Uptime       │  │  - Storage      │  │  - Services     │        │ │
│  │  │  - Machine ID   │  │  - Network HW   │  │  - Applications │        │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘        │ │
│  │                                                                         │ │
│  │  ┌─────────────────┐  ┌─────────────────┐                              │ │
│  │  │  Network Info   │  │  macOS Specific │                              │ │
│  │  │  - Interfaces   │  │  - System_profiler │                           │ │
│  │  │  - Connections  │  │  - Security     │                              │ │
│  │  │  - Statistics   │  │  - Permissions  │                              │ │
│  │  │  - Routing      │  │  - Keychain     │                              │ │
│  │  └─────────────────┘  └─────────────────┘                              │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                      CACHE E OTIMIZAÇÕES                                │ │
│  │                                                                         │ │
│  │  • Cache com TTL (5 min)                                               │ │
│  │  • Coleta assíncrona                                                   │ │
│  │  • Timeout por operação (30s)                                          │ │
│  │  • Fallback para dados essenciais                                      │ │
│  │  • Geração automática de Machine ID                                    │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2. Sistema de Comunicação (Communications)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    COMMUNICATIONS MANAGER                                   │
│                     internal/comms/                                        │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                     DUPLA CONECTIVIDADE                                 │ │
│  │                                                                         │ │
│  │  ┌─────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                    HTTP CLIENT                                      │ │ │
│  │  │                                                                     │ │ │
│  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐    │ │ │
│  │  │  │   Registration  │  │   Heartbeat     │  │   Inventory     │    │ │ │
│  │  │  │   - Machine     │  │   - Status      │  │   - Full Data   │    │ │ │
│  │  │  │   - Auth        │  │   - Health      │  │   - Periodic    │    │ │ │
│  │  │  │   - Metadata    │  │   - Metrics     │  │   - Sync        │    │ │ │
│  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘    │ │ │
│  │  └─────────────────────────────────────────────────────────────────────┘ │ │
│  │                                                                         │ │
│  │  ┌─────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                  WEBSOCKET CLIENT                                   │ │ │
│  │  │                                                                     │ │ │
│  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐    │ │ │
│  │  │  │   Commands      │  │   Real-time     │  │   Auto-reconnect│    │ │ │
│  │  │  │   - Execution   │  │   - Bidirectional│  │   - Backoff     │    │ │ │
│  │  │  │   - Results     │  │   - Push/Pull   │  │   - Queue       │    │ │ │
│  │  │  │   - Status      │  │   - Monitoring  │  │   - Persistence │    │ │ │
│  │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘    │ │ │
│  │  └─────────────────────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                        SEGURANÇA                                        │ │
│  │                                                                         │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐        │ │
│  │  │  Token Manager  │  │  Rate Limiter   │  │  Input Sanitizer│        │ │
│  │  │  - Refresh      │  │  - Throttling   │  │  - Validation   │        │ │
│  │  │  - Validation   │  │  - Protection   │  │  - Injection    │        │ │
│  │  │  - Scope        │  │  - Quotas       │  │  - Prevention   │        │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘        │ │
│  │                                                                         │ │
│  │  ┌─────────────────┐  ┌─────────────────┐                              │ │
│  │  │  Certificate    │  │  TLS Config     │                              │ │
│  │  │  - Pinning      │  │  - Min Version  │                              │ │
│  │  │  - Validation   │  │  - Cipher Suites│                              │ │
│  │  │  - Chain        │  │  - Verification │                              │ │
│  │  └─────────────────┘  └─────────────────┘                              │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3. Sistema de Execução de Comandos (Executor)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      COMMAND EXECUTOR                                       │
│                   internal/executor/                                        │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                    TIPOS DE COMANDO                                     │ │
│  │                                                                         │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐        │ │
│  │  │     Shell       │  │      Info       │  │      Ping       │        │ │
│  │  │  - Bash/Zsh     │  │  - System Data  │  │  - Connectivity │        │ │
│  │  │  - Timeout      │  │  - Real-time    │  │  - Latency      │        │ │
│  │  │  - Sanitization │  │  - Inventory    │  │  - Response     │        │ │
│  │  │  - Output       │  │  - Metrics      │  │  - Validation   │        │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘        │ │
│  │                                                                         │ │
│  │  ┌─────────────────┐                                                    │ │
│  │  │    Restart      │                                                    │ │
│  │  │  - Graceful     │                                                    │ │
│  │  │  - Notification │                                                    │ │
│  │  │  - Validation   │                                                    │ │
│  │  │  - Safety       │                                                    │ │
│  │  └─────────────────┘                                                    │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                  CONTROLE DE EXECUÇÃO                                   │ │
│  │                                                                         │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐        │ │
│  │  │  Concurrency    │  │    Timeout      │  │    Security     │        │ │
│  │  │  - Semaphore    │  │  - Per Command  │  │  - Validation   │        │ │
│  │  │  - Max Parallel │  │  - Context      │  │  - Sandboxing   │        │ │
│  │  │  - Queue        │  │  - Cancellation │  │  - Permissions  │        │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘        │ │
│  │                                                                         │ │
│  │  ┌─────────────────┐  ┌─────────────────┐                              │ │
│  │  │    Metrics      │  │    Results      │                              │ │
│  │  │  - Execution    │  │  - Status       │                              │ │
│  │  │  - Duration     │  │  - Output       │                              │ │
│  │  │  - Success/Fail │  │  - Error        │                              │ │
│  │  │  - Performance  │  │  - Metadata     │                              │ │
│  │  └─────────────────┘  └─────────────────┘                              │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Fluxo de Dados Principal

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         FLUXO DE DADOS                                      │
│                                                                             │
│  START                                                                      │
│    │                                                                        │
│    ▼                                                                        │
│  ┌─────────────────┐                                                        │
│  │  Inicialização  │                                                        │
│  │  - Config       │                                                        │
│  │  - Logger       │                                                        │
│  │  - Componentes  │                                                        │
│  └─────────────────┘                                                        │
│            │                                                                │
│            ▼                                                                │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐        │
│  │   Collector     │    │  Communications │    │    Executor     │        │
│  │   Timer Loop    │    │   Connection    │    │   Standby       │        │
│  │      │          │    │      │          │    │      │          │        │
│  │      ▼          │    │      ▼          │    │      ▼          │        │
│  │  ┌─────────────┐│    │  ┌─────────────┐│    │  ┌─────────────┐│        │
│  │  │  Collect    ││    │  │  Register   ││    │  │   Listen    ││        │
│  │  │  Inventory  ││    │  │  Machine    ││    │  │  Commands   ││        │
│  │  │     │       ││    │  │     │       ││    │  │     │       ││        │
│  │  │     ▼       ││    │  │     ▼       ││    │  │     ▼       ││        │
│  │  │  Send via   ││    │  │  Start      ││    │  │  Execute    ││        │
│  │  │  HTTP       ││    │  │  Heartbeat  ││    │  │  & Return   ││        │
│  │  │  (Periodic) ││    │  │  (Timer)    ││    │  │  Results    ││        │
│  │  └─────────────┘│    │  │     │       ││    │  │     │       ││        │
│  │                 │    │  │     ▼       ││    │  │     ▼       ││        │
│  └─────────────────┘    │  │  WebSocket  ││    │  │  Commands   ││        │
│                         │  │  Listen     ││    │  │  via WS     ││        │
│                         │  │  Commands   ││    │  │             ││        │
│                         │  │     │       ││    │  └─────────────┘│        │
│                         │  │     ▼       ││    │                 │        │
│                         │  │  Forward    ││    │                 │        │
│                         │  │  to Agent   ││    │                 │        │
│                         │  └─────────────┘│    │                 │        │
│                         └─────────────────┘    └─────────────────┘        │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                        CANAIS DE COMUNICAÇÃO                            │ │
│  │                                                                         │ │
│  │  Agent ←→ Collector:   Timer-based triggers                            │ │
│  │  Agent ←→ Comms:       Bidirectional channels                          │ │
│  │  Agent ←→ Executor:    Command/Result channels                         │ │
│  │  Comms ←→ Backend:     HTTP + WebSocket                                │ │
│  │                                                                         │ │
│  │  Error Handling:       Dedicated error channel                         │ │
│  │  Metrics:             Centralized collection                           │ │
│  │  Shutdown:            Graceful with timeout                            │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Protocolos de Comunicação

### Heartbeat (HTTP)
```
Agent                                  Backend
  │                                       │
  │  POST /api/machines/heartbeat         │
  │  {                                    │
  │    "machine_id": "macos-001",         │
  │    "status": "online",                │
  │    "timestamp": "2024-01-01T10:00:00Z"│
  │    "system_health": {...},            │
  │    "agent_version": "1.0.0"           │
  │  }                                    │
  │ ────────────────────────────────────► │
  │                                       │
  │  200 OK                               │
  │  {                                    │
  │    "status": "received",              │
  │    "next_heartbeat": 30,              │
  │    "commands_pending": 0              │
  │  }                                    │
  │ ◄──────────────────────────────────── │
```

### Inventory (HTTP)
```
Agent                                  Backend
  │                                       │
  │  POST /api/machines/inventory         │
  │  {                                    │
  │    "machine_id": "macos-001",         │
  │    "timestamp": "2024-01-01T10:00:00Z"│
  │    "system_info": {...},              │
  │    "hardware_info": {...},            │
  │    "software_info": {...},            │
  │    "network_info": {...},             │
  │    "macos_info": {...}                │
  │  }                                    │
  │ ────────────────────────────────────► │
  │                                       │
  │  200 OK                               │
  │  {                                    │
  │    "status": "received",              │
  │    "inventory_id": "inv-12345"        │
  │  }                                    │
  │ ◄──────────────────────────────────── │
```

### Commands (WebSocket)
```
Backend                                Agent
  │                                       │
  │  {                                    │
  │    "type": "command",                 │
  │    "id": "cmd-12345",                 │
  │    "command": {                       │
  │      "type": "shell",                 │
  │      "command": "ls -la",             │
  │      "timeout": 30                    │
  │    }                                  │
  │  }                                    │
  │ ────────────────────────────────────► │
  │                                       │
  │                                       │ [Execução]
  │                                       │
  │  {                                    │
  │    "type": "command_result",          │
  │    "id": "result-12345",              │
  │    "command_id": "cmd-12345",         │
  │    "result": {                        │
  │      "status": "success",             │
  │      "output": "total 24\ndrwx...",   │
  │      "exit_code": 0,                  │
  │      "execution_time": 234            │
  │    }                                  │
  │  }                                    │
  │ ◄──────────────────────────────────── │
```

## Métricas e Monitoramento

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          SISTEMA DE MÉTRICAS                                │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                        MÉTRICAS DO AGENTE                               │ │
│  │                                                                         │ │
│  │  • StartTime          - Tempo de inicialização                         │ │
│  │  • HeartbeatCount     - Total de heartbeats enviados                   │ │
│  │  • InventoryCount     - Total de inventários enviados                  │ │
│  │  • CommandsExecuted   - Total de comandos executados                   │ │
│  │  • CommandsSuccessful - Comandos executados com sucesso                │ │
│  │  • CommandsFailed     - Comandos que falharam                          │ │
│  │  • LastHeartbeat      - Timestamp do último heartbeat                  │ │
│  │  • LastInventory      - Timestamp do último inventário                 │ │
│  │  • LastCommand        - Timestamp do último comando                    │ │
│  │  • ErrorCount         - Total de erros                                 │ │
│  │  • RetryCount         - Total de tentativas de retry                   │ │
│  │  • ConnectionAttempts - Tentativas de conexão                          │ │
│  │  • ConnectionFailures - Falhas de conexão                              │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                   MÉTRICAS DE COMUNICAÇÃO                               │ │
│  │                                                                         │ │
│  │  • TotalUptime        - Tempo total de atividade                       │ │
│  │  • HTTPRequests       - Total de requisições HTTP                      │ │
│  │  • WSMessages         - Total de mensagens WebSocket                   │ │
│  │  • ConnectionStatus   - Status da conexão atual                        │ │
│  │  • LastError          - Último erro ocorrido                           │ │
│  │  • LastErrorTime      - Timestamp do último erro                       │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                    HEALTH CHECK ENDPOINT                                │ │
│  │                                                                         │ │
│  │  GET /health                                                            │ │
│  │  {                                                                      │ │
│  │    "state": "running",                                                  │ │
│  │    "machine_id": "macos-001",                                           │ │
│  │    "uptime": "2h30m15s",                                                │ │
│  │    "heartbeat_count": 300,                                              │ │
│  │    "last_heartbeat": "2024-01-01T10:00:00Z",                           │ │
│  │    "system_health": {                                                   │ │
│  │      "cpu_usage_percent": 25.5,                                         │ │
│  │      "memory_usage_percent": 68.3,                                      │ │
│  │      "disk_usage_percent": 45.2,                                        │ │
│  │      "status": "healthy"                                                │ │
│  │    },                                                                   │ │
│  │    "circuit_breaker": "closed"                                          │ │
│  │  }                                                                      │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Tratamento de Erros e Recuperação

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      ESTRATÉGIAS DE RECUPERAÇÃO                             │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                      CIRCUIT BREAKER                                    │ │
│  │                                                                         │ │
│  │  CLOSED ──[5 falhas]──► OPEN ──[30s timeout]──► HALF-OPEN              │ │
│  │    │                     │                          │                  │ │
│  │    │                     │                          │                  │ │
│  │    │                     │                          ▼                  │ │
│  │    │                     │                    [3 tentativas]           │ │
│  │    │                     │                      sucesso/falha          │ │
│  │    │                     │                          │                  │ │
│  │    │                     ▼                          │                  │ │
│  │  Permite            Rejeita                        │                  │ │
│  │  Requisições        Requisições                     │                  │ │
│  │                                                     │                  │ │
│  │  ◄──────────────────────┐                         │                  │ │
│  │                         │                         │                  │ │
│  │                         └─────────────────────────┘                  │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                       RETRY STRATEGY                                    │ │
│  │                                                                         │ │
│  │  Tentativa 1: Imediata                                                 │ │
│  │  Tentativa 2: 2s + jitter                                              │ │
│  │  Tentativa 3: 4s + jitter                                              │ │
│  │  Tentativa 4: 8s + jitter                                              │ │
│  │  Tentativa 5: 16s + jitter                                             │ │
│  │  Max: 10 tentativas                                                    │ │
│  │                                                                         │ │
│  │  Jitter: ± 25% do tempo base                                           │ │
│  │  Backoff: Exponencial com limitação                                    │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                    TIPOS DE ERRO                                        │ │
│  │                                                                         │ │
│  │  • Network Errors     → Retry com backoff                              │ │
│  │  • Authentication    → Refresh token                                   │ │
│  │  • Rate Limiting     → Exponential backoff                             │ │
│  │  • Command Timeout   → Kill processo e relatar                         │ │
│  │  • Invalid Command   → Rejeitar imediatamente                          │ │
│  │  • System Errors     → Log e tentar recuperar                          │ │
│  │  • Config Errors     → Parar agente com erro                           │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Configuração e Deployment

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CONFIGURAÇÃO                                      │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                     CONFIG.JSON                                         │ │
│  │                                                                         │ │
│  │  {                                                                      │ │
│  │    "machine_id": "macos-dev-001",                                       │ │
│  │    "backend_url": "http://localhost:8080",                             │ │
│  │    "websocket_url": "ws://localhost:8080/ws",                          │ │
│  │    "token": "dev-token-123",                                            │ │
│  │    "heartbeat_interval": 30,                                            │ │
│  │    "collection_interval": 300,                                          │ │
│  │    "command_timeout": 30,                                               │ │
│  │    "retry_interval": 5,                                                 │ │
│  │    "max_retries": 5,                                                    │ │
│  │    "log_level": "info",                                                 │ │
│  │    "debug": false                                                       │ │
│  │  }                                                                      │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                    COMMAND LINE FLAGS                                   │ │
│  │                                                                         │ │
│  │  -config string     → Arquivo de configuração                          │ │
│  │  -log-level string  → Nível de log (debug,info,warning,error)          │ │
│  │  -verbose           → Modo debug                                        │ │
│  │  -version           → Mostrar versão                                    │ │
│  │  -help              → Mostrar ajuda                                     │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                      DEPLOYMENT                                         │ │
│  │                                                                         │ │
│  │  Desenvolvimento:                                                       │ │
│  │  $ go run ./cmd/agente                                                  │ │
│  │                                                                         │ │
│  │  Produção:                                                              │ │
│  │  $ go build -ldflags "-s -w" -o agente ./cmd/agente                    │ │
│  │  $ ./agente -config /etc/agente/config.json                            │ │
│  │                                                                         │ │
│  │  Service (macOS):                                                       │ │
│  │  $ sudo launchctl load /Library/LaunchDaemons/com.agente.plist         │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Resumo das Funcionalidades

### ✅ Implementadas
- [x] Coleta completa de inventário do sistema macOS
- [x] Comunicação HTTP síncrona (heartbeat, inventory, registration)
- [x] Comunicação WebSocket assíncrona (comandos em tempo real)
- [x] Execução segura de comandos remotos
- [x] Sistema de logging estruturado
- [x] Reconnect automático com backoff exponencial
- [x] Circuit breaker para proteção contra falhas
- [x] Retry strategy configurável
- [x] Métricas completas de performance
- [x] Graceful shutdown com timeout
- [x] Configuração flexível via JSON e flags
- [x] Tratamento robusto de erros
- [x] Gerenciamento de estado do agente
- [x] Sistema de segurança (tokens, TLS, sanitização)
- [x] Health check endpoint
- [x] Cache inteligente para dados do sistema
- [x] Execução paralela de coletas
- [x] Geração automática de Machine ID

### 🔄 Em Desenvolvimento
- [ ] Implementação completa de comandos shell
- [ ] Sistema de notificações push
- [ ] Relatórios detalhados de performance
- [ ] Dashboard web integrado

### 🎯 Próximos Passos
1. Testes de integração completos
2. Benchmarks de performance
3. Documentação de API
4. Packaging para distribuição
5. Integração com sistemas de monitoramento

---

**Agente macOS v1.0.0 - Arquitetura Completa e Funcional** ✅ 