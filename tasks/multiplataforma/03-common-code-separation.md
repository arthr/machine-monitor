# Task 03: Separar Código Comum das Implementações Específicas

## 📋 Objetivo
Identificar e separar o código comum que pode ser reutilizado entre plataformas, evitando duplicação e centralizando funcionalidades multiplataforma.

## 🎯 Entregáveis
- [ ] Funções comuns identificadas e centralizadas
- [ ] Utilitários multiplataforma organizados
- [ ] Código duplicado eliminado
- [ ] Interface `PlatformInfo` implementada

## 📊 Contexto
Com as interfaces definidas e build tags implementados, precisamos identificar o código que pode ser compartilhado entre plataformas e organizá-lo adequadamente.

## 🔧 Implementação

### 1. Expandir `internal/collector/common.go`
```go
package collector

import (
    "context"
    "fmt"
    "runtime"
    "strings"
    "time"
    
    "github.com/shirou/gopsutil/v3/host"
    "github.com/shirou/gopsutil/v3/cpu"
    "github.com/shirou/gopsutil/v3/mem"
    "github.com/shirou/gopsutil/v3/disk"
    "github.com/shirou/gopsutil/v3/net"
)

// GetBasicPlatformInfo coleta informações básicas usando gopsutil
func GetBasicPlatformInfo(ctx context.Context) (*PlatformInfo, error) {
    info, err := host.InfoWithContext(ctx)
    if err != nil {
        return nil, err
    }
    
    return &PlatformInfo{
        OS:           runtime.GOOS,
        Architecture: runtime.GOARCH,
        Version:      info.KernelVersion,
        Hostname:     info.Hostname,
        Uptime:       time.Duration(info.Uptime) * time.Second,
        Platform:     info.Platform,
    }, nil
}

// GetSystemStats coleta estatísticas básicas do sistema
func GetSystemStats(ctx context.Context) (*SystemStats, error) {
    stats := &SystemStats{}
    
    // CPU
    if cpuPercent, err := cpu.PercentWithContext(ctx, time.Second, false); err == nil && len(cpuPercent) > 0 {
        stats.CPUUsage = cpuPercent[0]
    }
    
    // Memory
    if memInfo, err := mem.VirtualMemoryWithContext(ctx); err == nil {
        stats.MemoryUsage = memInfo.UsedPercent
        stats.MemoryTotal = memInfo.Total
        stats.MemoryUsed = memInfo.Used
    }
    
    // Disk
    if diskInfo, err := disk.UsageWithContext(ctx, "/"); err == nil {
        stats.DiskUsage = diskInfo.UsedPercent
        stats.DiskTotal = diskInfo.Total
        stats.DiskUsed = diskInfo.Used
    }
    
    return stats, nil
}

// GetNetworkInterfaces coleta informações de rede
func GetNetworkInterfaces(ctx context.Context) ([]NetworkInterface, error) {
    interfaces, err := net.InterfacesWithContext(ctx)
    if err != nil {
        return nil, err
    }
    
    var result []NetworkInterface
    for _, iface := range interfaces {
        result = append(result, NetworkInterface{
            Name:         iface.Name,
            HardwareAddr: iface.HardwareAddr,
            Flags:        iface.Flags,
            Addrs:        convertAddrs(iface.Addrs),
        })
    }
    
    return result, nil
}

// Funções de validação e sanitização
func SanitizeApplicationName(name string) string {
    // Remove caracteres especiais e normaliza
    name = strings.TrimSpace(name)
    name = strings.ReplaceAll(name, "\n", "")
    name = strings.ReplaceAll(name, "\r", "")
    name = strings.ReplaceAll(name, "\t", " ")
    
    // Remove múltiplos espaços
    for strings.Contains(name, "  ") {
        name = strings.ReplaceAll(name, "  ", " ")
    }
    
    return name
}

func ValidateService(service *Service) bool {
    return service.Name != "" && service.Status != ""
}

func ValidateApplication(app *Application) bool {
    return app.Name != "" && app.Name != "Unknown"
}

// Funções de conversão e formatação
func FormatFileSize(bytes int64) string {
    const unit = 1024
    if bytes < unit {
        return fmt.Sprintf("%d B", bytes)
    }
    div, exp := int64(unit), 0
    for n := bytes / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func FormatDuration(d time.Duration) string {
    if d < time.Minute {
        return fmt.Sprintf("%.0fs", d.Seconds())
    }
    if d < time.Hour {
        return fmt.Sprintf("%.0fm", d.Minutes())
    }
    if d < 24*time.Hour {
        return fmt.Sprintf("%.0fh", d.Hours())
    }
    return fmt.Sprintf("%.0fd", d.Hours()/24)
}

// Funções de fallback para Machine ID
func GenerateFallbackMachineID() (string, error) {
    // Usar hostname + MAC address como fallback
    hostname, err := host.Info()
    if err != nil {
        return "", err
    }
    
    interfaces, err := net.Interfaces()
    if err != nil {
        return "", err
    }
    
    var macAddr string
    for _, iface := range interfaces {
        if iface.HardwareAddr != "" && iface.Name != "lo" {
            macAddr = iface.HardwareAddr
            break
        }
    }
    
    if macAddr == "" {
        macAddr = "unknown"
    }
    
    return fmt.Sprintf("fallback-%s-%s", hostname.Hostname, macAddr), nil
}

// Funções auxiliares para conversão
func convertAddrs(addrs []net.Addr) []string {
    var result []string
    for _, addr := range addrs {
        result = append(result, addr.Addr)
    }
    return result
}
```

