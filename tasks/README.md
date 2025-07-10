# 📋 Tasks do Agente macOS POC

Esta é a trilha de desenvolvimento modular para a POC do agente macOS.

## 🎯 Objetivo Geral
Desenvolver uma POC funcional do agente de monitoramento para macOS que demonstre:
- Coleta de inventário do sistema
- Comunicação HTTP + WebSocket com backend
- Execução segura de comandos remotos
- Reconnect automático e robustez

## 📚 Trilha de Tasks

### 🏗️ **Fase 1: Configuração Base**
- **[Task 01](01-setup-project.md)** - Setup do Projeto
  - Estrutura de diretórios
  - Configuração Go + dependências
  - Arquivos base

- **[Task 02](02-config-and-types.md)** - Configuração e Tipos
  - Sistema de configuração
  - Tipos de dados principais
  - Logger básico

### 🔧 **Fase 2: Componentes Principais**
- **[Task 03](03-main-entry.md)** - Ponto de Entrada
  - main.go com ciclo de vida
  - Gerenciamento de sinais
  - Tratamento de erros

- **[Task 04](04-agent-loop.md)** - Loop Principal
  - Coordenação de componentes
  - Operações temporizadas
  - Gerenciamento de estado

### 📊 **Fase 3: Coleta e Comunicação**
- **[Task 05](05-collector-system.md)** - Coletor de Sistema
  - Coleta de dados macOS
  - Integração com gopsutil
  - Otimizações e cache

- **[Task 06](06-comms-manager.md)** - Gerenciador de Comunicação
  - Cliente HTTP + WebSocket
  - Reconnect automático
  - Autenticação e segurança

### 🛡️ **Fase 4: Execução e Segurança**
- **[Task 07](07-command-executor.md)** - Executor de Comandos
  - Whitelist de comandos
  - Execução segura
  - Limitação de recursos

### 🧪 **Fase 5: Validação e Documentação**
- **[Task 08](08-integration-tests.md)** - Testes de Integração
  - Testes completos
  - Validação de robustez
  - Métricas de performance

- **[Task 09](09-documentation.md)** - Documentação Final
  - Documentação completa
  - Packaging e distribuição
  - Validação final

## 🎯 Critérios de Sucesso

### ✅ **Funcionalidades Mínimas**
- [x] Backend de debug funcional
- [ ] Agente coleta dados do macOS
- [ ] Comunicação HTTP/WebSocket
- [ ] Comandos remotos seguros
- [ ] Reconnect automático
- [ ] Performance aceitável

### ✅ **Métricas de Performance**
- CPU: < 1% em idle
- RAM: < 20MB
- Startup: < 2 segundos
- Coleta: < 5 segundos

### ✅ **Cobertura de Testes**
- Testes unitários: > 80%
- Testes de integração: Principais fluxos
- Testes de robustez: Cenários críticos

## 🚀 Como Executar

### 1. Preparação
```bash
# Verificar se o backend está rodando
cd backend-debug
npm start

# Verificar se está funcionando
curl http://localhost:8080/debug/stats
```

### 2. Desenvolvimento
```bash
# Seguir as tasks em ordem
# Cada task tem checklist específico
# Marcar itens conforme desenvolvimento
```

### 3. Validação
```bash
# Após cada task, validar funcionalidade
# Executar testes relevantes
# Verificar integração com backend
```

## 📈 Progresso

| Task | Status | Descrição |
|------|--------|-----------|
| 01   | ⏳ Pendente | Setup do Projeto |
| 02   | ⏳ Pendente | Configuração e Tipos |
| 03   | ⏳ Pendente | Ponto de Entrada |
| 04   | ⏳ Pendente | Loop Principal |
| 05   | ⏳ Pendente | Coletor de Sistema |
| 06   | ⏳ Pendente | Gerenciador de Comunicação |
| 07   | ⏳ Pendente | Executor de Comandos |
| 08   | ⏳ Pendente | Testes de Integração |
| 09   | ⏳ Pendente | Documentação Final |

## 🔗 Recursos

- **Backend**: `../backend-debug/`
- **Documentação**: `../docs/POC_AGENTE_MACOS.md`
- **Arquitetura**: `../ARQUITETURA_AGENTE.md`

## 📝 Notas

- Cada task é independente e pode ser desenvolvida incrementalmente
- Testes devem ser executados após cada task
- Integração com backend deve ser validada continuamente
- Documentação deve ser atualizada conforme progresso

---

**Pronto para começar o desenvolvimento?** 🚀 