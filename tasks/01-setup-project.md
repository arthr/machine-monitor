# Task 01: Setup do Projeto

## 🎯 Objetivo
Configurar a estrutura inicial do projeto Go para o agente macOS.

## 📋 Checklist

### ✅ Estrutura de Diretórios
- [ ] Criar diretório `agente-poc/`
- [ ] Criar estrutura de pastas:
  ```
  agente-poc/
  ├── cmd/
  │   └── agente/
  ├── internal/
  │   ├── agent/
  │   ├── collector/
  │   ├── comms/
  │   ├── executor/
  │   └── logging/
  ├── configs/
  ```

### ✅ Configuração Go
- [ ] Inicializar módulo Go: `go mod init agente-poc`
- [ ] Instalar dependências principais:
  - [ ] `github.com/gorilla/websocket`
  - [ ] `github.com/shirou/gopsutil/v3`
- [ ] Criar go.mod com versão Go 1.21

### ✅ Arquivos Base
- [ ] Criar README.md inicial
- [ ] Criar .gitignore para projetos Go
- [ ] Criar config.json base

## 🎯 Resultado Esperado
- Estrutura de projeto organizada
- Dependências instaladas
- Pronto para desenvolvimento dos componentes

## 🔗 Próxima Task
`02-config-and-types.md` - Configuração e tipos de dados

## 📝 Notas
- Usar convenções Go padrão
- Estrutura modular para facilitar testes
- Configurações separadas por ambiente 