# Task 04: Implementar Factory Pattern para Plataformas

## 📋 Objetivo
Implementar o padrão Factory para criação automática de collectors específicos por plataforma, centralizando a lógica de detecção e inicialização.

## 🎯 Entregáveis
- [ ] Factory function implementada
- [ ] Detecção automática de plataforma
- [ ] Configuração centralizada
- [ ] Testes de factory funcionando

## 📊 Contexto
Com as interfaces e implementações específicas prontas, precisamos de uma forma elegante de criar o collector correto baseado na plataforma atual, sem que o código cliente precise saber os detalhes.

## 🔧 Implementação

### 1. Criar `internal/collector/factory.go`
```go
package collector

import (
    "fmt"
    "runtime"
    
    "machine-monitor/internal/logging"
)

// PlatformType representa os tipos de plataforma suportados
type PlatformType string

const (
    PlatformDarwin  PlatformType = "darwin"
    PlatformWindows PlatformType = "windows"
    PlatformLinux   PlatformType = "linux"
    PlatformUnknown PlatformType = "unknown"
)

// CollectorFactory é responsável por criar collectors específicos
type CollectorFactory struct {
    logger logging.Logger
    cache  *DataCache
}

// NewCollectorFactory cria uma nova factory
func NewCollectorFactory(logger logging.Logger) *CollectorFactory {
    return &CollectorFactory{
        logger: logger,
        cache:  NewDataCache(),
    }
}

// CreatePlatformCollector cria um collector específico para a plataforma atual
func (f *CollectorFactory) CreatePlatformCollector(config *CollectorConfig) (PlatformCollector, error) {
    platform := f.DetectPlatform()
    
    f.logger.Info("Creating platform collector", map[string]interface{}{
        "platform": string(platform),
        "goos":     runtime.GOOS,
        "goarch":   runtime.GOARCH,
    })
    
    switch platform {
    case PlatformDarwin:
        return f.createDarwinCollector(config)
    case PlatformWindows:
        return f.createWindowsCollector(config)
    case PlatformLinux:
        return f.createLinuxCollector(config)
    default:
        return nil, fmt.Errorf("unsupported platform: %s", platform)
    }
}

// DetectPlatform detecta a plataforma atual
func (f *CollectorFactory) DetectPlatform() PlatformType {
    switch runtime.GOOS {
    case "darwin":
        return PlatformDarwin
    case "windows":
        return PlatformWindows
    case "linux":
        return PlatformLinux
    default:
        f.logger.Warn("Unknown platform detected", map[string]interface{}{
            "goos": runtime.GOOS,
        })
        return PlatformUnknown
    }
}

// GetSupportedPlatforms retorna lista de plataformas suportadas
func (f *CollectorFactory) GetSupportedPlatforms() []PlatformType {
    return []PlatformType{
        PlatformDarwin,
        PlatformWindows,
        PlatformLinux,
    }
}

// IsPlatformSupported verifica se a plataforma é suportada
func (f *CollectorFactory) IsPlatformSupported(platform PlatformType) bool {
    supported := f.GetSupportedPlatforms()
    for _, p := range supported {
        if p == platform {
            return true
        }
    }
    return false
}

// createDarwinCollector cria collector para macOS
func (f *CollectorFactory) createDarwinCollector(config *CollectorConfig) (PlatformCollector, error) {
    // Validações específicas do macOS
    if err := f.validateDarwinEnvironment(); err != nil {
        return nil, fmt.Errorf("macOS environment validation failed: %w", err)
    }
    
    collector := &DarwinCollector{
        logger: f.logger,
        config: config,
        cache:  f.cache,
    }
    
    return collector, nil
}

// createWindowsCollector cria collector para Windows
func (f *CollectorFactory) createWindowsCollector(config *CollectorConfig) (PlatformCollector, error) {
    // Validações específicas do Windows
    if err := f.validateWindowsEnvironment(); err != nil {
        return nil, fmt.Errorf("Windows environment validation failed: %w", err)
    }
    
    collector := &WindowsCollector{
        logger: f.logger,
        config: config,
        cache:  f.cache,
    }
    
    return collector, nil
}

// createLinuxCollector cria collector para Linux
func (f *CollectorFactory) createLinuxCollector(config *CollectorConfig) (PlatformCollector, error) {
    // Validações específicas do Linux
    if err := f.validateLinuxEnvironment(); err != nil {
        return nil, fmt.Errorf("Linux environment validation failed: %w", err)
    }
    
    collector := &LinuxCollector{
        logger: f.logger,
        config: config,
        cache:  f.cache,
    }
    
    return collector, nil
}

// Validações específicas por plataforma
func (f *CollectorFactory) validateDarwinEnvironment() error {
    // Verificar se comandos essenciais estão disponíveis
    requiredCommands := []string{"system_profiler", "launchctl", "ioreg"}
    return f.validateCommands(requiredCommands)
}

func (f *CollectorFactory) validateWindowsEnvironment() error {
    // Verificar se WMI está disponível
    // Verificar se comandos essenciais estão disponíveis
    requiredCommands := []string{"systeminfo", "wmic"}
    return f.validateCommands(requiredCommands)
}

func (f *CollectorFactory) validateLinuxEnvironment() error {
    // Verificar se comandos essenciais estão disponíveis
    requiredCommands := []string{"systemctl", "ps"}
    return f.validateCommands(requiredCommands)
}

func (f *CollectorFactory) validateCommands(commands []string) error {
    // Implementar validação de comandos disponíveis
    // Por enquanto, apenas log de aviso se comando não estiver disponível
    for _, cmd := range commands {
        if !f.isCommandAvailable(cmd) {
            f.logger.Warn("Required command not available", map[string]interface{}{
                "command": cmd,
            })
        }
    }
    return nil
}

func (f *CollectorFactory) isCommandAvailable(command string) bool {
    // Implementar verificação de disponibilidade de comando
    // Por enquanto, assumir que está disponível
    return true
}
```

