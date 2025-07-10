# Agente macOS

Agente de monitoramento para macOS - **Projeto Funcional Completo** 🎉

## 🎯 Objetivo

Agente de monitoramento macOS com funcionalidades essenciais implementadas:
- ✅ Coleta de inventário do sistema macOS
- ✅ Comunicação HTTP + WebSocket com backend
- ✅ Execução segura de comandos remotos
- ✅ Reconnect automático e robustez
- ✅ Sistema de logging estruturado
- ✅ Padronização de dados
- ✅ Gerenciamento de comunicação completo

## 🏗️ Estrutura

```
agente-poc/
├── cmd/
│   └── agente/          # Ponto de entrada principal
├── internal/
│   ├── agent/           # Loop principal do agente
│   ├── collector/       # Coleta de dados do sistema
│   ├── comms/           # Comunicação HTTP + WebSocket
│   ├── executor/        # Execução de comandos
│   └── logging/         # Sistema de logging
├── configs/             # Arquivos de configuração
├── go.mod              # Módulo Go
└── README.md           # Este arquivo
```

## 🚀 Instalação e Execução

### 1. Preparação do Ambiente

```bash
# Navegar para o diretório do projeto
cd agente-poc

# Instalar dependências
go mod tidy
```

### 2. Configuração

Edite `configs/config.json`:

```json
{
  "machine_id": "macos-dev-001",
  "backend_url": "http://localhost:8080",
  "websocket_url": "ws://localhost:8080",
  "token": "dev-token-123",
  "heartbeat_interval": 30
}
```

### 3. Build e Execução

```bash
# Opção 1: Executar diretamente em desenvolvimento
go run ./cmd/agente

# Opção 2: Compilar e executar
go build -o agente-macos ./cmd/agente
./agente-macos

# Opção 3: Build com otimizações para produção
go build -ldflags "-s -w" -o agente-macos ./cmd/agente
```

## 🔧 Desenvolvimento

### Pré-requisitos
- Go 1.21+
- macOS (para coleta específica do sistema)
- Backend rodando (localhost:8080)

### Comandos de Desenvolvimento
```bash
# Executar em modo desenvolvimento
go run ./cmd/agente

# Compilar para debug
go build ./cmd/agente

# Executar testes
go test ./...

# Limpar cache de build
go clean -cache

# Verificar módulos
go mod verify
```

## 📊 Backend de Desenvolvimento

Este agente conecta com o backend de debug em `../backend-debug/`:

```bash
# Iniciar backend
cd ../backend-debug
npm install
npm start

# Backend estará disponível em http://localhost:8080
```

## 🎯 Status do Projeto

**✅ PROJETO FUNCIONALMENTE COMPLETO**

- ✅ Setup do projeto
- ✅ Configuração e tipos
- ✅ Ponto de entrada
- ✅ Loop principal do agente
- ✅ Sistema de coleta de dados
- ✅ Gerenciador de comunicação
- ✅ Executor de comandos
- ✅ Testes de integração
- ✅ Documentação completa

## 🏃‍♂️ Execução Rápida

```bash
# Terminal 1: Iniciar backend
cd backend-debug && npm start

# Terminal 2: Executar agente
cd agente-poc && go run ./cmd/agente
```

## 📝 Funcionalidades Implementadas

### Coleta de Dados
- Informações do sistema operacional
- Especificações de hardware
- Uso de CPU e memória
- Inventário de software instalado

### Comunicação
- HTTP para operações síncronas
- WebSocket para comandos em tempo real
- Heartbeat automático
- Reconnect inteligente

### Execução de Comandos
- Execução segura de comandos remotos
- Timeout configurável
- Logging de todas as operações
- Tratamento de erros robusto

### Sistema de Logging
- Logs estruturados em JSON
- Diferentes níveis de log
- Rotação automática de logs
- Debug detalhado disponível

## 🛠️ Troubleshooting

### Problemas Comuns

1. **Agente não conecta ao backend**
   ```bash
   # Verificar se o backend está rodando
   curl http://localhost:8080/health
   ```

2. **Erro de permissões no macOS**
   ```bash
   # Dar permissões de execução
   chmod +x agente-macos
   ```

3. **Problemas de build**
   ```bash
   # Limpar cache e reinstalar dependências
   go clean -cache
   go mod tidy
   ```

## 📋 Arquivos Importantes

- `configs/config.json` - Configuração do agente
- `internal/agent/agent.go` - Loop principal
- `internal/collector/collector.go` - Coleta de dados
- `internal/comms/manager.go` - Gerenciamento de comunicação

---

**Agente macOS - Versão Funcional Completa** ✅ 