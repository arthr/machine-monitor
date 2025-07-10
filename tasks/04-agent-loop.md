# Task 04: Loop Principal do Agente

## 🎯 Objetivo
Implementar o loop principal do agente que coordena todas as operações (heartbeat, coleta, comandos).

## 📋 Checklist

### ✅ Estrutura do Agent
- [ ] Criar `internal/agent/agent.go` com:
  - [ ] Struct `Agent` com componentes necessários
  - [ ] Função `New()` para criação
  - [ ] Função `Start()` para inicialização
  - [ ] Função `Stop()` para finalização

### ✅ Integração de Componentes
- [ ] Integrar dependências:
  - [ ] Config (já criado)
  - [ ] Collector (interface)
  - [ ] Comms Manager (interface)
  - [ ] Executor (interface)
  - [ ] Logger (interface)

### ✅ Loop Principal
- [ ] Implementar loop com:
  - [ ] Context para cancelamento
  - [ ] Timer para heartbeat
  - [ ] Timer para coleta de inventário
  - [ ] Canal para comandos recebidos
  - [ ] Tratamento de erros

### ✅ Operações Principais
- [ ] Implementar métodos:
  - [ ] `sendHeartbeat()` - Envio periódico de heartbeat
  - [ ] `sendInventory()` - Coleta e envio de inventário
  - [ ] `handleCommand()` - Processamento de comandos
  - [ ] `handleError()` - Tratamento de erros

### ✅ Gerenciamento de Estado
- [ ] Implementar controle de:
  - [ ] Estado do agente (starting, running, stopping)
  - [ ] Contadores de operações
  - [ ] Última execução de cada operação
  - [ ] Métricas básicas

### ✅ Configurações Dinâmicas
- [ ] Suporte a:
  - [ ] Intervalos configuráveis
  - [ ] Retry automático
  - [ ] Backoff exponencial
  - [ ] Timeouts ajustáveis

## 🎯 Resultado Esperado
- Loop principal funcional
- Coordenação entre componentes
- Operações temporizadas funcionando
- Tratamento robusto de erros
- Estado do agente bem definido

## 🔗 Próxima Task
`05-collector-system.md` - Implementação do coletor de sistema

## 📝 Notas
- Usar channels para comunicação entre goroutines
- Implementar timeouts para todas as operações
- Logs detalhados para debug
- Preparar para monitoramento de métricas
- Considerar circuit breaker para operações críticas 