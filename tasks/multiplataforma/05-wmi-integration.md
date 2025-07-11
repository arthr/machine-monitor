# Task 05: Integração com WMI (Windows Management Instrumentation)

## 📋 Objetivo
Implementar integração robusta com WMI para coleta de dados específicos do Windows, incluindo informações de hardware, sistema e aplicações.

## 🎯 Entregáveis
- [ ] Wrapper WMI implementado
- [ ] Queries WMI otimizadas
- [ ] Error handling robusto
- [ ] Testes de integração WMI

## 📊 Contexto
O WMI é a principal API para coleta de informações do sistema Windows. Precisamos implementar uma camada de abstração que facilite queries e trate erros adequadamente.

## 🔧 Implementação

### 1. Criar `internal/collector/wmi_windows.go`
```go
//go:build windows
// +build windows

package collector

import (
    "context"
    "fmt"
    "strings"
    "time"
    
    "github.com/go-ole/go-ole"
    "github.com/go-ole/go-ole/oleutil"
    "machine-monitor/internal/logging"
)

// WMIClient encapsula operações WMI
type WMIClient struct {
    logger   logging.Logger
    timeout  time.Duration
    retries  int
}

// NewWMIClient cria um novo cliente WMI
func NewWMIClient(logger logging.Logger) *WMIClient {
    return &WMIClient{
        logger:  logger,
        timeout: 30 * time.Second,
        retries: 3,
    }
}

// WMIResult representa o resultado de uma query WMI
type WMIResult struct {
    Properties map[string]interface{}
    Error      error
}

// QueryWMI executa uma query WMI e retorna os resultados
func (w *WMIClient) QueryWMI(ctx context.Context, query string) ([]WMIResult, error) {
    w.logger.Debug("Executing WMI query", map[string]interface{}{
        "query": query,
    })
    
    startTime := time.Now()
    defer func() {
        w.logger.Debug("WMI query completed", map[string]interface{}{
            "query":    query,
            "duration": time.Since(startTime).Milliseconds(),
        })
    }()
    
    // Inicializar COM
    err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize COM: %w", err)
    }
    defer ole.CoUninitialize()
    
    // Criar locator WMI
    unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
    if err != nil {
        return nil, fmt.Errorf("failed to create WMI locator: %w", err)
    }
    defer unknown.Release()
    
    wmi, err := unknown.QueryInterface(ole.IID_IDispatch)
    if err != nil {
        return nil, fmt.Errorf("failed to query WMI interface: %w", err)
    }
    defer wmi.Release()
    
    // Conectar ao namespace WMI
    serviceRaw, err := oleutil.CallMethod(wmi, "ConnectServer", nil, "root\\cimv2")
    if err != nil {
        return nil, fmt.Errorf("failed to connect to WMI service: %w", err)
    }
    service := serviceRaw.ToIDispatch()
    defer service.Release()
    
    // Executar query
    resultRaw, err := oleutil.CallMethod(service, "ExecQuery", query)
    if err != nil {
        return nil, fmt.Errorf("failed to execute WMI query: %w", err)
    }
    result := resultRaw.ToIDispatch()
    defer result.Release()
    
    // Processar resultados
    return w.processWMIResults(result)
}

// processWMIResults processa os resultados da query WMI
func (w *WMIClient) processWMIResults(result *ole.IDispatch) ([]WMIResult, error) {
    var results []WMIResult
    
    // Obter enumerador
    enumRaw, err := oleutil.CallMethod(result, "_NewEnum")
    if err != nil {
        return nil, fmt.Errorf("failed to get result enumerator: %w", err)
    }
    enum := enumRaw.ToIUnknown()
    defer enum.Release()
    
    // Iterar sobre resultados
    for {
        itemRaw, err := oleutil.CallMethod(enum, "Next")
        if err != nil {
            break
        }
        
        items := itemRaw.ToIDispatch()
        if items == nil {
            break
        }
        
        // Processar item
        properties, err := w.extractProperties(items)
        items.Release()
        
        if err != nil {
            w.logger.Warn("Failed to extract WMI properties", map[string]interface{}{
                "error": err.Error(),
            })
            continue
        }
        
        results = append(results, WMIResult{
            Properties: properties,
            Error:      nil,
        })
    }
    
    return results, nil
}

// extractProperties extrai propriedades de um objeto WMI
func (w *WMIClient) extractProperties(item *ole.IDispatch) (map[string]interface{}, error) {
    properties := make(map[string]interface{})
    
    // Obter propriedades do objeto
    propsRaw, err := oleutil.GetProperty(item, "Properties_")
    if err != nil {
        return nil, fmt.Errorf("failed to get properties: %w", err)
    }
    props := propsRaw.ToIDispatch()
    defer props.Release()
    
    // Obter contagem de propriedades
    countRaw, err := oleutil.GetProperty(props, "Count")
    if err != nil {
        return nil, fmt.Errorf("failed to get property count: %w", err)
    }
    count := int(countRaw.Val)
    
    // Iterar sobre propriedades
    for i := 0; i < count; i++ {
        propRaw, err := oleutil.CallMethod(props, "Item", i)
        if err != nil {
            continue
        }
        prop := propRaw.ToIDispatch()
        
        // Obter nome da propriedade
        nameRaw, err := oleutil.GetProperty(prop, "Name")
        if err != nil {
            prop.Release()
            continue
        }
        name := nameRaw.ToString()
        
        // Obter valor da propriedade
        valueRaw, err := oleutil.GetProperty(prop, "Value")
        if err != nil {
            prop.Release()
            continue
        }
        
        properties[name] = w.convertWMIValue(valueRaw)
        prop.Release()
    }
    
    return properties, nil
}

// convertWMIValue converte valores WMI para tipos Go
func (w *WMIClient) convertWMIValue(value *ole.VARIANT) interface{} {
    if value == nil {
        return nil
    }
    
    switch value.VT {
    case ole.VT_NULL:
        return nil
    case ole.VT_BSTR:
        return value.ToString()
    case ole.VT_I4:
        return int(value.Val)
    case ole.VT_UI4:
        return uint32(value.Val)
    case ole.VT_I8:
        return int64(value.Val)
    case ole.VT_UI8:
        return uint64(value.Val)
    case ole.VT_BOOL:
        return value.Val != 0
    case ole.VT_R8:
        return float64(value.Val)
    default:
        return value.ToString()
    }
}

// QuerySingleValue executa query e retorna valor único
func (w *WMIClient) QuerySingleValue(ctx context.Context, query string, property string) (interface{}, error) {
    results, err := w.QueryWMI(ctx, query)
    if err != nil {
        return nil, err
    }
    
    if len(results) == 0 {
        return nil, fmt.Errorf("no results found for query: %s", query)
    }
    
    if value, exists := results[0].Properties[property]; exists {
        return value, nil
    }
    
    return nil, fmt.Errorf("property %s not found in results", property)
}

// QueryMultipleValues executa query e retorna múltiplos valores
func (w *WMIClient) QueryMultipleValues(ctx context.Context, query string, properties []string) ([]map[string]interface{}, error) {
    results, err := w.QueryWMI(ctx, query)
    if err != nil {
        return nil, err
    }
    
    var values []map[string]interface{}
    for _, result := range results {
        item := make(map[string]interface{})
        for _, prop := range properties {
            if value, exists := result.Properties[prop]; exists {
                item[prop] = value
            }
        }
        values = append(values, item)
    }
    
    return values, nil
}

// Common WMI queries
const (
    // Sistema
    QuerySystemInfo = "SELECT * FROM Win32_ComputerSystem"
    QueryOSInfo     = "SELECT * FROM Win32_OperatingSystem"
    QueryBIOSInfo   = "SELECT * FROM Win32_BIOS"
    
    // Hardware
    QueryCPUInfo        = "SELECT * FROM Win32_Processor"
    QueryMemoryInfo     = "SELECT * FROM Win32_PhysicalMemory"
    QueryDiskInfo       = "SELECT * FROM Win32_LogicalDisk"
    QueryNetworkInfo    = "SELECT * FROM Win32_NetworkAdapter WHERE NetEnabled=True"
    
    // Software
    QueryInstalledApps  = "SELECT * FROM Win32_Product"
    QueryServices       = "SELECT * FROM Win32_Service"
    QueryProcesses      = "SELECT * FROM Win32_Process"
    
    // Identificação
    QueryComputerProduct = "SELECT UUID FROM Win32_ComputerSystemProduct"
    QueryMotherboard     = "SELECT SerialNumber FROM Win32_BaseBoard"
    QueryBIOSSerial      = "SELECT SerialNumber FROM Win32_BIOS"
)

// WMIQueries contém queries pré-definidas
var WMIQueries = map[string]string{
    "system_info":       QuerySystemInfo,
    "os_info":          QueryOSInfo,
    "bios_info":        QueryBIOSInfo,
    "cpu_info":         QueryCPUInfo,
    "memory_info":      QueryMemoryInfo,
    "disk_info":        QueryDiskInfo,
    "network_info":     QueryNetworkInfo,
    "installed_apps":   QueryInstalledApps,
    "services":         QueryServices,
    "processes":        QueryProcesses,
    "computer_product": QueryComputerProduct,
    "motherboard":      QueryMotherboard,
    "bios_serial":      QueryBIOSSerial,
}

// GetWMIQuery retorna query pré-definida
func GetWMIQuery(name string) (string, bool) {
    query, exists := WMIQueries[name]
    return query, exists
}

// ValidateWMIQuery valida sintaxe básica de query WMI
func ValidateWMIQuery(query string) error {
    query = strings.TrimSpace(strings.ToUpper(query))
    
    if !strings.HasPrefix(query, "SELECT") {
        return fmt.Errorf("WMI query must start with SELECT")
    }
    
    if !strings.Contains(query, "FROM") {
        return fmt.Errorf("WMI query must contain FROM clause")
    }
    
    // Verificar classes WMI comuns
    validClasses := []string{
        "WIN32_COMPUTERSYSTEM",
        "WIN32_OPERATINGSYSTEM",
        "WIN32_BIOS",
        "WIN32_PROCESSOR",
        "WIN32_PHYSICALMEMORY",
        "WIN32_LOGICALDISK",
        "WIN32_NETWORKADAPTER",
        "WIN32_PRODUCT",
        "WIN32_SERVICE",
        "WIN32_PROCESS",
        "WIN32_COMPUTERSYSTEMPRODUCT",
        "WIN32_BASEBOARD",
    }
    
    validClass := false
    for _, class := range validClasses {
        if strings.Contains(query, class) {
            validClass = true
            break
        }
    }
    
    if !validClass {
        return fmt.Errorf("WMI query contains unknown or potentially unsafe class")
    }
    
    return nil
}
```