### 2. Criar `internal/collector/validation.go`
```go
package collector

import (
    "regexp"
    "strings"
)

// Validadores específicos por tipo de dados
var (
    // Regex para validar nomes de aplicações
    appNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\s\-\._]+$`)
    
    // Regex para validar nomes de serviços
    serviceNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\-\._]+$`)
    
    // Lista de nomes de aplicações inválidas
    invalidAppNames = []string{
        "",
        "Unknown",
        "N/A",
        "null",
        "undefined",
    }
)

// ValidateApplicationData valida dados de aplicação
func ValidateApplicationData(app *Application) error {
    if app.Name == "" {
        return fmt.Errorf("application name cannot be empty")
    }
    
    // Verificar nomes inválidos
    for _, invalid := range invalidAppNames {
        if strings.EqualFold(app.Name, invalid) {
            return fmt.Errorf("invalid application name: %s", app.Name)
        }
    }
    
    // Validar formato do nome
    if !appNameRegex.MatchString(app.Name) {
        return fmt.Errorf("invalid application name format: %s", app.Name)
    }
    
    return nil
}

// ValidateServiceData valida dados de serviço
func ValidateServiceData(service *Service) error {
    if service.Name == "" {
        return fmt.Errorf("service name cannot be empty")
    }
    
    if !serviceNameRegex.MatchString(service.Name) {
        return fmt.Errorf("invalid service name format: %s", service.Name)
    }
    
    // Validar status
    validStatuses := []string{"running", "stopped", "disabled", "unknown"}
    validStatus := false
    for _, status := range validStatuses {
        if strings.EqualFold(service.Status, status) {
            validStatus = true
            break
        }
    }
    
    if !validStatus {
        return fmt.Errorf("invalid service status: %s", service.Status)
    }
    
    return nil
}

// SanitizeAndValidateData sanitiza e valida dados coletados
func SanitizeAndValidateData(info *SystemInfo) error {
    // Sanitizar aplicações
    var validApps []Application
    for _, app := range info.Applications {
        app.Name = SanitizeApplicationName(app.Name)
        if err := ValidateApplicationData(&app); err == nil {
            validApps = append(validApps, app)
        }
    }
    info.Applications = validApps
    
    // Sanitizar serviços
    var validServices []Service
    for _, service := range info.Services {
        service.Name = strings.TrimSpace(service.Name)
        if err := ValidateServiceData(&service); err == nil {
            validServices = append(validServices, service)
        }
    }
    info.Services = validServices
    
    return nil
}
```

