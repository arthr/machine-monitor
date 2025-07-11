# Task 07: Implementar Machine ID para Windows

## 📋 Objetivo
Implementar geração robusta de Machine ID único para Windows usando múltiplas fontes (WMI, Registry, hardware) com fallbacks apropriados.

## 🎯 Entregáveis
- [ ] Gerador de Machine ID implementado
- [ ] Múltiplas fontes de identificação
- [ ] Sistema de fallback robusto
- [ ] Persistência e validação

## 📊 Contexto
O Machine ID é crucial para identificação única da máquina. Windows oferece várias fontes de identificação que devem ser combinadas para máxima confiabilidade.

## 🔧 Implementação

### 1. Atualizar `internal/collector/platform_windows.go`
```go
//go:build windows
// +build windows

package collector

import (
    "context"
    "crypto/sha256"
    "fmt"
    "strings"
    
    "machine-monitor/internal/logging"
)

// WindowsCollector implementa PlatformCollector para Windows
type WindowsCollector struct {
    logger        logging.Logger
    config        *CollectorConfig
    cache         *DataCache
    wmiHelper     *WMIHelper
    registryHelper *RegistryHelper
}

// NewPlatformCollector cria um collector específico para Windows
func NewPlatformCollector(logger logging.Logger, config *CollectorConfig) PlatformCollector {
    return &WindowsCollector{
        logger:         logger,
        config:         config,
        wmiHelper:      NewWMIHelper(logger),
        registryHelper: NewRegistryHelper(logger),
    }
}

// GetMachineID implementa geração robusta de Machine ID para Windows
func (w *WindowsCollector) GetMachineID(ctx context.Context) (string, error) {
    w.logger.Info("Generating Windows Machine ID")
    
    // Tentar múltiplas fontes em ordem de preferência
    sources := []struct {
        name string
        fn   func(context.Context) (string, error)
    }{
        {"motherboard_uuid", w.getMotherboardUUID},
        {"bios_serial", w.getBIOSSerial},
        {"machine_guid", w.getMachineGUID},
        {"system_uuid", w.getSystemUUID},
        {"hardware_hash", w.getHardwareHash},
        {"fallback", w.generateFallbackID},
    }
    
    var attempts []string
    for _, source := range sources {
        if id, err := source.fn(ctx); err == nil && id != "" {
            w.logger.Info("Machine ID generated successfully", map[string]interface{}{
                "source": source.name,
                "id":     w.maskID(id),
            })
            return id, nil
        } else {
            w.logger.Debug("Machine ID source failed", map[string]interface{}{
                "source": source.name,
                "error":  err,
            })
            attempts = append(attempts, fmt.Sprintf("%s: %v", source.name, err))
        }
    }
    
    return "", fmt.Errorf("failed to generate machine ID from any source: %s", strings.Join(attempts, "; "))
}

// getMotherboardUUID obtém UUID da motherboard via WMI
func (w *WindowsCollector) getMotherboardUUID(ctx context.Context) (string, error) {
    uuid, err := w.wmiHelper.GetSystemUUID(ctx)
    if err != nil {
        return "", fmt.Errorf("WMI motherboard UUID failed: %w", err)
    }
    
    if w.isValidUUID(uuid) {
        return fmt.Sprintf("mb-%s", w.normalizeUUID(uuid)), nil
    }
    
    return "", fmt.Errorf("invalid motherboard UUID format: %s", uuid)
}

// getBIOSSerial obtém serial number do BIOS
func (w *WindowsCollector) getBIOSSerial(ctx context.Context) (string, error) {
    serial, err := w.wmiHelper.getBIOSSerial(ctx)
    if err != nil {
        return "", fmt.Errorf("BIOS serial failed: %w", err)
    }
    
    if w.isValidSerial(serial) {
        return fmt.Sprintf("bios-%s", w.normalizeSerial(serial)), nil
    }
    
    return "", fmt.Errorf("invalid BIOS serial: %s", serial)
}

// getMachineGUID obtém Machine GUID do Registry
func (w *WindowsCollector) getMachineGUID(ctx context.Context) (string, error) {
    guid, err := w.registryHelper.GetMachineGUID(ctx)
    if err != nil {
        return "", fmt.Errorf("Registry machine GUID failed: %w", err)
    }
    
    if w.isValidGUID(guid) {
        return fmt.Sprintf("reg-%s", w.normalizeGUID(guid)), nil
    }
    
    return "", fmt.Errorf("invalid machine GUID: %s", guid)
}

// getSystemUUID obtém UUID do sistema via múltiplas fontes
func (w *WindowsCollector) getSystemUUID(ctx context.Context) (string, error) {
    // Tentar Computer System Product UUID
    if uuid, err := w.wmiHelper.getProductUUID(ctx); err == nil && uuid != "" {
        return fmt.Sprintf("sys-%s", w.normalizeUUID(uuid)), nil
    }
    
    // Tentar Base Board Serial Number
    if serial, err := w.wmiHelper.getMotherboardSerial(ctx); err == nil && serial != "" {
        return fmt.Sprintf("board-%s", w.normalizeSerial(serial)), nil
    }
    
    return "", fmt.Errorf("no valid system UUID found")
}

// getHardwareHash gera hash baseado em características do hardware
func (w *WindowsCollector) getHardwareHash(ctx context.Context) (string, error) {
    var components []string
    
    // CPU Information
    if cpuInfo, err := w.wmiHelper.getCPUInfo(ctx); err == nil {
        if name, ok := cpuInfo["Name"].(string); ok {
            components = append(components, fmt.Sprintf("cpu:%s", name))
        }
        if id, ok := cpuInfo["ProcessorId"].(string); ok {
            components = append(components, fmt.Sprintf("cpuid:%s", id))
        }
    }
    
    // Memory Information
    if memInfo, err := w.wmiHelper.getMemoryInfo(ctx); err == nil {
        if capacity, ok := memInfo["TotalCapacity"].(uint64); ok {
            components = append(components, fmt.Sprintf("mem:%d", capacity))
        }
    }
    
    // Disk Information
    if diskInfo, err := w.wmiHelper.getDiskInfo(ctx); err == nil {
        if serial, ok := diskInfo["SerialNumber"].(string); ok && serial != "" {
            components = append(components, fmt.Sprintf("disk:%s", serial))
        }
    }
    
    // Network MAC Address
    if netInfo, err := w.wmiHelper.getNetworkInfo(ctx); err == nil {
        if mac, ok := netInfo["MACAddress"].(string); ok && mac != "" {
            components = append(components, fmt.Sprintf("mac:%s", mac))
        }
    }
    
    if len(components) < 2 {
        return "", fmt.Errorf("insufficient hardware components for hash")
    }
    
    // Generate hash
    hashInput := strings.Join(components, "|")
    hash := sha256.Sum256([]byte(hashInput))
    return fmt.Sprintf("hw-%x", hash[:16]), nil
}

// generateFallbackID gera ID de fallback usando informações disponíveis
func (w *WindowsCollector) generateFallbackID(ctx context.Context) (string, error) {
    var parts []string
    
    // Computer name
    if name, err := w.registryHelper.getComputerName(); err == nil && name != "" {
        parts = append(parts, name)
    }
    
    // Windows install date
    if installDate, err := w.registryHelper.getWindowsInstallDate(); err == nil && installDate != "" {
        parts = append(parts, installDate)
    }
    
    // Current user SID
    if sid, err := w.getCurrentUserSID(); err == nil && sid != "" {
        parts = append(parts, sid)
    }
    
    // System root drive serial
    if serial, err := w.getSystemDriveSerial(); err == nil && serial != "" {
        parts = append(parts, serial)
    }
    
    if len(parts) == 0 {
        return "", fmt.Errorf("no fallback components available")
    }
    
    // Generate hash from available parts
    hashInput := strings.Join(parts, "|")
    hash := sha256.Sum256([]byte(hashInput))
    return fmt.Sprintf("fallback-%x", hash[:12]), nil
}

// Validation functions
func (w *WindowsCollector) isValidUUID(uuid string) bool {
    if len(uuid) < 32 {
        return false
    }
    
    // Check for placeholder values
    invalidUUIDs := []string{
        "00000000-0000-0000-0000-000000000000",
        "FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF",
        "12345678-1234-1234-1234-123456789012",
    }
    
    normalizedUUID := strings.ToUpper(strings.ReplaceAll(uuid, "-", ""))
    for _, invalid := range invalidUUIDs {
        if normalizedUUID == strings.ReplaceAll(invalid, "-", "") {
            return false
        }
    }
    
    return true
}

func (w *WindowsCollector) isValidSerial(serial string) bool {
    if len(serial) < 4 {
        return false
    }
    
    // Check for placeholder values
    invalidSerials := []string{
        "To Be Filled By O.E.M.",
        "Default string",
        "Not Specified",
        "System Serial Number",
        "0000000000",
        "1111111111",
        "XXXXXXXXXX",
    }
    
    for _, invalid := range invalidSerials {
        if strings.EqualFold(serial, invalid) {
            return false
        }
    }
    
    return true
}

func (w *WindowsCollector) isValidGUID(guid string) bool {
    return len(guid) >= 32 && !strings.Contains(strings.ToLower(guid), "default")
}

// Normalization functions
func (w *WindowsCollector) normalizeUUID(uuid string) string {
    return strings.ToLower(strings.ReplaceAll(uuid, "-", ""))
}

func (w *WindowsCollector) normalizeSerial(serial string) string {
    return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(serial), " ", ""))
}

func (w *WindowsCollector) normalizeGUID(guid string) string {
    return strings.ToLower(strings.ReplaceAll(guid, "-", ""))
}

// maskID masks sensitive parts of ID for logging
func (w *WindowsCollector) maskID(id string) string {
    if len(id) <= 8 {
        return "***"
    }
    return id[:4] + "***" + id[len(id)-4:]
}

// Helper functions for additional data sources
func (w *WindowsCollector) getCurrentUserSID() (string, error) {
    // Implementation to get current user SID
    return "", fmt.Errorf("not implemented")
}

func (w *WindowsCollector) getSystemDriveSerial() (string, error) {
    // Implementation to get system drive serial number
    return "", fmt.Errorf("not implemented")
}

// CollectInstalledApps implementa coleta de aplicações para Windows
func (w *WindowsCollector) CollectInstalledApps(ctx context.Context) ([]Application, error) {
    w.logger.Info("Collecting installed applications on Windows")
    
    var allApps []Application
    
    // Coletar via WMI
    if wmiApps, err := w.wmiHelper.GetInstalledApplications(ctx); err == nil {
        allApps = append(allApps, wmiApps...)
        w.logger.Debug("WMI applications collected", map[string]interface{}{
            "count": len(wmiApps),
        })
    } else {
        w.logger.Warn("Failed to collect WMI applications", map[string]interface{}{
            "error": err.Error(),
        })
    }
    
    // Coletar via Registry
    if regApps, err := w.registryHelper.GetInstalledPrograms(ctx); err == nil {
        allApps = append(allApps, regApps...)
        w.logger.Debug("Registry applications collected", map[string]interface{}{
            "count": len(regApps),
        })
    } else {
        w.logger.Warn("Failed to collect Registry applications", map[string]interface{}{
            "error": err.Error(),
        })
    }
    
    // Remover duplicatas e validar
    uniqueApps := w.removeDuplicateApplications(allApps)
    
    w.logger.Info("Application collection completed", map[string]interface{}{
        "total_found": len(allApps),
        "unique":      len(uniqueApps),
    })
    
    return uniqueApps, nil
}

// CollectSystemServices implementa coleta de serviços para Windows
func (w *WindowsCollector) CollectSystemServices(ctx context.Context) ([]Service, error) {
    w.logger.Info("Collecting system services on Windows")
    
    services, err := w.wmiHelper.GetSystemServices(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to collect Windows services: %w", err)
    }
    
    w.logger.Info("Service collection completed", map[string]interface{}{
        "count": len(services),
    })
    
    return services, nil
}

// CollectPlatformSpecific implementa coleta de dados específicos do Windows
func (w *WindowsCollector) CollectPlatformSpecific(ctx context.Context) (map[string]interface{}, error) {
    w.logger.Info("Collecting Windows-specific information")
    
    specific := make(map[string]interface{})
    
    // WMI system information
    if wmiInfo, err := w.wmiHelper.GetSystemInformation(ctx); err == nil {
        specific["wmi"] = wmiInfo
    }
    
    // Registry system information
    if regInfo, err := w.registryHelper.GetSystemInformation(ctx); err == nil {
        specific["registry"] = regInfo
    }
    
    // Windows features
    if features, err := w.getWindowsFeatures(ctx); err == nil {
        specific["features"] = features
    }
    
    return specific, nil
}

// Helper functions
func (w *WindowsCollector) removeDuplicateApplications(apps []Application) []Application {
    seen := make(map[string]bool)
    var unique []Application
    
    for _, app := range apps {
        key := fmt.Sprintf("%s|%s|%s", app.Name, app.Version, app.Vendor)
        if !seen[key] {
            seen[key] = true
            unique = append(unique, app)
        }
    }
    
    return unique
}

func (w *WindowsCollector) getWindowsFeatures(ctx context.Context) ([]map[string]interface{}, error) {
    // Implementation for Windows features collection
    return []map[string]interface{}{}, nil
}
```

