# Task 06: Implementar Registry Scanning para Windows

## 📋 Objetivo
Implementar scanning do Registry Windows para descoberta de aplicações instaladas, configurações do sistema e informações complementares ao WMI.

## 🎯 Entregáveis
- [ ] Registry scanner implementado
- [ ] Descoberta de aplicações via Registry
- [ ] Coleta de informações do sistema
- [ ] Validação e sanitização de dados

## 📊 Contexto
O Registry Windows contém informações valiosas sobre aplicações instaladas que nem sempre estão disponíveis via WMI. Precisamos implementar um scanner seguro e eficiente.

## 🔧 Implementação

### 1. Criar `internal/collector/registry_windows.go`
```go
//go:build windows
// +build windows

package collector

import (
    "context"
    "fmt"
    "path/filepath"
    "strings"
    "time"
    
    "golang.org/x/sys/windows/registry"
    "machine-monitor/internal/logging"
)

// RegistryScanner implementa scanning do Registry Windows
type RegistryScanner struct {
    logger logging.Logger
}

// NewRegistryScanner cria um novo scanner do Registry
func NewRegistryScanner(logger logging.Logger) *RegistryScanner {
    return &RegistryScanner{
        logger: logger,
    }
}

// RegistryApp representa uma aplicação encontrada no Registry
type RegistryApp struct {
    Name         string
    Version      string
    Publisher    string
    InstallDate  string
    InstallPath  string
    UninstallString string
    Size         int64
    RegistryKey  string
}

// Common Registry paths for installed applications
const (
    UninstallKeyPath    = `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`
    UninstallKeyPathWow = `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`
    
    // System information paths
    WindowsVersionPath = `SOFTWARE\Microsoft\Windows NT\CurrentVersion`
    SystemInfoPath     = `SYSTEM\CurrentControlSet\Control\ComputerName\ComputerName`
)

// ScanInstalledApplications scans Registry for installed applications
func (r *RegistryScanner) ScanInstalledApplications(ctx context.Context) ([]Application, error) {
    r.logger.Info("Starting Registry scan for installed applications")
    
    var allApps []Application
    
    // Scan 64-bit applications
    if apps64, err := r.scanUninstallKey(ctx, registry.LOCAL_MACHINE, UninstallKeyPath); err == nil {
        allApps = append(allApps, apps64...)
    } else {
        r.logger.Warn("Failed to scan 64-bit applications", map[string]interface{}{
            "error": err.Error(),
        })
    }
    
    // Scan 32-bit applications (WOW64)
    if apps32, err := r.scanUninstallKey(ctx, registry.LOCAL_MACHINE, UninstallKeyPathWow); err == nil {
        allApps = append(allApps, apps32...)
    } else {
        r.logger.Warn("Failed to scan 32-bit applications", map[string]interface{}{
            "error": err.Error(),
        })
    }
    
    // Scan current user applications
    if userApps, err := r.scanUninstallKey(ctx, registry.CURRENT_USER, UninstallKeyPath); err == nil {
        allApps = append(allApps, userApps...)
    } else {
        r.logger.Debug("No user-specific applications found", map[string]interface{}{
            "error": err.Error(),
        })
    }
    
    // Remove duplicates and validate
    uniqueApps := r.removeDuplicateApps(allApps)
    validApps := r.validateApplications(uniqueApps)
    
    r.logger.Info("Registry scan completed", map[string]interface{}{
        "total_found":    len(allApps),
        "unique_apps":    len(uniqueApps),
        "valid_apps":     len(validApps),
    })
    
    return validApps, nil
}

// scanUninstallKey scans a specific uninstall registry key
func (r *RegistryScanner) scanUninstallKey(ctx context.Context, root registry.Key, path string) ([]Application, error) {
    var apps []Application
    
    key, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS)
    if err != nil {
        return nil, fmt.Errorf("failed to open registry key %s: %w", path, err)
    }
    defer key.Close()
    
    subkeys, err := key.ReadSubKeyNames(-1)
    if err != nil {
        return nil, fmt.Errorf("failed to read subkeys from %s: %w", path, err)
    }
    
    for _, subkey := range subkeys {
        select {
        case <-ctx.Done():
            return apps, ctx.Err()
        default:
        }
        
        if app := r.scanApplicationKey(root, filepath.Join(path, subkey), subkey); app != nil {
            apps = append(apps, *app)
        }
    }
    
    return apps, nil
}

// scanApplicationKey scans a specific application registry key
func (r *RegistryScanner) scanApplicationKey(root registry.Key, keyPath, keyName string) *Application {
    key, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE)
    if err != nil {
        return nil
    }
    defer key.Close()
    
    // Get application name
    displayName, _, err := key.GetStringValue("DisplayName")
    if err != nil || displayName == "" {
        // Skip entries without display name
        return nil
    }
    
    // Skip system components and updates
    if r.shouldSkipApplication(key, displayName, keyName) {
        return nil
    }
    
    app := &Application{
        Name: strings.TrimSpace(displayName),
        Type: "system",
    }
    
    // Get version
    if version, _, err := key.GetStringValue("DisplayVersion"); err == nil {
        app.Version = strings.TrimSpace(version)
    }
    
    // Get publisher/vendor
    if publisher, _, err := key.GetStringValue("Publisher"); err == nil {
        app.Vendor = strings.TrimSpace(publisher)
    }
    
    // Get install date
    if installDate, _, err := key.GetStringValue("InstallDate"); err == nil {
        app.InstallDate = r.formatInstallDate(installDate)
    }
    
    // Get install location
    if installLocation, _, err := key.GetStringValue("InstallLocation"); err == nil {
        app.Path = strings.TrimSpace(installLocation)
    }
    
    // Get size
    if size, _, err := key.GetIntegerValue("EstimatedSize"); err == nil {
        app.Size = int64(size) * 1024 // Convert from KB to bytes
    }
    
    return app
}

// shouldSkipApplication determines if an application should be skipped
func (r *RegistryScanner) shouldSkipApplication(key registry.Key, displayName, keyName string) bool {
    // Skip if SystemComponent is set
    if systemComponent, _, err := key.GetIntegerValue("SystemComponent"); err == nil && systemComponent == 1 {
        return true
    }
    
    // Skip if ParentKeyName is set (usually updates)
    if parentKey, _, err := key.GetStringValue("ParentKeyName"); err == nil && parentKey != "" {
        return true
    }
    
    // Skip if WindowsInstaller is set
    if windowsInstaller, _, err := key.GetIntegerValue("WindowsInstaller"); err == nil && windowsInstaller == 1 {
        // But keep if it has a proper display name
        if displayName == "" {
            return true
        }
    }
    
    // Skip common system update patterns
    skipPatterns := []string{
        "Security Update",
        "Hotfix",
        "Update for",
        "KB",
        "Microsoft Visual C++ 20",
        ".NET Framework",
    }
    
    displayNameUpper := strings.ToUpper(displayName)
    for _, pattern := range skipPatterns {
        if strings.Contains(displayNameUpper, strings.ToUpper(pattern)) {
            return true
        }
    }
    
    // Skip GUID-only names
    if strings.HasPrefix(keyName, "{") && strings.HasSuffix(keyName, "}") && len(keyName) == 38 {
        if displayName == keyName {
            return true
        }
    }
    
    return false
}

// formatInstallDate formats install date from Registry format
func (r *RegistryScanner) formatInstallDate(dateStr string) string {
    if len(dateStr) == 8 {
        // Format: YYYYMMDD
        year := dateStr[:4]
        month := dateStr[4:6]
        day := dateStr[6:8]
        return fmt.Sprintf("%s-%s-%s", year, month, day)
    }
    return dateStr
}

// removeDuplicateApps removes duplicate applications
func (r *RegistryScanner) removeDuplicateApps(apps []Application) []Application {
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

// validateApplications validates and filters applications
func (r *RegistryScanner) validateApplications(apps []Application) []Application {
    var valid []Application
    
    for _, app := range apps {
        if r.isValidApplication(app) {
            valid = append(valid, app)
        }
    }
    
    return valid
}

// isValidApplication checks if an application is valid
func (r *RegistryScanner) isValidApplication(app Application) bool {
    // Must have a name
    if app.Name == "" {
        return false
    }
    
    // Name must be reasonable length
    if len(app.Name) < 2 || len(app.Name) > 200 {
        return false
    }
    
    // Skip common invalid names
    invalidNames := []string{
        "Adobe Flash Player",
        "Microsoft Visual C++",
        "Microsoft .NET Framework",
        "Windows Software Development Kit",
    }
    
    for _, invalid := range invalidNames {
        if strings.Contains(app.Name, invalid) {
            return false
        }
    }
    
    return true
}

// GetSystemInformation gets system information from Registry
func (r *RegistryScanner) GetSystemInformation(ctx context.Context) (map[string]interface{}, error) {
    info := make(map[string]interface{})
    
    // Windows version information
    if versionInfo, err := r.getWindowsVersionInfo(); err == nil {
        info["windows_version"] = versionInfo
    }
    
    // Computer name
    if computerName, err := r.getComputerName(); err == nil {
        info["computer_name"] = computerName
    }
    
    return info, nil
}

// getWindowsVersionInfo gets Windows version from Registry
func (r *RegistryScanner) getWindowsVersionInfo() (map[string]interface{}, error) {
    key, err := registry.OpenKey(registry.LOCAL_MACHINE, WindowsVersionPath, registry.QUERY_VALUE)
    if err != nil {
        return nil, err
    }
    defer key.Close()
    
    info := make(map[string]interface{})
    
    // Product name
    if productName, _, err := key.GetStringValue("ProductName"); err == nil {
        info["product_name"] = productName
    }
    
    // Current version
    if currentVersion, _, err := key.GetStringValue("CurrentVersion"); err == nil {
        info["current_version"] = currentVersion
    }
    
    // Build number
    if buildNumber, _, err := key.GetStringValue("CurrentBuildNumber"); err == nil {
        info["build_number"] = buildNumber
    }
    
    // Release ID
    if releaseId, _, err := key.GetStringValue("ReleaseId"); err == nil {
        info["release_id"] = releaseId
    }
    
    // Install date
    if installDate, _, err := key.GetIntegerValue("InstallDate"); err == nil {
        info["install_date"] = time.Unix(int64(installDate), 0).Format("2006-01-02")
    }
    
    return info, nil
}

// getComputerName gets computer name from Registry
func (r *RegistryScanner) getComputerName() (string, error) {
    key, err := registry.OpenKey(registry.LOCAL_MACHINE, SystemInfoPath, registry.QUERY_VALUE)
    if err != nil {
        return "", err
    }
    defer key.Close()
    
    computerName, _, err := key.GetStringValue("ComputerName")
    return computerName, err
}

// GetInstalledFeatures gets Windows features from Registry
func (r *RegistryScanner) GetInstalledFeatures(ctx context.Context) ([]map[string]interface{}, error) {
    // This would scan for Windows features/roles
    // Implementation depends on specific requirements
    return []map[string]interface{}{}, nil
}
```

