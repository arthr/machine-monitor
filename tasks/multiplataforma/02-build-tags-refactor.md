# Task 02: Implementar Build Tags e Refatoração

## 📋 Objetivo
Implementar build tags (build constraints) para permitir compilação condicional por plataforma, separando o código específico do macOS, Windows e Linux.

## 🎯 Entregáveis
- [ ] Build tags configurados para cada plataforma
- [ ] Código atual refatorado para usar build tags
- [ ] Estrutura de arquivos reorganizada por plataforma
- [ ] Compilação condicional funcionando

## 📊 Contexto
O código atual mistura implementações específicas do macOS. Precisamos separar esse código em arquivos específicos por plataforma usando build tags do Go.

## 🔧 Implementação

### 1. Criar `internal/collector/platform_darwin.go`
```go
//go:build darwin
// +build darwin

package collector

import (
    "context"
    "os/exec"
    "strings"
    "encoding/json"
)

// DarwinCollector implementa PlatformCollector para macOS
type DarwinCollector struct {
    logger logging.Logger
    config *CollectorConfig
}

// NewPlatformCollector cria um collector específico para macOS
func NewPlatformCollector(logger logging.Logger, config *CollectorConfig) PlatformCollector {
    return &DarwinCollector{
        logger: logger,
        config: config,
    }
}

func (d *DarwinCollector) GetMachineID(ctx context.Context) (string, error) {
    // Implementação usando system_profiler e ioreg
    cmd := exec.CommandContext(ctx, "system_profiler", "SPHardwareDataType", "-json")
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    
    // Parse JSON e extrair UUID
    var data map[string]interface{}
    if err := json.Unmarshal(output, &data); err != nil {
        return "", err
    }
    
    // Extrair UUID do hardware
    // Implementar lógica específica
    return "mac-" + extractUUID(data), nil
}

func (d *DarwinCollector) CollectInstalledApps(ctx context.Context) ([]Application, error) {
    // Implementação usando /Applications e system_profiler
    apps := []Application{}
    
    // Scan /Applications
    systemApps, err := d.scanApplicationsFolder("/Applications")
    if err == nil {
        apps = append(apps, systemApps...)
    }
    
    // Scan ~/Applications
    userApps, err := d.scanApplicationsFolder("~/Applications")
    if err == nil {
        apps = append(apps, userApps...)
    }
    
    return apps, nil
}

func (d *DarwinCollector) CollectSystemServices(ctx context.Context) ([]Service, error) {
    // Implementação usando launchctl
    cmd := exec.CommandContext(ctx, "launchctl", "list")
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    
    return d.parseLaunchctlOutput(string(output)), nil
}

func (d *DarwinCollector) CollectPlatformSpecific(ctx context.Context) (map[string]interface{}, error) {
    specific := make(map[string]interface{})
    
    // System Profiler data
    if profilerData, err := d.getSystemProfilerData(ctx); err == nil {
        specific["system_profiler"] = profilerData
    }
    
    // Brew packages
    if brewData, err := d.getBrewPackages(ctx); err == nil {
        specific["brew_packages"] = brewData
    }
    
    return specific, nil
}

// Funções auxiliares específicas do macOS
func (d *DarwinCollector) scanApplicationsFolder(path string) ([]Application, error) {
    // Implementar scan de .app bundles
    return []Application{}, nil
}

func (d *DarwinCollector) parseLaunchctlOutput(output string) []Service {
    // Implementar parse da saída do launchctl
    return []Service{}
}

func (d *DarwinCollector) getSystemProfilerData(ctx context.Context) (map[string]interface{}, error) {
    // Implementar coleta via system_profiler
    return map[string]interface{}{}, nil
}

func (d *DarwinCollector) getBrewPackages(ctx context.Context) ([]map[string]interface{}, error) {
    // Implementar coleta de packages do Homebrew
    return []map[string]interface{}{}, nil
}
```

### 2. Criar `internal/collector/platform_windows.go`
```go
//go:build windows
// +build windows

package collector

import (
    "context"
    "github.com/go-ole/go-ole"
    "github.com/go-ole/go-ole/oleutil"
)

// WindowsCollector implementa PlatformCollector para Windows
type WindowsCollector struct {
    logger logging.Logger
    config *CollectorConfig
}

// NewPlatformCollector cria um collector específico para Windows
func NewPlatformCollector(logger logging.Logger, config *CollectorConfig) PlatformCollector {
    return &WindowsCollector{
        logger: logger,
        config: config,
    }
}

func (w *WindowsCollector) GetMachineID(ctx context.Context) (string, error) {
    // Implementação usando WMI
    uuid, err := w.getWMIValue("SELECT UUID FROM Win32_ComputerSystemProduct")
    if err != nil {
        return "", err
    }
    return "win-" + uuid, nil
}

func (w *WindowsCollector) CollectInstalledApps(ctx context.Context) ([]Application, error) {
    // Implementação usando Registry + WMI
    apps := []Application{}
    
    // Registry scan
    regApps, err := w.getAppsFromRegistry()
    if err == nil {
        apps = append(apps, regApps...)
    }
    
    // WMI scan
    wmiApps, err := w.getAppsFromWMI()
    if err == nil {
        apps = append(apps, wmiApps...)
    }
    
    return apps, nil
}

func (w *WindowsCollector) CollectSystemServices(ctx context.Context) ([]Service, error) {
    // Implementação usando WMI Win32_Service
    return w.getServicesFromWMI()
}

func (w *WindowsCollector) CollectPlatformSpecific(ctx context.Context) (map[string]interface{}, error) {
    specific := make(map[string]interface{})
    
    // SystemInfo
    if sysInfo, err := w.getSystemInfo(ctx); err == nil {
        specific["system_info"] = sysInfo
    }
    
    // Windows Features
    if features, err := w.getWindowsFeatures(ctx); err == nil {
        specific["windows_features"] = features
    }
    
    return specific, nil
}

// Funções auxiliares específicas do Windows
func (w *WindowsCollector) getWMIValue(query string) (string, error) {
    // Implementar query WMI
    return "", nil
}

func (w *WindowsCollector) getAppsFromRegistry() ([]Application, error) {
    // Implementar scan do Registry
    return []Application{}, nil
}

func (w *WindowsCollector) getAppsFromWMI() ([]Application, error) {
    // Implementar scan via WMI
    return []Application{}, nil
}

func (w *WindowsCollector) getServicesFromWMI() ([]Service, error) {
    // Implementar coleta de serviços via WMI
    return []Service{}, nil
}
```

