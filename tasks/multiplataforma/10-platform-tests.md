# Task 10: Testes Específicos de Plataforma

## 📋 Objetivo
Implementar uma suíte abrangente de testes específicos para cada plataforma (Windows, macOS, Linux), garantindo que todas as funcionalidades trabalhem corretamente em seus respectivos ambientes.

## 🎯 Entregáveis
- [ ] Framework de testes multiplataforma
- [ ] Testes unitários específicos por plataforma
- [ ] Testes de compatibilidade de dados
- [ ] Mocks para APIs específicas de plataforma
- [ ] Relatórios de cobertura por plataforma
- [ ] Testes de regressão automatizados

## 📊 Contexto
Com a implementação das interfaces multiplataforma e código específico por plataforma, precisamos garantir que cada implementação funcione corretamente em seu ambiente nativo, mantendo consistência na API e nos dados retornados.

## 🔧 Implementação

### 1. Criar Framework de Testes Multiplataforma

#### `internal/collector/testing/platform_test_framework.go`
```go
package testing

import (
    "context"
    "runtime"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "machine-monitor/internal/collector"
    "machine-monitor/internal/logging"
)

// PlatformTestSuite define testes que devem rodar em todas as plataformas
type PlatformTestSuite struct {
    collector collector.PlatformCollector
    logger    logging.Logger
    t         *testing.T
}

func NewPlatformTestSuite(t *testing.T) *PlatformTestSuite {
    logger := logging.NewLogger(logging.Config{Level: "debug"})
    
    // Criar collector específico da plataforma atual
    var platformCollector collector.PlatformCollector
    
    switch runtime.GOOS {
    case "windows":
        platformCollector = collector.NewWindowsCollector(logger, &collector.Config{})
    case "darwin":
        platformCollector = collector.NewDarwinCollector(logger, &collector.Config{})
    case "linux":
        platformCollector = collector.NewLinuxCollector(logger, &collector.Config{})
    default:
        t.Fatalf("Plataforma não suportada: %s", runtime.GOOS)
    }
    
    return &PlatformTestSuite{
        collector: platformCollector,
        logger:    logger,
        t:         t,
    }
}

// RunAllPlatformTests executa todos os testes padrão
func (pts *PlatformTestSuite) RunAllPlatformTests() {
    pts.t.Run("MachineID", pts.TestMachineID)
    pts.t.Run("SystemInfo", pts.TestSystemInfo)
    pts.t.Run("InstalledApps", pts.TestInstalledApps)
    pts.t.Run("SystemServices", pts.TestSystemServices)
    pts.t.Run("Performance", pts.TestPerformance)
    pts.t.Run("DataConsistency", pts.TestDataConsistency)
}

func (pts *PlatformTestSuite) TestMachineID() {
    ctx := context.Background()
    
    // Teste básico de geração de Machine ID
    machineID, err := pts.collector.GetMachineID(ctx)
    require.NoError(pts.t, err)
    assert.NotEmpty(pts.t, machineID)
    assert.Greater(pts.t, len(machineID), 8, "Machine ID deve ter pelo menos 8 caracteres")
    
    // Teste de consistência - deve retornar o mesmo ID
    machineID2, err := pts.collector.GetMachineID(ctx)
    require.NoError(pts.t, err)
    assert.Equal(pts.t, machineID, machineID2, "Machine ID deve ser consistente")
    
    // Teste de formato específico da plataforma
    switch runtime.GOOS {
    case "windows":
        assert.True(pts.t, pts.isValidWindowsMachineID(machineID))
    case "darwin":
        assert.True(pts.t, pts.isValidDarwinMachineID(machineID))
    case "linux":
        assert.True(pts.t, pts.isValidLinuxMachineID(machineID))
    }
}

func (pts *PlatformTestSuite) TestSystemInfo() {
    ctx := context.Background()
    
    info, err := pts.collector.CollectPlatformSpecific(ctx)
    require.NoError(pts.t, err)
    require.NotNil(pts.t, info)
    
    // Verificações básicas
    assert.NotEmpty(pts.t, info.OS)
    assert.NotEmpty(pts.t, info.OSVersion)
    assert.NotEmpty(pts.t, info.Architecture)
    assert.NotEmpty(pts.t, info.Hostname)
    
    // Verificações específicas da plataforma
    switch runtime.GOOS {
    case "windows":
        pts.validateWindowsSystemInfo(info)
    case "darwin":
        pts.validateDarwinSystemInfo(info)
    case "linux":
        pts.validateLinuxSystemInfo(info)
    }
}

func (pts *PlatformTestSuite) TestInstalledApps() {
    ctx := context.Background()
    
    apps, err := pts.collector.CollectInstalledApps(ctx)
    require.NoError(pts.t, err)
    assert.NotEmpty(pts.t, apps, "Deve encontrar pelo menos algumas aplicações")
    
    // Verificar estrutura das aplicações
    for _, app := range apps {
        assert.NotEmpty(pts.t, app.Name, "Nome da aplicação não pode estar vazio")
        assert.NotEmpty(pts.t, app.Type, "Tipo da aplicação deve estar definido")
        
        // Verificar se o tipo é válido para a plataforma
        assert.Contains(pts.t, pts.getValidAppTypes(), app.Type)
    }
    
    // Verificar se encontrou aplicações conhecidas da plataforma
    pts.verifyKnownApps(apps)
}

func (pts *PlatformTestSuite) TestSystemServices() {
    ctx := context.Background()
    
    services, err := pts.collector.CollectSystemServices(ctx)
    require.NoError(pts.t, err)
    assert.NotEmpty(pts.t, services, "Deve encontrar pelo menos alguns serviços")
    
    // Verificar estrutura dos serviços
    for _, service := range services {
        assert.NotEmpty(pts.t, service.Name, "Nome do serviço não pode estar vazio")
        assert.NotEmpty(pts.t, service.Status, "Status do serviço deve estar definido")
        
        // Verificar se o status é válido
        validStatuses := []string{"running", "stopped", "paused", "starting", "stopping"}
        assert.Contains(pts.t, validStatuses, service.Status)
    }
    
    // Verificar se encontrou serviços conhecidos da plataforma
    pts.verifyKnownServices(services)
}

func (pts *PlatformTestSuite) TestPerformance() {
    ctx := context.Background()
    
    // Teste de performance da coleta completa
    start := time.Now()
    _, err := pts.collector.CollectPlatformSpecific(ctx)
    duration := time.Since(start)
    
    require.NoError(pts.t, err)
    assert.Less(pts.t, duration, 30*time.Second, "Coleta deve completar em menos de 30 segundos")
    
    // Teste de performance das aplicações
    start = time.Now()
    apps, err := pts.collector.CollectInstalledApps(ctx)
    duration = time.Since(start)
    
    require.NoError(pts.t, err)
    assert.Less(pts.t, duration, 60*time.Second, "Coleta de aplicações deve completar em menos de 60 segundos")
    assert.Greater(pts.t, len(apps), 0, "Deve encontrar pelo menos uma aplicação")
}

func (pts *PlatformTestSuite) TestDataConsistency() {
    ctx := context.Background()
    
    // Executar coleta múltiplas vezes para verificar consistência
    var results []collector.PlatformInfo
    
    for i := 0; i < 3; i++ {
        info, err := pts.collector.CollectPlatformSpecific(ctx)
        require.NoError(pts.t, err)
        results = append(results, *info)
    }
    
    // Verificar se dados estáticos são consistentes
    for i := 1; i < len(results); i++ {
        assert.Equal(pts.t, results[0].OS, results[i].OS)
        assert.Equal(pts.t, results[0].OSVersion, results[i].OSVersion)
        assert.Equal(pts.t, results[0].Architecture, results[i].Architecture)
        assert.Equal(pts.t, results[0].Hostname, results[i].Hostname)
    }
}

// Métodos auxiliares específicos por plataforma
func (pts *PlatformTestSuite) isValidWindowsMachineID(id string) bool {
    // Windows Machine ID deve ter prefixo específico
    prefixes := []string{"mb-", "bios-", "win-", "fallback-"}
    for _, prefix := range prefixes {
        if strings.HasPrefix(id, prefix) {
            return len(id) > len(prefix)+8
        }
    }
    return false
}

func (pts *PlatformTestSuite) isValidDarwinMachineID(id string) bool {
    // macOS Machine ID deve ter prefixo específico
    prefixes := []string{"hw-", "serial-", "fallback-"}
    for _, prefix := range prefixes {
        if strings.HasPrefix(id, prefix) {
            return len(id) > len(prefix)+8
        }
    }
    return false
}

func (pts *PlatformTestSuite) getValidAppTypes() []string {
    switch runtime.GOOS {
    case "windows":
        return []string{"Registry", "UWP", "Portable", "MSI", "EXE"}
    case "darwin":
        return []string{"App", "PKG", "DMG", "Homebrew"}
    case "linux":
        return []string{"DEB", "RPM", "Snap", "Flatpak", "AppImage"}
    default:
        return []string{"Unknown"}
    }
}

func (pts *PlatformTestSuite) verifyKnownApps(apps []collector.Application) {
    knownApps := pts.getKnownAppsForPlatform()
    
    foundCount := 0
    for _, knownApp := range knownApps {
        for _, app := range apps {
            if strings.Contains(strings.ToLower(app.Name), strings.ToLower(knownApp)) {
                foundCount++
                break
            }
        }
    }
    
    // Deve encontrar pelo menos 30% das aplicações conhecidas
    minExpected := len(knownApps) * 30 / 100
    assert.GreaterOrEqual(pts.t, foundCount, minExpected, 
        "Deve encontrar pelo menos %d aplicações conhecidas, encontrou %d", minExpected, foundCount)
}

func (pts *PlatformTestSuite) getKnownAppsForPlatform() []string {
    switch runtime.GOOS {
    case "windows":
        return []string{"Calculator", "Notepad", "Paint", "Microsoft Edge", "Windows Media Player"}
    case "darwin":
        return []string{"Safari", "Calculator", "TextEdit", "Preview", "Finder"}
    case "linux":
        return []string{"Firefox", "LibreOffice", "GIMP", "Terminal", "Files"}
    default:
        return []string{}
    }
}

func (pts *PlatformTestSuite) verifyKnownServices(services []collector.Service) {
    knownServices := pts.getKnownServicesForPlatform()
    
    foundCount := 0
    for _, knownService := range knownServices {
        for _, service := range services {
            if strings.Contains(strings.ToLower(service.Name), strings.ToLower(knownService)) {
                foundCount++
                break
            }
        }
    }
    
    // Deve encontrar pelo menos 50% dos serviços conhecidos
    minExpected := len(knownServices) * 50 / 100
    assert.GreaterOrEqual(pts.t, foundCount, minExpected,
        "Deve encontrar pelo menos %d serviços conhecidos, encontrou %d", minExpected, foundCount)
}

func (pts *PlatformTestSuite) getKnownServicesForPlatform() []string {
    switch runtime.GOOS {
    case "windows":
        return []string{"Winlogon", "Themes", "AudioSrv", "BITS", "Dhcp"}
    case "darwin":
        return []string{"com.apple.WindowServer", "com.apple.loginwindow", "com.apple.Finder"}
    case "linux":
        return []string{"systemd", "NetworkManager", "dbus", "ssh"}
    default:
        return []string{}
    }
}
```