### 2. Criar `internal/collector/registry_helpers_windows.go`
```go
//go:build windows
// +build windows

package collector

import (
    "context"
    "fmt"
    "strings"
    
    "golang.org/x/sys/windows/registry"
    "machine-monitor/internal/logging"
)

// RegistryHelper fornece métodos de conveniência para Registry
type RegistryHelper struct {
    scanner *RegistryScanner
    logger  logging.Logger
}

// NewRegistryHelper cria um novo helper do Registry
func NewRegistryHelper(logger logging.Logger) *RegistryHelper {
    return &RegistryHelper{
        scanner: NewRegistryScanner(logger),
        logger:  logger,
    }
}

// GetValue gets a string value from Registry
func (h *RegistryHelper) GetValue(root registry.Key, path, valueName string) (string, error) {
    key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
    if err != nil {
        return "", err
    }
    defer key.Close()
    
    value, _, err := key.GetStringValue(valueName)
    return value, err
}

// GetIntValue gets an integer value from Registry
func (h *RegistryHelper) GetIntValue(root registry.Key, path, valueName string) (uint64, error) {
    key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
    if err != nil {
        return 0, err
    }
    defer key.Close()
    
    value, _, err := key.GetIntegerValue(valueName)
    return value, err
}

// KeyExists checks if a registry key exists
func (h *RegistryHelper) KeyExists(root registry.Key, path string) bool {
    key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
    if err != nil {
        return false
    }
    key.Close()
    return true
}

// GetSubKeys gets all subkeys of a registry key
func (h *RegistryHelper) GetSubKeys(root registry.Key, path string) ([]string, error) {
    key, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS)
    if err != nil {
        return nil, err
    }
    defer key.Close()
    
    return key.ReadSubKeyNames(-1)
}

// GetValueNames gets all value names in a registry key
func (h *RegistryHelper) GetValueNames(root registry.Key, path string) ([]string, error) {
    key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
    if err != nil {
        return nil, err
    }
    defer key.Close()
    
    return key.ReadValueNames(-1)
}

// SafeGetValue safely gets a value with error handling
func (h *RegistryHelper) SafeGetValue(root registry.Key, path, valueName string) string {
    value, err := h.GetValue(root, path, valueName)
    if err != nil {
        h.logger.Debug("Failed to get registry value", map[string]interface{}{
            "path":       path,
            "value_name": valueName,
            "error":      err.Error(),
        })
        return ""
    }
    return strings.TrimSpace(value)
}

// SafeGetIntValue safely gets an integer value with error handling
func (h *RegistryHelper) SafeGetIntValue(root registry.Key, path, valueName string) uint64 {
    value, err := h.GetIntValue(root, path, valueName)
    if err != nil {
        h.logger.Debug("Failed to get registry int value", map[string]interface{}{
            "path":       path,
            "value_name": valueName,
            "error":      err.Error(),
        })
        return 0
    }
    return value
}

// GetMachineGUID gets the machine GUID from Registry
func (h *RegistryHelper) GetMachineGUID(ctx context.Context) (string, error) {
    // Try multiple sources for machine identification
    paths := []struct {
        root      registry.Key
        path      string
        valueName string
        prefix    string
    }{
        {registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, "MachineGuid", "crypto"},
        {registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "InstallDate", "install"},
        {registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\ComputerName\ComputerName`, "ComputerName", "name"},
    }
    
    for _, p := range paths {
        if value := h.SafeGetValue(p.root, p.path, p.valueName); value != "" {
            return fmt.Sprintf("%s-%s", p.prefix, value), nil
        }
    }
    
    return "", fmt.Errorf("failed to get machine GUID from Registry")
}