### 3. Criar `internal/collector/platform_linux.go`
```go
//go:build linux
// +build linux

package collector

import (
    "context"
    "os/exec"
)

// LinuxCollector implementa PlatformCollector para Linux
type LinuxCollector struct {
    logger logging.Logger
    config *CollectorConfig
}

// NewPlatformCollector cria um collector específico para Linux
func NewPlatformCollector(logger logging.Logger, config *CollectorConfig) PlatformCollector {
    return &LinuxCollector{
        logger: logger,
        config: config,
    }
}

func (l *LinuxCollector) GetMachineID(ctx context.Context) (string, error) {
    // Implementação usando /etc/machine-id ou DMI
    return "linux-placeholder", nil
}

func (l *LinuxCollector) CollectInstalledApps(ctx context.Context) ([]Application, error) {
    // Implementação usando package managers (apt, yum, etc.)
    return []Application{}, nil
}

func (l *LinuxCollector) CollectSystemServices(ctx context.Context) ([]Service, error) {
    // Implementação usando systemctl
    return []Service{}, nil
}

func (l *LinuxCollector) CollectPlatformSpecific(ctx context.Context) (map[string]interface{}, error) {
    // Implementação específica do Linux
    return map[string]interface{}{}, nil
}
```

### 4. Atualizar `internal/collector/collector.go`
```go
package collector

import (
    "context"
    "time"
    
    "github.com/shirou/gopsutil/v3/cpu"
    "github.com/shirou/gopsutil/v3/mem"
    "github.com/shirou/gopsutil/v3/disk"
    "github.com/shirou/gopsutil/v3/net"
    "github.com/shirou/gopsutil/v3/process"
    
    "machine-monitor/internal/logging"
)

type Collector struct {
    logger            logging.Logger
    config            *CollectorConfig
    platformCollector PlatformCollector // Nova interface
}

func NewCollector(logger logging.Logger, config *CollectorConfig) *Collector {
    return &Collector{
        logger:            logger,
        config:            config,
        platformCollector: NewPlatformCollector(logger, config), // Factory function
    }
}

func (c *Collector) CollectSystemInfo(ctx context.Context) (*SystemInfo, error) {
    info := &SystemInfo{
        Timestamp: time.Now(),
        MachineID: c.config.MachineID,
    }
    
    // Coleta usando gopsutil (multiplataforma)
    if err := c.collectBasicInfo(ctx, info); err != nil {
        return nil, err
    }
    
    // Coleta específica da plataforma
    if platformInfo, err := c.platformCollector.GetPlatformInfo(ctx); err == nil {
        info.Platform = platformInfo
    }
    
    if apps, err := c.platformCollector.CollectInstalledApps(ctx); err == nil {
        info.Applications = apps
    }
    
    if services, err := c.platformCollector.CollectSystemServices(ctx); err == nil {
        info.Services = services
    }
    
    if specific, err := c.platformCollector.CollectPlatformSpecific(ctx); err == nil {
        info.Specific = specific
    }
    
    return info, nil
}

func (c *Collector) collectBasicInfo(ctx context.Context, info *SystemInfo) error {
    // Implementação usando gopsutil (código existente)
    // CPU, Memory, Disk, Network, Processes
    return nil
}
```

## 📋 Checklist de Implementação

### Arquivos a Criar
- [ ] `internal/collector/platform_darwin.go` - Implementação macOS
- [ ] `internal/collector/platform_windows.go` - Implementação Windows
- [ ] `internal/collector/platform_linux.go` - Implementação Linux

### Arquivos a Modificar
- [ ] `internal/collector/collector.go` - Usar interface PlatformCollector
- [ ] `internal/collector/types.go` - Adicionar novos campos

### Build Tags
- [ ] Testar compilação com `GOOS=darwin`
- [ ] Testar compilação com `GOOS=windows`
- [ ] Testar compilação com `GOOS=linux`

### Validações
- [ ] Código compila em todas as plataformas
- [ ] Factory function funciona corretamente
- [ ] Interfaces são implementadas corretamente
- [ ] Build tags funcionam como esperado

## 🎯 Critérios de Sucesso
- [ ] Compilação condicional funcionando
- [ ] Código separado por plataforma
- [ ] Interface unificada mantida
- [ ] Estrutura preparada para implementações específicas

## 📚 Referências
- [Go Build Constraints](https://pkg.go.dev/go/build#hdr-Build_Constraints) - Documentação oficial
- [Build Tags Tutorial](https://dave.cheney.net/2013/10/12/how-to-use-conditional-compilation-with-the-go-build-tool) - Tutorial detalhado
- [Platform Specific Code](https://golang.org/pkg/runtime/#pkg-constants) - Constantes de runtime

## ⏭️ Próxima Task
[03-common-code-separation.md](03-common-code-separation.md) - Separar código comum das implementações específicas 