### 3. Criar `internal/collector/cache.go`
```go
package collector

import (
    "sync"
    "time"
)

// Cache para dados que não mudam frequentemente
type DataCache struct {
    mu              sync.RWMutex
    applications    []Application
    appsCacheTime   time.Time
    services        []Service
    servicesCacheTime time.Time
    platformInfo    *PlatformInfo
    platformCacheTime time.Time
    
    // Configurações de cache
    appsCacheTTL      time.Duration
    servicesCacheTTL  time.Duration
    platformCacheTTL  time.Duration
}

func NewDataCache() *DataCache {
    return &DataCache{
        appsCacheTTL:     30 * time.Minute, // Apps não mudam frequentemente
        servicesCacheTTL: 5 * time.Minute,  // Serviços podem mudar
        platformCacheTTL: 60 * time.Minute, // Info da plataforma é estática
    }
}

func (c *DataCache) GetApplications() ([]Application, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    if time.Since(c.appsCacheTime) > c.appsCacheTTL {
        return nil, false
    }
    
    return c.applications, true
}

func (c *DataCache) SetApplications(apps []Application) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.applications = apps
    c.appsCacheTime = time.Now()
}

func (c *DataCache) GetServices() ([]Service, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    if time.Since(c.servicesCacheTime) > c.servicesCacheTTL {
        return nil, false
    }
    
    return c.services, true
}

func (c *DataCache) SetServices(services []Service) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.services = services
    c.servicesCacheTime = time.Now()
}

func (c *DataCache) GetPlatformInfo() (*PlatformInfo, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    if time.Since(c.platformCacheTime) > c.platformCacheTTL {
        return nil, false
    }
    
    return c.platformInfo, true
}

func (c *DataCache) SetPlatformInfo(info *PlatformInfo) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.platformInfo = info
    c.platformCacheTime = time.Now()
}

func (c *DataCache) Clear() {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.applications = nil
    c.services = nil
    c.platformInfo = nil
    c.appsCacheTime = time.Time{}
    c.servicesCacheTime = time.Time{}
    c.platformCacheTime = time.Time{}
}
```

### 4. Atualizar `internal/collector/types.go`
```go
// Adicionar novas estruturas comuns
type SystemStats struct {
    CPUUsage    float64 `json:"cpu_usage"`
    MemoryUsage float64 `json:"memory_usage"`
    MemoryTotal uint64  `json:"memory_total"`
    MemoryUsed  uint64  `json:"memory_used"`
    DiskUsage   float64 `json:"disk_usage"`
    DiskTotal   uint64  `json:"disk_total"`
    DiskUsed    uint64  `json:"disk_used"`
}

type NetworkInterface struct {
    Name         string   `json:"name"`
    HardwareAddr string   `json:"hardware_addr"`
    Flags        []string `json:"flags"`
    Addrs        []string `json:"addrs"`
}

// Atualizar SystemInfo
type SystemInfo struct {
    // ... campos existentes ...
    
    // Novos campos multiplataforma
    Platform     *PlatformInfo     `json:"platform"`
    Applications []Application     `json:"applications,omitempty"`
    Services     []Service         `json:"services,omitempty"`
    Stats        *SystemStats      `json:"stats,omitempty"`
    Network      []NetworkInterface `json:"network,omitempty"`
    Specific     map[string]interface{} `json:"platform_specific,omitempty"`
}
```

## 📋 Checklist de Implementação

### Arquivos a Criar
- [ ] `internal/collector/validation.go` - Validadores de dados
- [ ] `internal/collector/cache.go` - Sistema de cache

### Arquivos a Expandir
- [ ] `internal/collector/common.go` - Funções comuns expandidas
- [ ] `internal/collector/types.go` - Novas estruturas

### Funcionalidades Comuns
- [ ] Coleta de informações básicas via gopsutil
- [ ] Validação e sanitização de dados
- [ ] Sistema de cache para dados estáticos
- [ ] Formatação e conversão de dados
- [ ] Geração de Machine ID fallback

### Validações
- [ ] Funções comuns funcionam em todas as plataformas
- [ ] Cache funciona corretamente
- [ ] Validação de dados está robusta
- [ ] Não há duplicação de código

## 🎯 Critérios de Sucesso
- [ ] Código comum centralizado e reutilizável
- [ ] Validação robusta de dados
- [ ] Sistema de cache eficiente
- [ ] Eliminação de duplicação entre plataformas

## 📚 Referências
- [gopsutil Examples](https://github.com/shirou/gopsutil/tree/master/_examples) - Exemplos de uso
- [Go Validation Patterns](https://blog.golang.org/go-slices-usage-and-internals) - Padrões de validação
- [Caching Strategies](https://github.com/patrickmn/go-cache) - Estratégias de cache

## ⏭️ Próxima Task
[04-platform-factory.md](04-platform-factory.md) - Implementar factory pattern para criação de collectors 