// GetInstalledPrograms gets installed programs with enhanced filtering
func (h *RegistryHelper) GetInstalledPrograms(ctx context.Context) ([]Application, error) {
    return h.scanner.ScanInstalledApplications(ctx)
}

// GetSystemConfiguration gets system configuration from Registry
func (h *RegistryHelper) GetSystemConfiguration(ctx context.Context) (map[string]interface{}, error) {
    config := make(map[string]interface{})
    
    // Boot configuration
    if bootConfig, err := h.getBootConfiguration(); err == nil {
        config["boot"] = bootConfig
    }
    
    // Network configuration
    if networkConfig, err := h.getNetworkConfiguration(); err == nil {
        config["network"] = networkConfig
    }
    
    // Hardware configuration
    if hardwareConfig, err := h.getHardwareConfiguration(); err == nil {
        config["hardware"] = hardwareConfig
    }
    
    return config, nil
}

// getBootConfiguration gets boot configuration from Registry
func (h *RegistryHelper) getBootConfiguration() (map[string]interface{}, error) {
    config := make(map[string]interface{})
    
    basePath := `SYSTEM\CurrentControlSet\Control`
    
    // Boot info
    if bootInfo := h.SafeGetValue(registry.LOCAL_MACHINE, basePath, "SystemBootDevice"); bootInfo != "" {
        config["boot_device"] = bootInfo
    }
    
    // Computer name
    if computerName := h.SafeGetValue(registry.LOCAL_MACHINE, basePath+`\ComputerName\ComputerName`, "ComputerName"); computerName != "" {
        config["computer_name"] = computerName
    }
    
    return config, nil
}

