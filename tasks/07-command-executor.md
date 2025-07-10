# Task 07: Executor de Comandos

## 🎯 Objetivo
Implementar o executor de comandos remotos com lista de comandos permitidos e validação de segurança.

## 📋 Checklist

### ✅ Estrutura Base
- [ ] Criar `internal/executor/executor.go` com:
  - [ ] Struct `Executor` com whitelist
  - [ ] Função `New()` para criação
  - [ ] Função `Execute()` principal
  - [ ] Struct `Result` para retorno

### ✅ Whitelist de Comandos
- [ ] Implementar `internal/executor/commands.go`:
  - [ ] Lista de comandos permitidos para macOS
  - [ ] Validação de comando + argumentos
  - [ ] Configuração via arquivo/env
  - [ ] Comandos específicos por usuário/grupo

### ✅ Execução Segura
- [ ] Implementar:
  - [ ] Validação de comandos contra whitelist
  - [ ] Sanitização de argumentos
  - [ ] Timeout configurável por comando
  - [ ] Limitação de recursos (CPU, memória)

### ✅ Comandos macOS
- [ ] Configurar comandos permitidos:
  - [ ] `system_profiler` com parâmetros específicos
  - [ ] `launchctl list` para serviços
  - [ ] `ps aux` para processos
  - [ ] `netstat -an` para rede
  - [ ] `sw_vers` para versão do sistema
  - [ ] `diskutil list` para discos
  - [ ] `top -l 1` para estatísticas

### ✅ Controle de Execução
- [ ] Implementar:
  - [ ] Context com timeout
  - [ ] Cancelamento de comandos
  - [ ] Limitação de saída (truncar se muito grande)
  - [ ] Captura de stderr e stdout

### ✅ Tratamento de Erros
- [ ] Implementar:
  - [ ] Códigos de erro específicos
  - [ ] Logs detalhados de execução
  - [ ] Retry para comandos falhados
  - [ ] Fallback para comandos alternativos

### ✅ Monitoramento
- [ ] Implementar:
  - [ ] Métricas de execução
  - [ ] Tempo de execução
  - [ ] Taxa de sucesso/falha
  - [ ] Comandos mais executados

## 🎯 Resultado Esperado
- Execução segura de comandos remotos
- Whitelist configurável
- Tratamento robusto de erros
- Limitação de recursos
- Monitoramento de execução

## 🔗 Próxima Task
`08-integration-tests.md` - Testes de integração

## 📝 Notas
- Segurança é prioridade máxima
- Nunca executar comandos não listados
- Validar todos os argumentos
- Logs devem incluir usuário/origem
- Preparar para auditoria de segurança
- Considerar sandboxing no futuro 