### 2. Criar `internal/collector/wmi_helpers_windows.go`
```go
//go:build windows
// +build windows

package collector

import (
    "context"
    "fmt"
    "strings"
    "time"
)

// WMIHelper fornece métodos de conveniência para WMI
type WMIHelper struct {
    client *WMIClient
    logger logging.Logger
}

// NewWMIHelper cria um novo helper WMI
func NewWMIHelper(logger logging.Logger) *WMIHelper {
    return &WMIHelper{
        client: NewWMIClient(logger),
        logger: logger,
    }
}

// GetSystemUUID obtém UUID único do sistema
func (h *WMIHelper) GetSystemUUID(ctx context.Context) (string, error) {
    // Tentar UUID da motherboard primeiro
    if uuid, err := h.getMotherboardUUID(ctx); err == nil && uuid != "" {
        return fmt.Sprintf("mb-%s", uuid), nil
    }
    
    // Tentar serial do BIOS
    if serial, err := h.getBIOSSerial(ctx); err == nil && serial != "" {
        return fmt.Sprintf("bios-%s", serial), nil
    }
    
    // Tentar UUID do produto
    if uuid, err := h.getProductUUID(ctx); err == nil && uuid != "" {
        return fmt.Sprintf("prod-%s", uuid), nil
    }
    
    return "", fmt.Errorf("failed to obtain system UUID from any source")
}

// getMotherboardUUID obtém UUID da motherboard
func (h *WMIHelper) getMotherboardUUID(ctx context.Context) (string, error) {
    value, err := h.client.QuerySingleValue(ctx, QueryComputerProduct, "UUID")
    if err != nil {
        return "", err
    }
    
    if uuid, ok := value.(string); ok && uuid != "" {
        return strings.ReplaceAll(uuid, "-", ""), nil
    }
    
    return "", fmt.Errorf("invalid UUID format")
}

// getBIOSSerial obtém serial do BIOS
func (h *WMIHelper) getBIOSSerial(ctx context.Context) (string, error) {
    value, err := h.client.QuerySingleValue(ctx, QueryBIOSSerial, "SerialNumber")
    if err != nil {
        return "", err
    }
    
    if serial, ok := value.(string); ok && serial != "" {
        return serial, nil
    }
    
    return "", fmt.Errorf("BIOS serial not available")
}

// getProductUUID obtém UUID do produto
func (h *WMIHelper) getProductUUID(ctx context.Context) (string, error) {
    results, err := h.client.QueryWMI(ctx, QuerySystemInfo)
    if err != nil {
        return "", err
    }
    
    if len(results) > 0 {
        if name, exists := results[0].Properties["Name"]; exists {
            if nameStr, ok := name.(string); ok {
                return fmt.Sprintf("sys-%s", nameStr), nil
            }
        }
    }
    
    return "", fmt.Errorf("system name not available")
}

// GetInstalledApplications obtém lista de aplicações instaladas
func (h *WMIHelper) GetInstalledApplications(ctx context.Context) ([]Application, error) {
    properties := []string{"Name", "Version", "Vendor", "InstallDate", "InstallLocation"}
    
    results, err := h.client.QueryMultipleValues(ctx, QueryInstalledApps, properties)
    if err != nil {
        return nil, err
    }
    
    var apps []Application
    for _, result := range results {
        app := Application{
            Name:        h.getStringValue(result, "Name"),
            Version:     h.getStringValue(result, "Version"),
            Vendor:      h.getStringValue(result, "Vendor"),
            InstallDate: h.getStringValue(result, "InstallDate"),
            Path:        h.getStringValue(result, "InstallLocation"),
            Type:        "system",
        }
        
        if app.Name != "" {
            apps = append(apps, app)
        }
    }
    
    return apps, nil
}

// GetSystemServices obtém lista de serviços do sistema
func (h *WMIHelper) GetSystemServices(ctx context.Context) ([]Service, error) {
    properties := []string{"Name", "DisplayName", "State", "StartMode", "ProcessId", "PathName", "Description"}
    
    results, err := h.client.QueryMultipleValues(ctx, QueryServices, properties)
    if err != nil {
        return nil, err
    }
    
    var services []Service
    for _, result := range results {
        service := Service{
            Name:        h.getStringValue(result, "Name"),
            DisplayName: h.getStringValue(result, "DisplayName"),
            Status:      h.getStringValue(result, "State"),
            StartType:   h.getStringValue(result, "StartMode"),
            ProcessID:   h.getIntValue(result, "ProcessId"),
            Path:        h.getStringValue(result, "PathName"),
            Description: h.getStringValue(result, "Description"),
        }
        
        if service.Name != "" {
            services = append(services, service)
        }
    }
    
    return services, nil
}

// GetSystemInformation obtém informações gerais do sistema
func (h *WMIHelper) GetSystemInformation(ctx context.Context) (map[string]interface{}, error) {
    info := make(map[string]interface{})
    
    // Informações do sistema
    if systemInfo, err := h.client.QueryWMI(ctx, QuerySystemInfo); err == nil && len(systemInfo) > 0 {
        info["computer_system"] = systemInfo[0].Properties
    }
    
    // Informações do OS
    if osInfo, err := h.client.QueryWMI(ctx, QueryOSInfo); err == nil && len(osInfo) > 0 {
        info["operating_system"] = osInfo[0].Properties
    }
    
    // Informações do BIOS
    if biosInfo, err := h.client.QueryWMI(ctx, QueryBIOSInfo); err == nil && len(biosInfo) > 0 {
        info["bios"] = biosInfo[0].Properties
    }
    
    // Informações do CPU
    if cpuInfo, err := h.client.QueryWMI(ctx, QueryCPUInfo); err == nil && len(cpuInfo) > 0 {
        info["processor"] = cpuInfo[0].Properties
    }
    
    return info, nil
}

// Funções auxiliares para conversão de tipos
func (h *WMIHelper) getStringValue(data map[string]interface{}, key string) string {
    if value, exists := data[key]; exists {
        if str, ok := value.(string); ok {
            return strings.TrimSpace(str)
        }
    }
    return ""
}

func (h *WMIHelper) getIntValue(data map[string]interface{}, key string) int {
    if value, exists := data[key]; exists {
        switch v := value.(type) {
        case int:
            return v
        case int32:
            return int(v)
        case int64:
            return int(v)
        case uint32:
            return int(v)
        case uint64:
            return int(v)
        }
    }
    return 0
}

func (h *WMIHelper) getBoolValue(data map[string]interface{}, key string) bool {
    if value, exists := data[key]; exists {
        if b, ok := value.(bool); ok {
            return b
        }
    }
    return false
}
```