### 2. Testes Específicos do Windows

#### `internal/collector/platform_windows_test.go`
```go
//go:build windows

package collector

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "machine-monitor/internal/logging"
)

func TestWindowsCollector(t *testing.T) {
    logger := logging.NewLogger(logging.Config{Level: "debug"})
    collector := NewWindowsCollector(logger, &Config{})
    
    t.Run("WMI_Connectivity", func(t *testing.T) {
        testWMIConnectivity(t, collector)
    })
    
    t.Run("Registry_Access", func(t *testing.T) {
        testRegistryAccess(t, collector)
    })
    
    t.Run("UWP_Apps", func(t *testing.T) {
        testUWPApps(t, collector)
    })
    
    t.Run("Windows_Services", func(t *testing.T) {
        testWindowsServices(t, collector)
    })
    
    t.Run("Windows_Machine_ID", func(t *testing.T) {
        testWindowsMachineID(t, collector)
    })
}

func testWMIConnectivity(t *testing.T, collector *WindowsCollector) {
    ctx := context.Background()
    
    // Teste de conectividade WMI básica
    query := "SELECT Name FROM Win32_OperatingSystem"
    results, err := collector.queryWMI(query)
    
    require.NoError(t, err)
    assert.NotEmpty(t, results)
    assert.Contains(t, results[0], "Name")
}

func testRegistryAccess(t *testing.T, collector *WindowsCollector) {
    // Teste de acesso ao Registry
    apps, err := collector.getInstalledProgramsFromRegistry()
    
    require.NoError(t, err)
    assert.NotEmpty(t, apps, "Deve encontrar aplicações no Registry")
    
    // Verificar se encontrou aplicações básicas do Windows
    foundWindowsApps := 0
    for _, app := range apps {
        if strings.Contains(strings.ToLower(app.Name), "microsoft") {
            foundWindowsApps++
        }
    }
    
    assert.Greater(t, foundWindowsApps, 0, "Deve encontrar pelo menos uma aplicação Microsoft")
}

func testUWPApps(t *testing.T, collector *WindowsCollector) {
    ctx := context.Background()
    
    apps, err := collector.getUWPApps()
    
    // UWP apps podem não estar disponíveis em todos os ambientes
    if err != nil {
        t.Skipf("UWP apps não disponíveis: %v", err)
    }
    
    assert.NotEmpty(t, apps, "Deve encontrar pelo menos algumas UWP apps")
    
    // Verificar estrutura das UWP apps
    for _, app := range apps {
        assert.Equal(t, "UWP", app.Type)
        assert.NotEmpty(t, app.Name)
    }
}

func testWindowsServices(t *testing.T, collector *WindowsCollector) {
    ctx := context.Background()
    
    services, err := collector.CollectSystemServices(ctx)
    
    require.NoError(t, err)
    assert.NotEmpty(t, services, "Deve encontrar serviços do Windows")
    
    // Verificar se encontrou serviços essenciais
    essentialServices := []string{"Winlogon", "Themes", "AudioSrv"}
    foundServices := 0
    
    for _, service := range services {
        for _, essential := range essentialServices {
            if strings.Contains(strings.ToLower(service.Name), strings.ToLower(essential)) {
                foundServices++
                break
            }
        }
    }
    
    assert.Greater(t, foundServices, 0, "Deve encontrar pelo menos um serviço essencial")
}

func testWindowsMachineID(t *testing.T, collector *WindowsCollector) {
    ctx := context.Background()
    
    // Teste de geração de Machine ID
    machineID, err := collector.GetMachineID(ctx)
    
    require.NoError(t, err)
    assert.NotEmpty(t, machineID)
    
    // Verificar formato específico do Windows
    validPrefixes := []string{"mb-", "bios-", "win-", "fallback-"}
    hasValidPrefix := false
    
    for _, prefix := range validPrefixes {
        if strings.HasPrefix(machineID, prefix) {
            hasValidPrefix = true
            break
        }
    }
    
    assert.True(t, hasValidPrefix, "Machine ID deve ter prefixo válido: %s", machineID)
}

// Testes de performance específicos do Windows
func TestWindowsPerformance(t *testing.T) {
    logger := logging.NewLogger(logging.Config{Level: "warn"})
    collector := NewWindowsCollector(logger, &Config{})
    
    t.Run("WMI_Query_Performance", func(t *testing.T) {
        ctx := context.Background()
        
        start := time.Now()
        _, err := collector.queryWMI("SELECT * FROM Win32_OperatingSystem")
        duration := time.Since(start)
        
        require.NoError(t, err)
        assert.Less(t, duration, 5*time.Second, "Query WMI deve completar em menos de 5 segundos")
    })
    
    t.Run("Registry_Scan_Performance", func(t *testing.T) {
        start := time.Now()
        apps, err := collector.getInstalledProgramsFromRegistry()
        duration := time.Since(start)
        
        require.NoError(t, err)
        assert.Less(t, duration, 10*time.Second, "Registry scan deve completar em menos de 10 segundos")
        assert.Greater(t, len(apps), 0, "Deve encontrar pelo menos uma aplicação")
    })
}

// Testes de erro e recuperação
func TestWindowsErrorHandling(t *testing.T) {
    logger := logging.NewLogger(logging.Config{Level: "debug"})
    collector := NewWindowsCollector(logger, &Config{})
    
    t.Run("Invalid_WMI_Query", func(t *testing.T) {
        _, err := collector.queryWMI("SELECT * FROM InvalidTable")
        assert.Error(t, err, "Query WMI inválida deve retornar erro")
    })
    
    t.Run("Registry_Access_Error", func(t *testing.T) {
        // Simular erro de acesso ao Registry
        // (implementar mock se necessário)
    })
}
```