## 📋 Checklist de Implementação

### Fontes de Machine ID
- [ ] Motherboard UUID via WMI
- [ ] BIOS Serial Number via WMI
- [ ] Machine GUID via Registry
- [ ] System UUID via múltiplas fontes WMI
- [ ] Hardware Hash baseado em componentes
- [ ] Fallback ID usando dados disponíveis

### Validação e Normalização
- [ ] Validação de UUIDs válidos
- [ ] Validação de seriais válidos
- [ ] Normalização de formatos
- [ ] Detecção de valores placeholder
- [ ] Mascaramento para logs

### Robustez
- [ ] Múltiplas fontes com fallback
- [ ] Tratamento de erros robusto
- [ ] Logging detalhado
- [ ] Performance otimizada

## 🎯 Critérios de Sucesso
- [ ] Machine ID único e persistente
- [ ] Múltiplas fontes funcionando
- [ ] Sistema de fallback robusto
- [ ] Validação adequada de dados

## 📚 Referências
- [Win32_ComputerSystemProduct](https://docs.microsoft.com/en-us/windows/win32/cimwin32prov/win32-computersystemproduct) - WMI class
- [Machine GUID](https://docs.microsoft.com/en-us/windows/win32/api/sysinfoapi/nf-sysinfoapi-getcomputernamea) - Registry location
- [Hardware ID Generation](https://docs.microsoft.com/en-us/windows-hardware/drivers/install/hardware-ids) - Hardware identification

## ⏭️ Próxima Task
[08-windows-services.md](08-windows-services.md) - Implementar coleta de serviços Windows 