# Agente macOS POC

Proof of Concept do agente de monitoramento para macOS.

## 🎯 Objetivo

Demonstrar funcionalidades essenciais:
- ✅ Coleta de inventário do sistema macOS
- ✅ Comunicação HTTP + WebSocket com backend
- ✅ Execução segura de comandos remotos
- ✅ Reconnect automático e robustez

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

## 🚀 Instalação

```bash
# Clonar ou baixar projeto
cd agente-poc

# Instalar dependências
go mod tidy

# Compilar
go build -o agente ./cmd/agente

# Executar
./agente
```

## ⚙️ Configuração

Edite `configs/config.json`:

```json
{
  "machine_id": "seu-machine-id",
  "backend_url": "http://localhost:8080",
  "websocket_url": "ws://localhost:8080",
  "token": "dev-token-123",
  "heartbeat_interval": 30
}
```

## 🔧 Desenvolvimento

### Pré-requisitos
- Go 1.21+
- macOS (para coleta específica do sistema)
- Backend rodando (localhost:8080)

### Comandos
```bash
go run ./cmd/agente          # Executar em desenvolvimento
go build ./cmd/agente        # Compilar
go test ./...                # Executar testes
```

## 📊 Backend

Este agente conecta com o backend de debug em `../backend-debug/`:

```bash
cd ../backend-debug
npm start
```

## 🎯 Status da POC

- [ ] Setup do projeto ⏳
- [ ] Configuração e tipos
- [ ] Ponto de entrada
- [ ] Loop principal
- [ ] Coletor de sistema
- [ ] Gerenciador de comunicação
- [ ] Executor de comandos
- [ ] Testes de integração
- [ ] Documentação final

## 📝 Notas

- POC focada em funcionalidade, não performance
- Segurança básica implementada
- Expansível para Windows/Linux no futuro
- Configurações para ambiente de desenvolvimento

---

**Agente em desenvolvimento** 🚧 