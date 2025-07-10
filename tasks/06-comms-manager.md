# Task 06: Gerenciador de Comunicação

## 🎯 Objetivo
Implementar o gerenciador de comunicação (HTTP + WebSocket) para conectar com o backend.

## 📋 Checklist

### ✅ Estrutura Base
- [ ] Criar `internal/comms/manager.go` com:
  - [ ] Struct `Manager` com clientes HTTP/WS
  - [ ] Função `New()` para criação
  - [ ] Função `Connect()` para inicialização
  - [ ] Função `Disconnect()` para finalização

### ✅ Cliente HTTP
- [ ] Implementar `internal/comms/http.go`:
  - [ ] Cliente HTTP configurável
  - [ ] Função `sendHTTP()` genérica
  - [ ] Retry automático com backoff
  - [ ] Timeout configurável
  - [ ] Headers de autenticação

### ✅ Cliente WebSocket
- [ ] Implementar `internal/comms/websocket.go`:
  - [ ] Conexão WebSocket com reconnect
  - [ ] Função `connectWebSocket()` com retry
  - [ ] Função `handleWebSocketMessages()`
  - [ ] Ping/Pong para keep-alive
  - [ ] Canal de comandos recebidos

### ✅ Operações Principais
- [ ] Implementar no Manager:
  - [ ] `SendHeartbeat()` - POST /heartbeat
  - [ ] `SendInventory()` - POST /inventory
  - [ ] `SendResult()` - Via WebSocket
  - [ ] `CommandChannel()` - Canal de comandos

### ✅ Tratamento de Conexão
- [ ] Implementar:
  - [ ] Reconnect automático para WebSocket
  - [ ] Detecção de desconexão
  - [ ] Queue para mensagens enquanto desconectado
  - [ ] Fallback HTTP para dados críticos

### ✅ Segurança
- [ ] Implementar:
  - [ ] Autenticação via Bearer token
  - [ ] Validação de certificados TLS
  - [ ] Headers de segurança
  - [ ] Sanitização de dados

### ✅ Monitoramento
- [ ] Implementar:
  - [ ] Métricas de conexão
  - [ ] Contadores de mensagens
  - [ ] Latência de requests
  - [ ] Status de conectividade

## 🎯 Resultado Esperado
- Comunicação HTTP/WebSocket funcional
- Reconnect automático
- Tratamento robusto de erros
- Autenticação segura
- Monitoramento de conectividade

## 🔗 Próxima Task
`07-command-executor.md` - Implementação do executor de comandos

## 📝 Notas
- Usar context.Context para timeouts
- Implementar circuit breaker para falhas
- Queue de mensagens deve ter limite
- Logs detalhados para debug de rede
- Preparar para múltiplos backends (load balancing) 