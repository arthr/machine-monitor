# Task 09: Descoberta Avançada de Aplicações Windows

## 📋 Objetivo
Implementar um sistema robusto de descoberta de aplicações instaladas no Windows, combinando múltiplas fontes de dados para obter informações completas e precisas.

## 🎯 Entregáveis
- [ ] Scanner de aplicações UWP (Microsoft Store)
- [ ] Detecção de aplicações portáveis
- [ ] Análise de metadados de aplicações
- [ ] Sistema de categorização automática
- [ ] Cache inteligente de descoberta
- [ ] Relatório de cobertura de descoberta

## 📊 Contexto
A task 06 implementou o básico de Registry scanning. Agora precisamos expandir para cobrir aplicações modernas do Windows, incluindo UWP apps, aplicações portáveis e melhorar a qualidade dos metadados coletados.

## 🔧 Implementação

### 1. Expandir `registry_windows.go`

```go
// Adicionar suporte para UWP apps
func (w *WindowsCollector) getUWPApps() ([]Application, error) {
    var apps []Application
    
    // PowerShell command para listar UWP apps
    cmd := exec.Command("powershell", "-Command", 
        "Get-AppxPackage | Where-Object {$_.Name -notlike '*Microsoft*' -and $_.Name -notlike '*Windows*'} | Select-Object Name, Version, Publisher, InstallLocation | ConvertTo-Json")
    
    output, err := cmd.Output()
    if err != nil {
        return apps, err
    }
    
    var uwpApps []UWPApp
    if err := json.Unmarshal(output, &uwpApps); err != nil {
        return apps, err
    }
    
    for _, uwpApp := range uwpApps {
        apps = append(apps, Application{
            Name:        uwpApp.Name,
            Version:     uwpApp.Version,
            Vendor:      uwpApp.Publisher,
            Path:        uwpApp.InstallLocation,
            Type:        "UWP",
            Category:    w.categorizeApp(uwpApp.Name),
        })
    }
    
    return apps, nil
}

type UWPApp struct {
    Name            string `json:"Name"`
    Version         string `json:"Version"`
    Publisher       string `json:"Publisher"`
    InstallLocation string `json:"InstallLocation"`
}
```

### 2. Criar `portable_apps_windows.go`

```go
//go:build windows

package collector

import (
    "os"
    "path/filepath"
    "strings"
    "context"
)

// PortableAppScanner escaneia aplicações portáveis
type PortableAppScanner struct {
    commonPaths []string
    logger      logging.Logger
}

func NewPortableAppScanner(logger logging.Logger) *PortableAppScanner {
    return &PortableAppScanner{
        commonPaths: []string{
            "C:\\PortableApps",
            "D:\\PortableApps",
            "E:\\PortableApps",
            os.Getenv("USERPROFILE") + "\\Desktop",
            os.Getenv("USERPROFILE") + "\\Downloads",
            "C:\\Tools",
            "C:\\Utils",
        },
        logger: logger,
    }
}

func (p *PortableAppScanner) ScanPortableApps(ctx context.Context) ([]Application, error) {
    var apps []Application
    
    for _, basePath := range p.commonPaths {
        if _, err := os.Stat(basePath); os.IsNotExist(err) {
            continue
        }
        
        err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
            if err != nil {
                return nil // Continue mesmo com erros
            }
            
            if info.IsDir() {
                return nil
            }
            
            if strings.HasSuffix(strings.ToLower(info.Name()), ".exe") {
                if app := p.analyzePortableApp(path, info); app != nil {
                    apps = append(apps, *app)
                }
            }
            
            return nil
        })
        
        if err != nil {
            p.logger.Warn("Erro ao escanear path portável", "path", basePath, "error", err)
        }
    }
    
    return apps, nil
}

func (p *PortableAppScanner) analyzePortableApp(path string, info os.FileInfo) *Application {
    // Verificar se é realmente uma aplicação portável
    if !p.isPortableApp(path) {
        return nil
    }
    
    version := p.extractVersionFromPath(path)
    name := p.extractNameFromPath(path)
    
    return &Application{
        Name:        name,
        Version:     version,
        Path:        path,
        Type:        "Portable",
        Size:        info.Size(),
        InstallDate: info.ModTime().Format("2006-01-02"),
        Category:    p.categorizeByPath(path),
    }
}

func (p *PortableAppScanner) isPortableApp(path string) bool {
    // Heurísticas para identificar apps portáveis
    pathLower := strings.ToLower(path)
    
    // Indicadores positivos
    portableIndicators := []string{
        "portable",
        "portableapps",
        "standalone",
        "noinstall",
    }
    
    for _, indicator := range portableIndicators {
        if strings.Contains(pathLower, indicator) {
            return true
        }
    }
    
    // Verificar se está em diretório próprio com recursos
    dir := filepath.Dir(path)
    entries, err := os.ReadDir(dir)
    if err != nil {
        return false
    }
    
    hasConfig := false
    hasResources := false
    
    for _, entry := range entries {
        name := strings.ToLower(entry.Name())
        if strings.Contains(name, "config") || strings.Contains(name, "settings") {
            hasConfig = true
        }
        if strings.HasSuffix(name, ".dll") || strings.HasSuffix(name, ".dat") {
            hasResources = true
        }
    }
    
    return hasConfig && hasResources
}
```