// getNetworkConfiguration gets network configuration from Registry
func (h *RegistryHelper) getNetworkConfiguration() (map[string]interface{}, error) {
    config := make(map[string]interface{})
    
    // Network adapters
    adaptersPath := `SYSTEM\CurrentControlSet\Control\Class\{4D36E972-E325-11CE-BFC1-08002BE10318}`
    
    if subkeys, err := h.GetSubKeys(registry.LOCAL_MACHINE, adaptersPath); err == nil {
        var adapters []map[string]interface{}
        for _, subkey := range subkeys {
            adapterPath := fmt.Sprintf("%s\\%s", adaptersPath, subkey)
            if desc := h.SafeGetValue(registry.LOCAL_MACHINE, adapterPath, "DriverDesc"); desc != "" {
                adapter := map[string]interface{}{
                    "description": desc,
                    "driver":      h.SafeGetValue(registry.LOCAL_MACHINE, adapterPath, "DriverVersion"),
                }
                adapters = append(adapters, adapter)
            }
        }
        config["adapters"] = adapters
    }
    
    return config, nil
}

// getHardwareConfiguration gets hardware configuration from Registry
func (h *RegistryHelper) getHardwareConfiguration() (map[string]interface{}, error) {
    config := make(map[string]interface{})
    
    // Processor information
    processorPath := `HARDWARE\DESCRIPTION\System\CentralProcessor\0`
    if processorName := h.SafeGetValue(registry.LOCAL_MACHINE, processorPath, "ProcessorNameString"); processorName != "" {
        config["processor"] = map[string]interface{}{
            "name":       processorName,
            "identifier": h.SafeGetValue(registry.LOCAL_MACHINE, processorPath, "Identifier"),
        }
    }
    
    return config, nil
}

