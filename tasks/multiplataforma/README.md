# 🌐 Tasks de Evolução Multiplataforma

## 📋 Visão Geral

Este diretório contém as tasks para evolução do agente POC para suporte multiplataforma, com foco inicial em Windows.

## 🎯 Objetivo

Transformar o agente POC (originalmente desenvolvido para macOS) em uma solução robusta multiplataforma que funcione nativamente em:
- ✅ **macOS** (já implementado)
- 🎯 **Windows** (foco principal)
- 🔄 **Linux** (suporte futuro)

## 📊 Status Atual

### ✅ **Já Multiplataforma (80%)**
- **Bibliotecas**: `gopsutil`, `gorilla/websocket` - multiplataforma nativo
- **Executor**: Whitelist Windows já implementada
- **Comunicação**: WebSocket/HTTP universais
- **Estruturas**: Tipos de dados genéricos

### 🔧 **Necessita Adaptação (20%)**
- **Machine ID**: Geração específica por plataforma
- **Descoberta de Apps**: Registry (Windows) vs /Applications (macOS)
- **Serviços**: WMI (Windows) vs launchctl (macOS)

## 📅 Cronograma (4-5 semanas)

### **Fase 1: Refatoração Base (Semanas 1-2)**
- [01-platform-interfaces.md](01-platform-interfaces.md) - Definir interfaces multiplataforma
- [02-build-tags-refactor.md](02-build-tags-refactor.md) - Implementar build tags
- [03-common-code-separation.md](03-common-code-separation.md) - Separar código comum
- [04-platform-factory.md](04-platform-factory.md) - Factory pattern para plataformas

### **Fase 2: Implementação Windows (Semanas 3-4)**
- [05-wmi-integration.md](05-wmi-integration.md) - Integração com WMI
- [06-registry-scanning.md](06-registry-scanning.md) - Scan do Registry Windows
- [07-windows-machine-id.md](07-windows-machine-id.md) - Machine ID para Windows
- [08-windows-services.md](08-windows-services.md) - Serviços Windows via WMI
- [09-windows-apps-discovery.md](09-windows-apps-discovery.md) - Descoberta de aplicações

### **Fase 3: Testes e Otimização (Semana 5)**
- [10-platform-tests.md](10-platform-tests.md) - Testes específicos por plataforma
- [11-integration-tests.md](11-integration-tests.md) - Testes de integração
- [12-performance-optimization.md](12-performance-optimization.md) - Otimização de performance
- [13-documentation.md](13-documentation.md) - Documentação final

## 🔧 Estrutura Final Esperada

```
internal/collector/
├── collector.go           # Interface e lógica comum
├── types.go              # Estruturas de dados (já OK)
├── interfaces.go         # Interfaces multiplataforma
├── common.go             # Funções compartilhadas
├── factory.go            # Factory pattern
├── platform_darwin.go   # Implementações macOS
├── platform_windows.go  # Implementações Windows  
├── platform_linux.go    # Implementações Linux
├── wmi_windows.go        # Utilitários WMI
└── registry_windows.go   # Utilitários Registry
```

## 📈 Métricas de Sucesso

- [ ] Compilação condicional por plataforma funcionando
- [ ] Machine ID único gerado em cada plataforma
- [ ] Descoberta de aplicações funcionando em Windows
- [ ] Serviços do sistema coletados via WMI
- [ ] Testes passando em múltiplas plataformas
- [ ] Performance mantida ou melhorada
- [ ] Documentação completa

## 🚀 Como Executar

1. **Preparação**: Configurar ambiente de desenvolvimento Windows
2. **Desenvolvimento**: Seguir as tasks em ordem
3. **Testes**: Validar cada implementação
4. **Integração**: Testar funcionamento completo

## 📚 Referências

- [PLANO_MULTIPLATAFORMA.md](../../agente-poc/PLANO_MULTIPLATAFORMA.md) - Plano detalhado
- [gopsutil Documentation](https://github.com/shirou/gopsutil) - Biblioteca multiplataforma
- [Windows WMI Reference](https://docs.microsoft.com/en-us/windows/win32/wmisdk/) - WMI APIs
- [Go Build Tags](https://pkg.go.dev/go/build#hdr-Build_Constraints) - Build constraints 