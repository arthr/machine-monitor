# Task 05: Coletor de Sistema

## 🎯 Objetivo
Implementar o coletor de informações do sistema macOS (hardware, software, rede, etc.).

## 📋 Checklist

### ✅ Interface Base
- [ ] Criar `internal/collector/collector.go` com:
  - [ ] Interface `Collector`
  - [ ] Struct `SystemCollector`
  - [ ] Função `New()` para criação
  - [ ] Método `CollectAll()` principal

### ✅ Coleta com gopsutil
- [ ] Implementar coleta usando gopsutil:
  - [ ] Informações do host (`host.Info()`)
  - [ ] Informações de CPU (`cpu.Info()`, `cpu.Percent()`)
  - [ ] Informações de memória (`mem.VirtualMemory()`)
  - [ ] Informações de disco (`disk.Usage()`)
  - [ ] Informações de rede (`net.Interfaces()`)

### ✅ Coleta Específica macOS
- [ ] Implementar `collectMacOSSpecific()` com:
  - [ ] `system_profiler SPHardwareDataType -json`
  - [ ] `system_profiler SPSoftwareDataType -json`
  - [ ] `sw_vers` para versão do sistema
  - [ ] Informações de energia (`pmset -g`)

### ✅ Aplicações Instaladas
- [ ] Implementar `getInstalledApps()`:
  - [ ] Listar `/Applications/*.app`
  - [ ] Extrair informações do bundle
  - [ ] Versões das aplicações
  - [ ] Última modificação

### ✅ Serviços em Execução
- [ ] Implementar `getRunningServices()`:
  - [ ] `launchctl list` para serviços
  - [ ] `ps aux` para processos
  - [ ] Filtrar processos relevantes
  - [ ] Status dos serviços

### ✅ Tratamento de Erros
- [ ] Implementar:
  - [ ] Timeout para comandos
  - [ ] Fallback para dados indisponíveis
  - [ ] Validação de dados coletados
  - [ ] Logs detalhados para debug

### ✅ Otimizações
- [ ] Implementar:
  - [ ] Cache para dados estáticos
  - [ ] Coleta incremental
  - [ ] Limitação de tamanho dos dados
  - [ ] Compressão de dados grandes

## 🎯 Resultado Esperado
- Coleta completa de informações do sistema
- Dados estruturados e validados
- Performance otimizada
- Tratamento robusto de erros
- Compatibilidade com diferentes versões do macOS

## 🔗 Próxima Task
`06-comms-manager.md` - Implementação do gerenciador de comunicação

## 📝 Notas
- Usar context.Context para cancelamento
- Implementar timeout para comandos externos
- Considerar permissões necessárias
- Dados sensíveis devem ser opcionais
- Preparar para expansão futura (outros coletores) 