### 3. Testes Específicos do macOS

#### `internal/collector/platform_darwin_test.go`
```go
//go:build darwin

package collector

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "machine-monitor/internal/logging"
)

func TestDarwinCollector(t *testing.T) {
    logger := logging.NewLogger(logging.Config{Level: "debug"})
    collector := NewDarwinCollector(logger, &Config{})
    
    t.Run("System_Profiler", func(t *testing.T) {
        testSystemProfiler(t, collector)
    })
    
    t.Run("Applications_Discovery", func(t *testing.T) {
        testApplicationsDiscovery(t, collector)
    })
    
    t.Run("Launchctl_Services", func(t *testing.T) {
        testLaunchctlServices(t, collector)
    })
    
    t.Run("Darwin_Machine_ID", func(t *testing.T) {
        testDarwinMachineID(t, collector)
    })
}

func testSystemProfiler(t *testing.T, collector *DarwinCollector) {
    ctx := context.Background()
    
    info, err := collector.CollectPlatformSpecific(ctx)
    
    require.NoError(t, err)
    assert.Equal(t, "darwin", info.OS)
    assert.NotEmpty(t, info.OSVersion)
    assert.NotEmpty(t, info.Architecture)
}

func testApplicationsDiscovery(t *testing.T, collector *DarwinCollector) {
    ctx := context.Background()
    
    apps, err := collector.CollectInstalledApps(ctx)
    
    require.NoError(t, err)
    assert.NotEmpty(t, apps, "Deve encontrar aplicações no macOS")
    
    // Verificar se encontrou aplicações básicas do macOS
    foundSystemApps := 0
    systemApps := []string{"Safari", "Calculator", "TextEdit", "Preview"}
    
    for _, app := range apps {
        for _, sysApp := range systemApps {
            if strings.Contains(app.Name, sysApp) {
                foundSystemApps++
                break
            }
        }
    }
    
    assert.Greater(t, foundSystemApps, 0, "Deve encontrar pelo menos uma aplicação do sistema")
}

func testLaunchctlServices(t *testing.T, collector *DarwinCollector) {
    ctx := context.Background()
    
    services, err := collector.CollectSystemServices(ctx)
    
    require.NoError(t, err)
    assert.NotEmpty(t, services, "Deve encontrar serviços do macOS")
    
    // Verificar se encontrou serviços essenciais
    essentialServices := []string{"com.apple.WindowServer", "com.apple.loginwindow"}
    foundServices := 0
    
    for _, service := range services {
        for _, essential := range essentialServices {
            if strings.Contains(service.Name, essential) {
                foundServices++
                break
            }
        }
    }
    
    assert.Greater(t, foundServices, 0, "Deve encontrar pelo menos um serviço essencial")
}

func testDarwinMachineID(t *testing.T, collector *DarwinCollector) {
    ctx := context.Background()
    
    machineID, err := collector.GetMachineID(ctx)
    
    require.NoError(t, err)
    assert.NotEmpty(t, machineID)
    
    // Verificar formato específico do macOS
    validPrefixes := []string{"hw-", "serial-", "fallback-"}
    hasValidPrefix := false
    
    for _, prefix := range validPrefixes {
        if strings.HasPrefix(machineID, prefix) {
            hasValidPrefix = true
            break
        }
    }
    
    assert.True(t, hasValidPrefix, "Machine ID deve ter prefixo válido: %s", machineID)
}
```