### 3. Criar `app_metadata_windows.go`

```go
//go:build windows

package collector

import (
    "os"
    "path/filepath"
    "strings"
    "syscall"
    "unsafe"
)

// AppMetadataAnalyzer analisa metadados detalhados de aplicações
type AppMetadataAnalyzer struct {
    logger logging.Logger
}

func NewAppMetadataAnalyzer(logger logging.Logger) *AppMetadataAnalyzer {
    return &AppMetadataAnalyzer{
        logger: logger,
    }
}

func (a *AppMetadataAnalyzer) EnrichApplicationData(app *Application) error {
    if app.Path == "" {
        return nil
    }
    
    // Obter informações de versão do arquivo
    if versionInfo, err := a.getFileVersionInfo(app.Path); err == nil {
        if app.Version == "" {
            app.Version = versionInfo.FileVersion
        }
        if app.Vendor == "" {
            app.Vendor = versionInfo.CompanyName
        }
        app.Description = versionInfo.FileDescription
        app.Copyright = versionInfo.LegalCopyright
    }
    
    // Analisar tamanho da instalação
    if size, err := a.calculateInstallSize(app.Path); err == nil {
        app.InstallSize = size
    }
    
    // Categorizar aplicação
    app.Category = a.categorizeApplication(app)
    
    // Verificar se é aplicação crítica do sistema
    app.IsSystemApp = a.isSystemApplication(app)
    
    return nil
}

type FileVersionInfo struct {
    FileVersion     string
    ProductVersion  string
    CompanyName     string
    FileDescription string
    LegalCopyright  string
    ProductName     string
}

func (a *AppMetadataAnalyzer) getFileVersionInfo(filePath string) (*FileVersionInfo, error) {
    // Usar Windows API para obter informações de versão
    kernel32 := syscall.NewLazyDLL("kernel32.dll")
    version := syscall.NewLazyDLL("version.dll")
    
    getFileVersionInfoSize := version.NewProc("GetFileVersionInfoSizeW")
    getFileVersionInfo := version.NewProc("GetFileVersionInfoW")
    verQueryValue := version.NewProc("VerQueryValueW")
    
    filePathPtr, _ := syscall.UTF16PtrFromString(filePath)
    
    // Obter tamanho necessário
    size, _, _ := getFileVersionInfoSize.Call(uintptr(unsafe.Pointer(filePathPtr)), 0)
    if size == 0 {
        return nil, fmt.Errorf("arquivo não tem informações de versão")
    }
    
    // Alocar buffer e obter informações
    buffer := make([]byte, size)
    ret, _, _ := getFileVersionInfo.Call(
        uintptr(unsafe.Pointer(filePathPtr)),
        0,
        size,
        uintptr(unsafe.Pointer(&buffer[0])),
    )
    
    if ret == 0 {
        return nil, fmt.Errorf("falha ao obter informações de versão")
    }
    
    // Extrair informações específicas
    info := &FileVersionInfo{}
    
    // Obter versão do arquivo
    if version := a.queryStringValue(buffer, "FileVersion"); version != "" {
        info.FileVersion = version
    }
    
    // Obter outras informações
    info.CompanyName = a.queryStringValue(buffer, "CompanyName")
    info.FileDescription = a.queryStringValue(buffer, "FileDescription")
    info.LegalCopyright = a.queryStringValue(buffer, "LegalCopyright")
    info.ProductName = a.queryStringValue(buffer, "ProductName")
    
    return info, nil
}

func (a *AppMetadataAnalyzer) categorizeApplication(app *Application) string {
    name := strings.ToLower(app.Name)
    path := strings.ToLower(app.Path)
    description := strings.ToLower(app.Description)
    
    categories := map[string][]string{
        "Development": {"visual studio", "code", "git", "docker", "python", "node", "java", "sdk"},
        "Media": {"vlc", "media player", "photoshop", "gimp", "audacity", "spotify"},
        "Gaming": {"steam", "epic", "origin", "ubisoft", "game", "gaming"},
        "Office": {"microsoft office", "word", "excel", "powerpoint", "libreoffice", "pdf"},
        "Browser": {"chrome", "firefox", "edge", "safari", "opera", "browser"},
        "Security": {"antivirus", "firewall", "vpn", "security", "defender"},
        "System": {"driver", "system", "windows", "microsoft", "intel", "nvidia", "amd"},
        "Communication": {"discord", "slack", "teams", "zoom", "skype", "whatsapp"},
        "Utility": {"winrar", "7zip", "notepad", "calculator", "cleaner", "optimizer"},
    }
    
    searchText := name + " " + path + " " + description
    
    for category, keywords := range categories {
        for _, keyword := range keywords {
            if strings.Contains(searchText, keyword) {
                return category
            }
        }
    }
    
    return "Other"
}

func (a *AppMetadataAnalyzer) isSystemApplication(app *Application) bool {
    systemPaths := []string{
        "c:\\windows\\",
        "c:\\program files\\windows",
        "c:\\program files (x86)\\windows",
        "c:\\program files\\microsoft\\",
        "c:\\program files (x86)\\microsoft\\",
    }
    
    pathLower := strings.ToLower(app.Path)
    
    for _, sysPath := range systemPaths {
        if strings.HasPrefix(pathLower, sysPath) {
            return true
        }
    }
    
    return false
}

func (a *AppMetadataAnalyzer) calculateInstallSize(appPath string) (int64, error) {
    var totalSize int64
    
    // Se é um arquivo, retornar seu tamanho
    if info, err := os.Stat(appPath); err == nil && !info.IsDir() {
        return info.Size(), nil
    }
    
    // Se é um diretório, calcular tamanho recursivamente
    err := filepath.Walk(appPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return nil // Continue mesmo com erros
        }
        if !info.IsDir() {
            totalSize += info.Size()
        }
        return nil
    })
    
    return totalSize, err
}
```

