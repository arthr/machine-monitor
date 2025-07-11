# 🌐 Plano de Evolução Multiplataforma do Agente POC

## 📋 Análise da Situação Atual

### ✅ **Pontos Positivos - Já Multiplataforma**

O agente já possui uma base sólida multiplataforma graças às bibliotecas utilizadas:

#### **Bibliotecas Multiplataforma Utilizadas**
- **`github.com/shirou/gopsutil/v3`** ⭐ - **Excelente escolha!**
  - Suporte completo: Windows, macOS, Linux
  - APIs unificadas para CPU, memória, disco, rede, processos
  - Dependências específicas já incluídas:
    - `github.com/go-ole/go-ole` - Windows COM/OLE
    - `github.com/yusufpapurcu/wmi` - Windows WMI
    - `golang.org/x/sys` - Syscalls multiplataforma

- **`github.com/gorilla/websocket`** ✅ - Multiplataforma nativo
- **Go runtime padrão** ✅ - Multiplataforma nativo

#### **Código Já Preparado**
- **Executor**: Já tem whitelist específica para Windows
- **Estruturas de dados**: Genéricas e multiplataforma
- **Comunicação**: WebSocket e HTTP são universais
- **Logging**: Interface genérica

### ⚠️ **Pontos que Precisam de Adaptação**

#### **1. Collector - Comandos Específicos de Plataforma**

**Comandos macOS que precisam de equivalentes Windows:**
```bash
# macOS                    # Windows Equivalente
system_profiler         → systeminfo, wmic
launchctl list          → Get-Service (PowerShell), sc query
ioreg                   → wmic bios get serialnumber
/Applications           → Registry, Program Files
Info.plist              → Registry, file properties
brew                    → winget, chocolatey (opcional)
```

#### **2. Geração de Machine ID**
- **macOS**: `system_profiler` + `ioreg` para UUID do hardware
- **Windows**: WMI queries para UUID da motherboard

#### **3. Descoberta de Aplicações**
- **macOS**: `/Applications/*.app` + `Info.plist`
- **Windows**: Registry + Program Files + WMI

#### **4. Serviços do Sistema**
- **macOS**: `launchctl`
- **Windows**: Service Manager + WMI

## 🎯 **Estratégia de Implementação**

### **Fase 1: Refatoração para Suporte Multiplataforma (1-2 semanas)**

#### **1.1 Reestruturação do Collector**
```
internal/collector/
├── collector.go           # Interface e lógica comum
├── types.go              # Estruturas de dados (já OK)
├── common.go             # Funções compartilhadas
├── platform_darwin.go   # Implementações macOS
├── platform_windows.go  # Implementações Windows  
├── platform_linux.go    # Implementações Linux
└── registry_windows.go   # Utilitários Windows Registry
```

#### **1.2 Interface Unificada**
```go
type PlatformCollector interface {
    CollectPlatformSpecific(ctx context.Context) (*PlatformInfo, error)
    GetMachineID(ctx context.Context) (string, error)
    CollectInstalledApps(ctx context.Context) ([]Application, error)
    CollectSystemServices(ctx context.Context) ([]Service, error)
}
```

### **Fase 2: Implementação Windows (2-3 semanas)**

#### **2.1 Machine ID no Windows**
```go
// Usar WMI para obter UUID único
func (w *WindowsCollector) GetMachineID(ctx context.Context) (string, error) {
    // Opção 1: Motherboard UUID
    uuid := getWMIValue("SELECT UUID FROM Win32_ComputerSystemProduct")
    
    // Opção 2: BIOS Serial Number  
    if uuid == "" {
        uuid = getWMIValue("SELECT SerialNumber FROM Win32_BIOS")
    }
    
    // Opção 3: Fallback para hostname + MAC address
    return generateFallbackID()
}
```

#### **2.2 Descoberta de Aplicações Windows**
```go
func (w *WindowsCollector) CollectInstalledApps(ctx context.Context) ([]Application, error) {
    var apps []Application
    
    // Método 1: Registry (Uninstall keys)
    apps = append(apps, w.getAppsFromRegistry()...)
    
    // Método 2: WMI Win32_Product (mais lento, mas completo)
    apps = append(apps, w.getAppsFromWMI()...)
    
    // Método 3: Program Files scan
    apps = append(apps, w.getAppsFromProgramFiles()...)
    
    return apps, nil
}
```

#### **2.3 Serviços Windows**
```go
func (w *WindowsCollector) CollectSystemServices(ctx context.Context) ([]Service, error) {
    // Usar WMI Win32_Service
    return getWMIServices("SELECT * FROM Win32_Service")
}
```