### 4. Testes de Integração Multiplataforma

#### `internal/collector/integration_test.go`
```go
package collector

import (
    "context"
    "runtime"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "machine-monitor/internal/logging"
    testingFramework "machine-monitor/internal/collector/testing"
)

func TestPlatformIntegration(t *testing.T) {
    // Executar testes de integração usando o framework
    suite := testingFramework.NewPlatformTestSuite(t)
    suite.RunAllPlatformTests()
}

func TestCrossPlatformConsistency(t *testing.T) {
    logger := logging.NewLogger(logging.Config{Level: "debug"})
    
    // Criar collector para a plataforma atual
    collector := createCollectorForCurrentPlatform(t, logger)
    
    ctx := context.Background()
    
    // Testar consistência da API entre plataformas
    t.Run("API_Consistency", func(t *testing.T) {
        // Todos os métodos devem estar disponíveis
        machineID, err := collector.GetMachineID(ctx)
        require.NoError(t, err)
        assert.NotEmpty(t, machineID)
        
        info, err := collector.CollectPlatformSpecific(ctx)
        require.NoError(t, err)
        assert.NotNil(t, info)
        
        apps, err := collector.CollectInstalledApps(ctx)
        require.NoError(t, err)
        assert.NotNil(t, apps)
        
        services, err := collector.CollectSystemServices(ctx)
        require.NoError(t, err)
        assert.NotNil(t, services)
    })
    
    // Testar formato consistente dos dados
    t.Run("Data_Format_Consistency", func(t *testing.T) {
        apps, err := collector.CollectInstalledApps(ctx)
        require.NoError(t, err)
        
        for _, app := range apps {
            // Campos obrigatórios devem estar presentes
            assert.NotEmpty(t, app.Name, "Nome da aplicação é obrigatório")
            assert.NotEmpty(t, app.Type, "Tipo da aplicação é obrigatório")
            
            // Campos opcionais devem ter formato válido quando presentes
            if app.Version != "" {
                assert.Regexp(t, `^[\d\w\.\-_]+$`, app.Version, "Versão deve ter formato válido")
            }
        }
    })
}

func createCollectorForCurrentPlatform(t *testing.T, logger logging.Logger) PlatformCollector {
    switch runtime.GOOS {
    case "windows":
        return NewWindowsCollector(logger, &Config{})
    case "darwin":
        return NewDarwinCollector(logger, &Config{})
    case "linux":
        return NewLinuxCollector(logger, &Config{})
    default:
        t.Fatalf("Plataforma não suportada: %s", runtime.GOOS)
        return nil
    }
}
```