// ValidateRegistryPath validates a registry path for safety
func (h *RegistryHelper) ValidateRegistryPath(path string) error {
    // Prevent access to sensitive paths
    forbiddenPaths := []string{
        "SAM",
        "SECURITY",
        "SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Authentication",
        "SOFTWARE\\Microsoft\\Cryptography\\RNG",
    }
    
    pathUpper := strings.ToUpper(path)
    for _, forbidden := range forbiddenPaths {
        if strings.Contains(pathUpper, strings.ToUpper(forbidden)) {
            return fmt.Errorf("access to registry path %s is forbidden", path)
        }
    }
    
    return nil
}
```

## 📋 Checklist de Implementação

### Arquivos a Criar
- [ ] `internal/collector/registry_windows.go` - Scanner principal
- [ ] `internal/collector/registry_helpers_windows.go` - Helpers de conveniência

### Funcionalidades Registry
- [ ] Scanning de aplicações instaladas
- [ ] Filtragem de aplicações do sistema
- [ ] Informações de versão do Windows
- [ ] Configuração do sistema
- [ ] Validação de segurança

### Chaves Registry Escaneadas
- [ ] Uninstall keys (64-bit e 32-bit)
- [ ] Windows version information
- [ ] Computer name e configuração
- [ ] Network adapters
- [ ] Hardware information

### Validações
- [ ] Scanning funciona corretamente
- [ ] Filtragem remove aplicações irrelevantes
- [ ] Dados são sanitizados adequadamente
- [ ] Acesso a paths sensíveis é bloqueado

## 🎯 Critérios de Sucesso
- [ ] Registry scanning funcional e seguro
- [ ] Descoberta precisa de aplicações
- [ ] Filtragem eficaz de ruído
- [ ] Performance adequada (<10s para scan completo)

## 📚 Referências
- [Windows Registry](https://docs.microsoft.com/en-us/windows/win32/sysinfo/registry) - Documentação oficial
- [golang.org/x/sys/windows/registry](https://pkg.go.dev/golang.org/x/sys/windows/registry) - Biblioteca Registry
- [Registry Keys for Applications](https://docs.microsoft.com/en-us/windows/win32/msi/uninstall-registry-key) - Chaves de desinstalação

## ⏭️ Próxima Task
[07-windows-machine-id.md](07-windows-machine-id.md) - Implementar Machine ID para Windows 