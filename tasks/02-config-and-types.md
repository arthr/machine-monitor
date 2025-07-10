# Task 02: Configuração e Tipos de Dados

## 🎯 Objetivo
Implementar sistema de configuração e definir tipos de dados principais do agente.

## 📋 Checklist

### ✅ Configuração Base
- [ ] Criar `internal/agent/config.go` com estrutura:
  ```go
  type Config struct {
      MachineID         string `json:"machine_id"`
      BackendURL        string `json:"backend_url"`
      WSUrl             string `json:"websocket_url"`
      Token             string `json:"token"`
      HeartbeatInterval int    `json:"heartbeat_interval"`
  }
  ```
- [ ] Implementar função `loadConfig(path string)` com validação
- [ ] Criar `configs/config.json` com dados de desenvolvimento

### ✅ Tipos de Dados
- [ ] Criar `internal/collector/types.go` com estruturas:
  - [ ] `SystemInfo` - Informações do sistema
  - [ ] `HardwareInfo` - Dados de hardware
  - [ ] `SoftwareInfo` - Dados de software
  - [ ] `NetworkInfo` - Informações de rede
  - [ ] `InventoryData` - Dados completos do inventário

### ✅ Tipos de Comunicação
- [ ] Criar `internal/comms/types.go` com estruturas:
  - [ ] `Command` - Comando recebido
  - [ ] `CommandResult` - Resultado do comando
  - [ ] `HeartbeatData` - Dados do heartbeat
  - [ ] `InventoryMessage` - Mensagem de inventário

### ✅ Logger Básico
- [ ] Criar `internal/logging/logger.go` com:
  - [ ] Interface `Logger`
  - [ ] Implementação básica com níveis
  - [ ] Configuração via arquivo/env

## 🎯 Resultado Esperado
- Sistema de configuração funcional
- Tipos de dados bem definidos
- Logger básico implementado
- Validação de configuração

## 🔗 Próxima Task
`03-main-entry.md` - Implementação do ponto de entrada

## 📝 Notas
- Usar tags JSON para serialização
- Validar configurações obrigatórias
- Preparar para múltiplos ambientes (dev, prod)
- Logger deve ser substituível (interface) 