#### **2.4 Informações Específicas do Windows**
```go
type WindowsInfo struct {
    SystemInfo      map[string]interface{} `json:"system_info"`
    WindowsServices []WindowsService       `json:"windows_services"`
    WindowsFeatures []WindowsFeature       `json:"windows_features,omitempty"`
    Registry        *RegistryInfo          `json:"registry,omitempty"`
    WinGet          *WinGetInfo           `json:"winget,omitempty"`
}
```

### **Fase 3: Otimização e Testes (1 semana)**

#### **3.1 Testes Multiplataforma**
```go
// Testes específicos por plataforma
func TestCollector_Windows(t *testing.T) { ... }
func TestCollector_macOS(t *testing.T) { ... }
func TestCollector_Linux(t *testing.T) { ... }
```

#### **3.2 Build Tags**
```go
//go:build windows
// +build windows

// platform_windows.go
```

## 🔧 **Implementação Detalhada - Windows**

### **1. Estrutura de Arquivos**

```go
// platform_windows.go
//go:build windows

package collector

import (
    "context"
    "fmt"
    "os/exec"
    "strings"
    "syscall"
    "unsafe"
    
    "github.com/go-ole/go-ole"
    "github.com/go-ole/go-ole/oleutil"
)

type WindowsCollector struct {
    logger logging.Logger
    config *CollectorConfig
}

func NewWindowsCollector(logger logging.Logger, config *CollectorConfig) *WindowsCollector {
    return &WindowsCollector{
        logger: logger,
        config: config,
    }
}
```

### **2. WMI Helper Functions**

```go
// wmi_windows.go
func (w *WindowsCollector) queryWMI(query string) ([]map[string]interface{}, error) {
    ole.CoInitialize(0)
    defer ole.CoUninitialize()
    
    unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
    if err != nil {
        return nil, err
    }
    defer unknown.Release()
    
    wmi, err := unknown.QueryInterface(ole.IID_IDispatch)
    if err != nil {
        return nil, err
    }
    defer wmi.Release()
    
    serviceRaw, err := oleutil.CallMethod(wmi, "ConnectServer")
    if err != nil {
        return nil, err
    }
    service := serviceRaw.ToIDispatch()
    defer service.Release()
    
    resultRaw, err := oleutil.CallMethod(service, "ExecQuery", query)
    if err != nil {
        return nil, err
    }
    result := resultRaw.ToIDispatch()
    defer result.Release()
    
    // Processar resultados...
    return parseWMIResults(result), nil
}
```

### **3. Registry Helper Functions**

```go
// registry_windows.go
//go:build windows

import "golang.org/x/sys/windows/registry"

func (w *WindowsCollector) getInstalledProgramsFromRegistry() ([]Application, error) {
    var apps []Application
    
    // Chaves do registro para programas instalados
    keys := []string{
        `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
        `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
    }
    
    for _, keyPath := range keys {
        key, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.ENUMERATE_SUB_KEYS)
        if err != nil {
            continue
        }
        defer key.Close()
        
        subkeys, err := key.ReadSubKeyNames(-1)
        if err != nil {
            continue
        }
        
        for _, subkey := range subkeys {
            app := w.getAppFromRegistryKey(keyPath + `\` + subkey)
            if app != nil {
                apps = append(apps, *app)
            }
        }
    }
    
    return apps, nil
}

func (w *WindowsCollector) getAppFromRegistryKey(keyPath string) *Application {
    key, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
    if err != nil {
        return nil
    }
    defer key.Close()
    
    name, _, err := key.GetStringValue("DisplayName")
    if err != nil {
        return nil
    }
    
    version, _, _ := key.GetStringValue("DisplayVersion")
    vendor, _, _ := key.GetStringValue("Publisher")
    installDate, _, _ := key.GetStringValue("InstallDate")
    
    return &Application{
        Name:        name,
        Version:     version,
        Vendor:      vendor,
        InstallDate: installDate,
        Path:        "", // Pode ser obtido de InstallLocation
    }
}
```

### **4. Machine ID Windows**