### 2. Atualizar `internal/collector/collector.go`
```go
package collector

import (
    "context"
    "fmt"
    "time"
    
    "machine-monitor/internal/logging"
)

type Collector struct {
    logger            logging.Logger
    config            *CollectorConfig
    factory           *CollectorFactory
    platformCollector PlatformCollector
    cache             *DataCache
}

// NewCollector cria um novo collector usando factory pattern
func NewCollector(logger logging.Logger, config *CollectorConfig) (*Collector, error) {
    factory := NewCollectorFactory(logger)
    
    platformCollector, err := factory.CreatePlatformCollector(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create platform collector: %w", err)
    }
    
    return &Collector{
        logger:            logger,
        config:            config,
        factory:           factory,
        platformCollector: platformCollector,
        cache:             factory.cache,
    }, nil
}

// GetPlatformInfo retorna informações sobre a plataforma
func (c *Collector) GetPlatformInfo() map[string]interface{} {
    return map[string]interface{}{
        "platform":           string(c.factory.DetectPlatform()),
        "supported_platforms": c.factory.GetSupportedPlatforms(),
        "goos":               runtime.GOOS,
        "goarch":             runtime.GOARCH,
    }
}

// CollectSystemInfo coleta informações completas do sistema
func (c *Collector) CollectSystemInfo(ctx context.Context) (*SystemInfo, error) {
    startTime := time.Now()
    
    info := &SystemInfo{
        Timestamp: startTime,
        MachineID: c.config.MachineID,
    }
    
    // Coleta básica usando gopsutil (multiplataforma)
    if err := c.collectBasicInfo(ctx, info); err != nil {
        c.logger.Error("Failed to collect basic info", map[string]interface{}{
            "error": err.Error(),
        })
        return nil, err
    }
    
    // Coleta específica da plataforma
    if err := c.collectPlatformSpecificInfo(ctx, info); err != nil {
        c.logger.Warn("Failed to collect platform-specific info", map[string]interface{}{
            "error": err.Error(),
        })
        // Não retornar erro, dados básicos ainda são válidos
    }
    
    // Validação e sanitização final
    if err := SanitizeAndValidateData(info); err != nil {
        c.logger.Warn("Data validation issues", map[string]interface{}{
            "error": err.Error(),
        })
    }
    
    collectTime := time.Since(startTime)
    c.logger.Info("System info collection completed", map[string]interface{}{
        "duration_ms":      collectTime.Milliseconds(),
        "applications":     len(info.Applications),
        "services":         len(info.Services),
        "network_interfaces": len(info.Network),
    })
    
    return info, nil
}

// collectBasicInfo coleta informações básicas usando gopsutil
func (c *Collector) collectBasicInfo(ctx context.Context, info *SystemInfo) error {
    // Platform info
    if platformInfo, err := GetBasicPlatformInfo(ctx); err == nil {
        info.Platform = platformInfo
    }
    
    // System stats
    if stats, err := GetSystemStats(ctx); err == nil {
        info.Stats = stats
    }
    
    // Network interfaces
    if network, err := GetNetworkInterfaces(ctx); err == nil {
        info.Network = network
    }
    
    return nil
}

// collectPlatformSpecificInfo coleta informações específicas da plataforma
func (c *Collector) collectPlatformSpecificInfo(ctx context.Context, info *SystemInfo) error {
    // Usar cache quando possível
    if apps, cached := c.cache.GetApplications(); cached {
        info.Applications = apps
    } else {
        if apps, err := c.platformCollector.CollectInstalledApps(ctx); err == nil {
            info.Applications = apps
            c.cache.SetApplications(apps)
        }
    }
    
    if services, cached := c.cache.GetServices(); cached {
        info.Services = services
    } else {
        if services, err := c.platformCollector.CollectSystemServices(ctx); err == nil {
            info.Services = services
            c.cache.SetServices(services)
        }
    }
    
    // Dados específicos da plataforma
    if specific, err := c.platformCollector.CollectPlatformSpecific(ctx); err == nil {
        info.Specific = specific
    }
    
    return nil
}

// RefreshCache limpa o cache forçando nova coleta
func (c *Collector) RefreshCache() {
    c.cache.Clear()
    c.logger.Info("Cache cleared")
}

// GetCacheStatus retorna status do cache
func (c *Collector) GetCacheStatus() map[string]interface{} {
    // Implementar status do cache
    return map[string]interface{}{
        "cache_enabled": true,
        "last_refresh": time.Now(),
    }
}
```