### 4. Atualizar `collector.go` para usar os novos scanners

```go
func (c *Collector) collectApplications(ctx context.Context) ([]Application, error) {
    var allApps []Application
    
    // Apps do Registry (já implementado)
    if registryApps, err := c.platform.CollectInstalledApps(ctx); err == nil {
        allApps = append(allApps, registryApps...)
    }
    
    // Apps UWP
    if uwpApps, err := c.platform.getUWPApps(); err == nil {
        allApps = append(allApps, uwpApps...)
    }
    
    // Apps portáveis
    portableScanner := NewPortableAppScanner(c.logger)
    if portableApps, err := portableScanner.ScanPortableApps(ctx); err == nil {
        allApps = append(allApps, portableApps...)
    }
    
    // Enriquecer com metadados
    metadataAnalyzer := NewAppMetadataAnalyzer(c.logger)
    for i := range allApps {
        metadataAnalyzer.EnrichApplicationData(&allApps[i])
    }
    
    // Remover duplicatas
    allApps = c.removeDuplicateApps(allApps)
    
    return allApps, nil
}

func (c *Collector) removeDuplicateApps(apps []Application) []Application {
    seen := make(map[string]bool)
    var unique []Application
    
    for _, app := range apps {
        key := strings.ToLower(app.Name + app.Version)
        if !seen[key] {
            seen[key] = true
            unique = append(unique, app)
        }
    }
    
    return unique
}
```