### 5. Configuração de CI/CD para Testes Multiplataforma

#### `.github/workflows/platform-tests.yml`
```yaml
name: Platform Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test-windows:
    runs-on: windows-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.21
    
    - name: Run Windows Tests
      run: |
        go test -v -tags=windows ./internal/collector/... -coverprofile=coverage-windows.out
        go tool cover -html=coverage-windows.out -o coverage-windows.html
    
    - name: Upload Windows Coverage
      uses: actions/upload-artifact@v3
      with:
        name: coverage-windows
        path: coverage-windows.html

  test-macos:
    runs-on: macos-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.21
    
    - name: Run macOS Tests
      run: |
        go test -v -tags=darwin ./internal/collector/... -coverprofile=coverage-macos.out
        go tool cover -html=coverage-macos.out -o coverage-macos.html
    
    - name: Upload macOS Coverage
      uses: actions/upload-artifact@v3
      with:
        name: coverage-macos
        path: coverage-macos.html

  test-linux:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.21
    
    - name: Run Linux Tests
      run: |
        go test -v -tags=linux ./internal/collector/... -coverprofile=coverage-linux.out
        go tool cover -html=coverage-linux.out -o coverage-linux.html
    
    - name: Upload Linux Coverage
      uses: actions/upload-artifact@v3
      with:
        name: coverage-linux
        path: coverage-linux.html
```