### 3. Criar `internal/collector/factory_test.go`
```go
package collector

import (
    "runtime"
    "testing"
    
    "machine-monitor/internal/logging"
)

func TestCollectorFactory(t *testing.T) {
    logger := logging.NewLogger("test", "info")
    factory := NewCollectorFactory(logger)
    
    t.Run("DetectPlatform", func(t *testing.T) {
        platform := factory.DetectPlatform()
        
        expectedPlatform := PlatformType(runtime.GOOS)
        switch runtime.GOOS {
        case "darwin":
            expectedPlatform = PlatformDarwin
        case "windows":
            expectedPlatform = PlatformWindows
        case "linux":
            expectedPlatform = PlatformLinux
        default:
            expectedPlatform = PlatformUnknown
        }
        
        if platform != expectedPlatform {
            t.Errorf("Expected platform %s, got %s", expectedPlatform, platform)
        }
    })
    
    t.Run("GetSupportedPlatforms", func(t *testing.T) {
        platforms := factory.GetSupportedPlatforms()
        
        if len(platforms) != 3 {
            t.Errorf("Expected 3 supported platforms, got %d", len(platforms))
        }
        
        expectedPlatforms := []PlatformType{
            PlatformDarwin,
            PlatformWindows,
            PlatformLinux,
        }
        
        for _, expected := range expectedPlatforms {
            found := false
            for _, platform := range platforms {
                if platform == expected {
                    found = true
                    break
                }
            }
            if !found {
                t.Errorf("Expected platform %s not found in supported platforms", expected)
            }
        }
    })
    
    t.Run("IsPlatformSupported", func(t *testing.T) {
        testCases := []struct {
            platform PlatformType
            expected bool
        }{
            {PlatformDarwin, true},
            {PlatformWindows, true},
            {PlatformLinux, true},
            {PlatformUnknown, false},
            {PlatformType("invalid"), false},
        }
        
        for _, tc := range testCases {
            result := factory.IsPlatformSupported(tc.platform)
            if result != tc.expected {
                t.Errorf("IsPlatformSupported(%s) = %v, expected %v", tc.platform, result, tc.expected)
            }
        }
    })
    
    t.Run("CreatePlatformCollector", func(t *testing.T) {
        config := &CollectorConfig{
            MachineID: "test-machine",
        }
        
        collector, err := factory.CreatePlatformCollector(config)
        if err != nil {
            t.Errorf("Failed to create platform collector: %v", err)
        }
        
        if collector == nil {
            t.Error("Expected non-nil collector")
        }
    })
}

func TestCollectorCreation(t *testing.T) {
    logger := logging.NewLogger("test", "info")
    config := &CollectorConfig{
        MachineID: "test-machine",
    }
    
    collector, err := NewCollector(logger, config)
    if err != nil {
        t.Errorf("Failed to create collector: %v", err)
    }
    
    if collector == nil {
        t.Error("Expected non-nil collector")
    }
    
    // Testar informações da plataforma
    platformInfo := collector.GetPlatformInfo()
    if platformInfo["platform"] == nil {
        t.Error("Expected platform info to contain platform")
    }
    
    if platformInfo["goos"] != runtime.GOOS {
        t.Errorf("Expected GOOS %s, got %s", runtime.GOOS, platformInfo["goos"])
    }
}
```

## 📋 Checklist de Implementação

### Arquivos a Criar
- [ ] `internal/collector/factory.go` - Factory principal
- [ ] `internal/collector/factory_test.go` - Testes da factory

### Arquivos a Modificar
- [ ] `internal/collector/collector.go` - Usar factory pattern
- [ ] `internal/collector/platform_*.go` - Adicionar campo cache

### Funcionalidades
- [ ] Detecção automática de plataforma
- [ ] Criação de collector específico
- [ ] Validação de ambiente
- [ ] Sistema de cache integrado
- [ ] Informações de plataforma

### Validações
- [ ] Factory cria collector correto para cada plataforma
- [ ] Detecção de plataforma funciona corretamente
- [ ] Validações de ambiente são executadas
- [ ] Cache é compartilhado entre componentes
- [ ] Testes passam em todas as plataformas

## 🎯 Critérios de Sucesso
- [ ] Factory pattern implementado corretamente
- [ ] Criação automática de collectors
- [ ] Validação robusta de ambiente
- [ ] Código limpo e testável

## 📚 Referências
- [Factory Pattern](https://refactoring.guru/design-patterns/factory-method) - Padrão de design
- [Go Factory Pattern](https://golang.org/doc/effective_go.html#constructors) - Implementação em Go
- [Runtime Package](https://golang.org/pkg/runtime/) - Detecção de plataforma

## ⏭️ Próxima Task
[05-wmi-integration.md](05-wmi-integration.md) - Integração com WMI para Windows 