## ✅ Critérios de Sucesso

### Funcionalidade
- [ ] Descoberta de aplicações UWP funcionando
- [ ] Detecção de aplicações portáveis
- [ ] Metadados enriquecidos coletados
- [ ] Categorização automática precisa
- [ ] Remoção de duplicatas eficiente

### Performance
- [ ] Descoberta completa em < 30 segundos
- [ ] Uso de memória < 100MB durante scanning
- [ ] Cache de metadados funcionando

### Qualidade
- [ ] Cobertura > 95% das aplicações instaladas
- [ ] Precisão de categorização > 85%
- [ ] Detecção de apps sistema vs usuário

## 🧪 Testes

### Testes Unitários
```go
func TestUWPAppDiscovery(t *testing.T) {
    collector := NewWindowsCollector(logger, config)
    apps, err := collector.getUWPApps()
    
    assert.NoError(t, err)
    assert.NotEmpty(t, apps)
    
    // Verificar se encontrou apps conhecidos
    found := false
    for _, app := range apps {
        if strings.Contains(app.Name, "Calculator") {
            found = true
            assert.Equal(t, "UWP", app.Type)
            break
        }
    }
    assert.True(t, found, "Calculator UWP não encontrado")
}

func TestPortableAppDetection(t *testing.T) {
    scanner := NewPortableAppScanner(logger)
    
    // Criar app portável de teste
    testDir := t.TempDir()
    testApp := filepath.Join(testDir, "TestApp.exe")
    testConfig := filepath.Join(testDir, "config.ini")
    
    os.WriteFile(testApp, []byte("fake exe"), 0644)
    os.WriteFile(testConfig, []byte("fake config"), 0644)
    
    scanner.commonPaths = []string{testDir}
    
    apps, err := scanner.ScanPortableApps(context.Background())
    assert.NoError(t, err)
    assert.Len(t, apps, 1)
    assert.Equal(t, "Portable", apps[0].Type)
}
```

### Testes de Integração
```go
func TestCompleteAppDiscovery(t *testing.T) {
    collector := NewWindowsCollector(logger, config)
    
    apps, err := collector.collectApplications(context.Background())
    assert.NoError(t, err)
    assert.NotEmpty(t, apps)
    
    // Verificar tipos de aplicações encontradas
    types := make(map[string]int)
    categories := make(map[string]int)
    
    for _, app := range apps {
        types[app.Type]++
        categories[app.Category]++
    }
    
    // Deve encontrar pelo menos 3 tipos diferentes
    assert.GreaterOrEqual(t, len(types), 3)
    
    // Deve categorizar pelo menos 80% das apps
    uncategorized := categories["Other"]
    assert.LessOrEqual(t, uncategorized, len(apps)/5)
}
```

## 📚 Referências

### Documentação Windows
- [Get-AppxPackage PowerShell](https://docs.microsoft.com/en-us/powershell/module/appx/get-appxpackage)
- [Version Information API](https://docs.microsoft.com/en-us/windows/win32/api/winver/)
- [Windows Registry](https://docs.microsoft.com/en-us/windows/win32/sysinfo/registry)

### Bibliotecas Go
- [golang.org/x/sys/windows](https://pkg.go.dev/golang.org/x/sys/windows)
- [github.com/go-ole/go-ole](https://pkg.go.dev/github.com/go-ole/go-ole)

## 🔄 Próximos Passos
Após completar esta task, prosseguir para:
- **Task 10**: Testes de plataforma
- **Task 11**: Testes de integração
- **Task 12**: Otimização de performance 