## ✅ Critérios de Sucesso

### Cobertura de Testes
- [ ] Cobertura > 85% para código específico de plataforma
- [ ] Cobertura > 95% para interfaces e código comum
- [ ] Todos os métodos públicos testados

### Qualidade dos Testes
- [ ] Testes unitários para cada função específica de plataforma
- [ ] Testes de integração para fluxos completos
- [ ] Testes de performance para operações críticas
- [ ] Testes de erro e recuperação

### Consistência Multiplataforma
- [ ] API consistente entre plataformas
- [ ] Formato de dados padronizado
- [ ] Comportamento similar em cenários equivalentes

## 🧪 Execução dos Testes

### Localmente
```bash
# Testes da plataforma atual
go test -v ./internal/collector/...

# Testes com cobertura
go test -v -coverprofile=coverage.out ./internal/collector/...
go tool cover -html=coverage.out

# Testes específicos de plataforma
go test -v -tags=windows ./internal/collector/...
go test -v -tags=darwin ./internal/collector/...
go test -v -tags=linux ./internal/collector/...
```

### CI/CD
```bash
# Executar em múltiplas plataformas via GitHub Actions
git push origin feature/platform-tests
```

## 📚 Referências

### Testing em Go
- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Framework](https://github.com/stretchr/testify)
- [Build Constraints](https://pkg.go.dev/cmd/go#hdr-Build_constraints)

### CI/CD
- [GitHub Actions](https://docs.github.com/en/actions)
- [Go Coverage](https://go.dev/blog/cover)

## 🔄 Próximos Passos
Após completar esta task, prosseguir para:
- **Task 11**: Testes de integração completos
- **Task 12**: Otimização de performance
- **Task 13**: Documentação final 