## 📋 Checklist de Implementação

### Arquivos a Criar
- [ ] `internal/collector/wmi_windows.go` - Cliente WMI principal
- [ ] `internal/collector/wmi_helpers_windows.go` - Helpers de conveniência

### Funcionalidades WMI
- [ ] Conexão e inicialização COM
- [ ] Execução de queries WMI
- [ ] Processamento de resultados
- [ ] Conversão de tipos
- [ ] Error handling robusto

### Queries Implementadas
- [ ] System UUID (múltiplas fontes)
- [ ] Aplicações instaladas
- [ ] Serviços do sistema
- [ ] Informações do sistema
- [ ] Hardware information

### Validações
- [ ] Queries WMI funcionam corretamente
- [ ] Conversão de tipos está correta
- [ ] Error handling é robusto
- [ ] Performance é adequada

## 🎯 Critérios de Sucesso
- [ ] Integração WMI funcional e robusta
- [ ] Queries otimizadas e seguras
- [ ] Tratamento de erros adequado
- [ ] Performance aceitável (<5s para queries básicas)

## 📚 Referências
- [WMI Classes](https://docs.microsoft.com/en-us/windows/win32/cimwin32prov/win32-provider) - Classes WMI
- [go-ole Documentation](https://pkg.go.dev/github.com/go-ole/go-ole) - Biblioteca OLE
- [WQL Reference](https://docs.microsoft.com/en-us/windows/win32/wmisdk/wql-sql-for-wmi) - WQL syntax

## ⏭️ Próxima Task
[06-registry-scanning.md](06-registry-scanning.md) - Implementar scanning do Registry Windows 