```go
func (w *WindowsCollector) GetMachineID(ctx context.Context) (string, error) {
    // Método 1: UUID da motherboard
    if uuid, err := w.getMotherboardUUID(); err == nil && uuid != "" {
        return "mb-" + uuid, nil
    }
    
    // Método 2: BIOS Serial Number
    if serial, err := w.getBIOSSerial(); err == nil && serial != "" {
        return "bios-" + serial, nil
    }
    
    // Método 3: Windows Product ID
    if productID, err := w.getWindowsProductID(); err == nil && productID != "" {
        return "win-" + productID, nil
    }
    
    // Fallback
    return w.generateFallbackID()
}

func (w *WindowsCollector) getMotherboardUUID() (string, error) {
    results, err := w.queryWMI("SELECT UUID FROM Win32_ComputerSystemProduct")
    if err != nil || len(results) == 0 {
        return "", err
    }
    
    if uuid, ok := results[0]["UUID"].(string); ok {
        return strings.ReplaceAll(uuid, "-", ""), nil
    }
    
    return "", fmt.Errorf("UUID not found")
}
```

## 📊 **Comandos Windows Equivalentes**

### **Mapeamento de Funcionalidades**

| **Funcionalidade** | **macOS** | **Windows** | **Implementação** |
|---|---|---|---|
| **Informações do Sistema** | `system_profiler` | `systeminfo` | WMI + systeminfo |
| **Hardware UUID** | `ioreg` | WMI Win32_ComputerSystemProduct | WMI query |
| **Serviços** | `launchctl` | `Get-Service` | WMI Win32_Service |
| **Processos** | `ps` | `tasklist` | gopsutil (já OK) |
| **Aplicações** | `/Applications` | Registry + Program Files | Registry scan |
| **Rede** | `ifconfig` | `ipconfig` | gopsutil (já OK) |
| **Versão do OS** | `sw_vers` | `ver` | WMI Win32_OperatingSystem |

### **Novos Comandos Windows no Executor**

```go
// Já implementado em commands.go!
"systeminfo": {...},
"tasklist": {...},
"netstat": {...},
"ipconfig": {...},
"wmic": {...},
"powershell": {...},
```

## 🚀 **Cronograma de Implementação**

### **Semana 1-2: Refatoração Base**
- [ ] Criar interfaces multiplataforma
- [ ] Separar código específico de plataforma
- [ ] Implementar build tags
- [ ] Testes básicos

### **Semana 3-4: Implementação Windows**
- [ ] WMI integration
- [ ] Registry scanning
- [ ] Machine ID Windows
- [ ] Windows-specific info

### **Semana 5: Testes e Otimização**
- [ ] Testes em Windows 10/11
- [ ] Performance tuning
- [ ] Documentação
- [ ] Build pipeline

## 🎯 **Benefícios da Abordagem**

### **1. Facilidade de Implementação**
- ✅ **gopsutil** já resolve 80% do trabalho
- ✅ **WMI** via go-ole para dados específicos Windows
- ✅ **Registry** via golang.org/x/sys/windows
- ✅ **Executor** já tem comandos Windows

### **2. Manutenibilidade**
- 🔧 Código separado por plataforma
- 🔧 Interfaces claras
- 🔧 Testes específicos por OS
- 🔧 Build tags para compilação condicional

### **3. Performance**
- ⚡ Cache compartilhado entre plataformas
- ⚡ Coleta paralela mantida
- ⚡ WMI otimizado para Windows
- ⚡ Registry scan eficiente

## 📋 **Checklist de Implementação**

### **Preparação**
- [ ] Análise detalhada das diferenças Windows vs macOS
- [ ] Setup ambiente de desenvolvimento Windows
- [ ] Testes das bibliotecas WMI

### **Desenvolvimento**
- [ ] Refatoração do collector para interfaces
- [ ] Implementação WindowsCollector
- [ ] WMI helpers e Registry helpers
- [ ] Machine ID para Windows
- [ ] Descoberta de aplicações Windows
- [ ] Serviços Windows via WMI

### **Testes**
- [ ] Testes unitários por plataforma
- [ ] Testes de integração Windows
- [ ] Comparação de dados macOS vs Windows
- [ ] Performance testing

### **Finalização**
- [ ] Documentação da API multiplataforma
- [ ] Guia de instalação Windows
- [ ] Build scripts para múltiplas plataformas
- [ ] CI/CD pipeline

## 🎉 **Conclusão**

A evolução para multiplataforma é **altamente viável** devido às excelentes escolhas arquiteturais iniciais:

- **gopsutil** elimina 80% do trabalho multiplataforma
- **Executor** já tem comandos Windows implementados
- **Estruturas de dados** já são genéricas
- **WMI integration** é straightforward com go-ole

**Estimativa total: 4-5 semanas** para implementação completa com testes.

O agente está muito bem posicionado para se tornar uma solução robusta multiplataforma! 🚀 