# Task 08: Implementar Coleta de Serviços Windows via WMI

## 📋 Objetivo
Implementar coleta detalhada de serviços do sistema Windows usando WMI, com filtragem, categorização e análise de dependências.

## 🎯 Entregáveis
- [ ] Coleta de serviços via WMI otimizada
- [ ] Filtragem e categorização de serviços
- [ ] Análise de dependências de serviços
- [ ] Métricas de performance dos serviços

## 📊 Contexto
Os serviços Windows são componentes críticos do sistema. Precisamos coletar informações detalhadas sobre estado, configuração e dependências dos serviços.

## 🔧 Implementação

### 1. Expandir `internal/collector/wmi_helpers_windows.go`
```go
// Adicionar ao arquivo existente

// ServiceInfo representa informações detalhadas de um serviço
type ServiceInfo struct {
    Name           string                 `json:"name"`
    DisplayName    string                 `json:"display_name"`
    Description    string                 `json:"description"`
    State          string                 `json:"state"`
    StartMode      string                 `json:"start_mode"`
    ServiceType    string                 `json:"service_type"`
    ProcessID      int                    `json:"process_id,omitempty"`
    PathName       string                 `json:"path_name,omitempty"`
    StartName      string                 `json:"start_name,omitempty"`
    ErrorControl   string                 `json:"error_control,omitempty"`
    Dependencies   []string               `json:"dependencies,omitempty"`
    Dependents     []string               `json:"dependents,omitempty"`
    Performance    *ServicePerformance    `json:"performance,omitempty"`
    Category       string                 `json:"category"`
    Criticality    string                 `json:"criticality"`
}

// ServicePerformance representa métricas de performance de um serviço
type ServicePerformance struct {
    MemoryUsage    uint64 `json:"memory_usage,omitempty"`
    CPUTime        uint64 `json:"cpu_time,omitempty"`
    HandleCount    uint32 `json:"handle_count,omitempty"`
    ThreadCount    uint32 `json:"thread_count,omitempty"`
    StartTime      string `json:"start_time,omitempty"`
}

// GetDetailedSystemServices obtém informações detalhadas dos serviços
func (h *WMIHelper) GetDetailedSystemServices(ctx context.Context) ([]ServiceInfo, error) {
    h.logger.Info("Collecting detailed Windows services information")
    
    // Query para informações básicas dos serviços
    serviceQuery := `SELECT Name, DisplayName, Description, State, StartMode, ServiceType, 
                     ProcessId, PathName, StartName, ErrorControl FROM Win32_Service`
    
    properties := []string{
        "Name", "DisplayName", "Description", "State", "StartMode", 
        "ServiceType", "ProcessId", "PathName", "StartName", "ErrorControl",
    }
    
    results, err := h.client.QueryMultipleValues(ctx, serviceQuery, properties)
    if err != nil {
        return nil, fmt.Errorf("failed to query services: %w", err)
    }
    
    var services []ServiceInfo
    for _, result := range results {
        service := ServiceInfo{
            Name:         h.getStringValue(result, "Name"),
            DisplayName:  h.getStringValue(result, "DisplayName"),
            Description:  h.getStringValue(result, "Description"),
            State:        h.getStringValue(result, "State"),
            StartMode:    h.getStringValue(result, "StartMode"),
            ServiceType:  h.getStringValue(result, "ServiceType"),
            ProcessID:    h.getIntValue(result, "ProcessId"),
            PathName:     h.getStringValue(result, "PathName"),
            StartName:    h.getStringValue(result, "StartName"),
            ErrorControl: h.getStringValue(result, "ErrorControl"),
        }
        
        if service.Name != "" {
            // Enriquecer com informações adicionais
            h.enrichServiceInfo(ctx, &service)
            services = append(services, service)
        }
    }
    
    // Coletar dependências
    h.collectServiceDependencies(ctx, services)
    
    h.logger.Info("Service collection completed", map[string]interface{}{
        "total_services": len(services),
        "running":        h.countServicesByState(services, "Running"),
        "stopped":        h.countServicesByState(services, "Stopped"),
    })
    
    return services, nil
}

// enrichServiceInfo enriquece informações do serviço
func (h *WMIHelper) enrichServiceInfo(ctx context.Context, service *ServiceInfo) {
    // Categorizar serviço
    service.Category = h.categorizeService(service)
    service.Criticality = h.assessServiceCriticality(service)
    
    // Coletar métricas de performance se o serviço estiver rodando
    if service.State == "Running" && service.ProcessID > 0 {
        if perf, err := h.getServicePerformance(ctx, service.ProcessID); err == nil {
            service.Performance = perf
        }
    }
}

// categorizeService categoriza o serviço baseado em seu nome e caminho
func (h *WMIHelper) categorizeService(service *ServiceInfo) string {
    name := strings.ToLower(service.Name)
    path := strings.ToLower(service.PathName)
    
    categories := map[string][]string{
        "system": {
            "winlogon", "csrss", "wininit", "services", "lsass", "dwm",
            "explorer", "svchost", "system", "registry", "smss",
        },
        "security": {
            "wscsvc", "wuauserv", "cryptsvc", "bits", "msiserver",
            "trustedinstaller", "wdnissvc", "windefend", "mpssvc",
        },
        "network": {
            "lanmanserver", "lanmanworkstation", "dnscache", "dhcp",
            "netlogon", "netman", "nlasvc", "iphlpsvc", "w32time",
        },
        "audio": {
            "audiosrv", "audioendpointbuilder", "mmcss",
        },
        "print": {
            "spooler", "printnotify",
        },
        "storage": {
            "vss", "swprv", "msdtc", "vds",
        },
        "application": {
            "mysql", "apache", "nginx", "iis", "sql", "oracle",
        },
        "hardware": {
            "pnrpsvc", "upnphost", "ssdpsrv", "fdrespub",
        },
    }
    
    for category, keywords := range categories {
        for _, keyword := range keywords {
            if strings.Contains(name, keyword) || strings.Contains(path, keyword) {
                return category
            }
        }
    }
    
    return "other"
}

// assessServiceCriticality avalia a criticidade do serviço
func (h *WMIHelper) assessServiceCriticality(service *ServiceInfo) string {
    name := strings.ToLower(service.Name)
    startMode := strings.ToLower(service.StartMode)
    
    // Serviços críticos do sistema
    criticalServices := []string{
        "csrss", "wininit", "services", "lsass", "winlogon",
        "dwm", "explorer", "system", "registry", "smss",
    }
    
    // Serviços importantes
    importantServices := []string{
        "eventlog", "rpcss", "dcomlaunch", "plugplay",
        "cryptsvc", "bits", "wuauserv", "dnscache",
    }
    
    for _, critical := range criticalServices {
        if strings.Contains(name, critical) {
            return "critical"
        }
    }
    
    for _, important := range importantServices {
        if strings.Contains(name, important) {
            return "important"
        }
    }
    
    // Serviços automáticos são geralmente importantes
    if startMode == "auto" || startMode == "automatic" {
        return "normal"
    }
    
    return "low"
}

// getServicePerformance obtém métricas de performance do serviço
func (h *WMIHelper) getServicePerformance(ctx context.Context, processID int) (*ServicePerformance, error) {
    query := fmt.Sprintf("SELECT WorkingSetSize, PageFileUsage, HandleCount, ThreadCount, CreationDate FROM Win32_Process WHERE ProcessId = %d", processID)
    
    results, err := h.client.QueryWMI(ctx, query)
    if err != nil || len(results) == 0 {
        return nil, err
    }
    
    result := results[0].Properties
    
    perf := &ServicePerformance{
        MemoryUsage: h.getUint64Value(result, "WorkingSetSize"),
        HandleCount: h.getUint32Value(result, "HandleCount"),
        ThreadCount: h.getUint32Value(result, "ThreadCount"),
    }
    
    // Converter data de criação
    if creationDate, ok := result["CreationDate"].(string); ok {
        perf.StartTime = h.convertWMIDateTime(creationDate)
    }
    
    return perf, nil
}

// collectServiceDependencies coleta dependências entre serviços
func (h *WMIHelper) collectServiceDependencies(ctx context.Context, services []ServiceInfo) {
    h.logger.Debug("Collecting service dependencies")
    
    // Query para dependências
    depQuery := "SELECT Antecedent, Dependent FROM Win32_DependentService"
    
    results, err := h.client.QueryWMI(ctx, depQuery)
    if err != nil {
        h.logger.Warn("Failed to collect service dependencies", map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    
    // Mapear serviços por nome para acesso rápido
    serviceMap := make(map[string]*ServiceInfo)
    for i := range services {
        serviceMap[services[i].Name] = &services[i]
    }
    
    // Processar dependências
    for _, result := range results {
        antecedent := h.extractServiceNameFromPath(h.getStringValue(result.Properties, "Antecedent"))
        dependent := h.extractServiceNameFromPath(h.getStringValue(result.Properties, "Dependent"))
        
        if antecedentService, exists := serviceMap[antecedent]; exists {
            antecedentService.Dependents = append(antecedentService.Dependents, dependent)
        }
        
        if dependentService, exists := serviceMap[dependent]; exists {
            dependentService.Dependencies = append(dependentService.Dependencies, antecedent)
        }
    }
}

// extractServiceNameFromPath extrai nome do serviço do caminho WMI
func (h *WMIHelper) extractServiceNameFromPath(path string) string {
    // Formato: Win32_Service.Name="ServiceName"
    if strings.Contains(path, `Name="`) {
        start := strings.Index(path, `Name="`) + 6
        end := strings.Index(path[start:], `"`)
        if end > 0 {
            return path[start : start+end]
        }
    }
    return ""
}

// convertWMIDateTime converte data/hora WMI para formato legível
func (h *WMIHelper) convertWMIDateTime(wmiDate string) string {
    // Formato WMI: YYYYMMDDHHMMSS.mmmmmm+UUU
    if len(wmiDate) >= 14 {
        year := wmiDate[0:4]
        month := wmiDate[4:6]
        day := wmiDate[6:8]
        hour := wmiDate[8:10]
        minute := wmiDate[10:12]
        second := wmiDate[12:14]
        
        return fmt.Sprintf("%s-%s-%s %s:%s:%s", year, month, day, hour, minute, second)
    }
    return wmiDate
}

// countServicesByState conta serviços por estado
func (h *WMIHelper) countServicesByState(services []ServiceInfo, state string) int {
    count := 0
    for _, service := range services {
        if strings.EqualFold(service.State, state) {
            count++
        }
    }
    return count
}

// GetServiceStatistics obtém estatísticas dos serviços
func (h *WMIHelper) GetServiceStatistics(ctx context.Context, services []ServiceInfo) map[string]interface{} {
    stats := make(map[string]interface{})
    
    // Contadores por estado
    stateCount := make(map[string]int)
    categoryCount := make(map[string]int)
    criticalityCount := make(map[string]int)
    startModeCount := make(map[string]int)
    
    var totalMemory uint64
    runningServices := 0
    
    for _, service := range services {
        stateCount[service.State]++
        categoryCount[service.Category]++
        criticalityCount[service.Criticality]++
        startModeCount[service.StartMode]++
        
        if service.State == "Running" {
            runningServices++
            if service.Performance != nil {
                totalMemory += service.Performance.MemoryUsage
            }
        }
    }
    
    stats["total_services"] = len(services)
    stats["running_services"] = runningServices
    stats["total_memory_usage"] = totalMemory
    stats["states"] = stateCount
    stats["categories"] = categoryCount
    stats["criticality"] = criticalityCount
    stats["start_modes"] = startModeCount
    
    return stats
}

// Helper functions para conversão de tipos
func (h *WMIHelper) getUint64Value(data map[string]interface{}, key string) uint64 {
    if value, exists := data[key]; exists {
        switch v := value.(type) {
        case uint64:
            return v
        case int64:
            return uint64(v)
        case int:
            return uint64(v)
        case uint32:
            return uint64(v)
        case int32:
            return uint64(v)
        }
    }
    return 0
}

func (h *WMIHelper) getUint32Value(data map[string]interface{}, key string) uint32 {
    if value, exists := data[key]; exists {
        switch v := value.(type) {
        case uint32:
            return v
        case int32:
            return uint32(v)
        case int:
            return uint32(v)
        case uint64:
            return uint32(v)
        case int64:
            return uint32(v)
        }
    }
    return 0
}
```

### 2. Criar `internal/collector/services_analyzer_windows.go`
```go
//go:build windows
// +build windows

package collector

import (
    "context"
    "sort"
    "strings"
    
    "machine-monitor/internal/logging"
)

// ServiceAnalyzer analisa serviços do sistema
type ServiceAnalyzer struct {
    logger logging.Logger
}

// NewServiceAnalyzer cria um novo analisador de serviços
func NewServiceAnalyzer(logger logging.Logger) *ServiceAnalyzer {
    return &ServiceAnalyzer{
        logger: logger,
    }
}

// ServiceAnalysis representa análise dos serviços
type ServiceAnalysis struct {
    TotalServices     int                    `json:"total_services"`
    RunningServices   int                    `json:"running_services"`
    CriticalServices  []ServiceInfo          `json:"critical_services"`
    ProblematicServices []ServiceInfo        `json:"problematic_services"`
    HighMemoryServices []ServiceInfo         `json:"high_memory_services"`
    Statistics        map[string]interface{} `json:"statistics"`
    Recommendations   []string               `json:"recommendations"`
}

// AnalyzeServices realiza análise completa dos serviços
func (a *ServiceAnalyzer) AnalyzeServices(ctx context.Context, services []ServiceInfo) *ServiceAnalysis {
    a.logger.Info("Analyzing Windows services")
    
    analysis := &ServiceAnalysis{
        TotalServices:   len(services),
        RunningServices: a.countRunningServices(services),
        Statistics:      make(map[string]interface{}),
    }
    
    // Identificar serviços críticos
    analysis.CriticalServices = a.findCriticalServices(services)
    
    // Identificar serviços problemáticos
    analysis.ProblematicServices = a.findProblematicServices(services)
    
    // Identificar serviços com alto uso de memória
    analysis.HighMemoryServices = a.findHighMemoryServices(services)
    
    // Gerar estatísticas
    analysis.Statistics = a.generateStatistics(services)
    
    // Gerar recomendações
    analysis.Recommendations = a.generateRecommendations(services)
    
    a.logger.Info("Service analysis completed", map[string]interface{}{
        "total":       analysis.TotalServices,
        "running":     analysis.RunningServices,
        "critical":    len(analysis.CriticalServices),
        "problematic": len(analysis.ProblematicServices),
    })
    
    return analysis
}

// countRunningServices conta serviços em execução
func (a *ServiceAnalyzer) countRunningServices(services []ServiceInfo) int {
    count := 0
    for _, service := range services {
        if service.State == "Running" {
            count++
        }
    }
    return count
}

// findCriticalServices encontra serviços críticos
func (a *ServiceAnalyzer) findCriticalServices(services []ServiceInfo) []ServiceInfo {
    var critical []ServiceInfo
    
    for _, service := range services {
        if service.Criticality == "critical" {
            critical = append(critical, service)
        }
    }
    
    // Ordenar por nome
    sort.Slice(critical, func(i, j int) bool {
        return critical[i].Name < critical[j].Name
    })
    
    return critical
}

// findProblematicServices encontra serviços com problemas
func (a *ServiceAnalyzer) findProblematicServices(services []ServiceInfo) []ServiceInfo {
    var problematic []ServiceInfo
    
    for _, service := range services {
        // Serviço crítico parado
        if service.Criticality == "critical" && service.State != "Running" {
            problematic = append(problematic, service)
            continue
        }
        
        // Serviço automático parado
        if strings.ToLower(service.StartMode) == "auto" && service.State == "Stopped" {
            problematic = append(problematic, service)
            continue
        }
        
        // Serviço com muitas dependências paradas
        if len(service.Dependencies) > 3 && service.State == "Stopped" {
            problematic = append(problematic, service)
        }
    }
    
    return problematic
}

// findHighMemoryServices encontra serviços com alto uso de memória
func (a *ServiceAnalyzer) findHighMemoryServices(services []ServiceInfo) []ServiceInfo {
    var highMemory []ServiceInfo
    const memoryThreshold = 100 * 1024 * 1024 // 100MB
    
    for _, service := range services {
        if service.Performance != nil && service.Performance.MemoryUsage > memoryThreshold {
            highMemory = append(highMemory, service)
        }
    }
    
    // Ordenar por uso de memória (decrescente)
    sort.Slice(highMemory, func(i, j int) bool {
        return highMemory[i].Performance.MemoryUsage > highMemory[j].Performance.MemoryUsage
    })
    
    // Retornar apenas os top 10
    if len(highMemory) > 10 {
        highMemory = highMemory[:10]
    }
    
    return highMemory
}

// generateStatistics gera estatísticas detalhadas
func (a *ServiceAnalyzer) generateStatistics(services []ServiceInfo) map[string]interface{} {
    stats := make(map[string]interface{})
    
    // Contadores
    stateCount := make(map[string]int)
    categoryCount := make(map[string]int)
    criticalityCount := make(map[string]int)
    startModeCount := make(map[string]int)
    
    var totalMemory uint64
    var totalHandles uint32
    var totalThreads uint32
    
    for _, service := range services {
        stateCount[service.State]++
        categoryCount[service.Category]++
        criticalityCount[service.Criticality]++
        startModeCount[service.StartMode]++
        
        if service.Performance != nil {
            totalMemory += service.Performance.MemoryUsage
            totalHandles += service.Performance.HandleCount
            totalThreads += service.Performance.ThreadCount
        }
    }
    
    stats["states"] = stateCount
    stats["categories"] = categoryCount
    stats["criticality"] = criticalityCount
    stats["start_modes"] = startModeCount
    stats["total_memory_usage"] = totalMemory
    stats["total_handles"] = totalHandles
    stats["total_threads"] = totalThreads
    
    return stats
}

// generateRecommendations gera recomendações baseadas na análise
func (a *ServiceAnalyzer) generateRecommendations(services []ServiceInfo) []string {
    var recommendations []string
    
    // Analisar serviços parados críticos
    criticalStopped := 0
    for _, service := range services {
        if service.Criticality == "critical" && service.State != "Running" {
            criticalStopped++
        }
    }
    
    if criticalStopped > 0 {
        recommendations = append(recommendations, 
            fmt.Sprintf("Verificar %d serviços críticos que estão parados", criticalStopped))
    }
    
    // Analisar uso de memória
    highMemoryCount := 0
    var totalMemory uint64
    for _, service := range services {
        if service.Performance != nil {
            totalMemory += service.Performance.MemoryUsage
            if service.Performance.MemoryUsage > 100*1024*1024 { // 100MB
                highMemoryCount++
            }
        }
    }
    
    if highMemoryCount > 5 {
        recommendations = append(recommendations, 
            "Múltiplos serviços com alto uso de memória detectados")
    }
    
    // Analisar serviços desnecessários
    unnecessaryCount := 0
    for _, service := range services {
        if service.StartMode == "Manual" && service.State == "Running" && 
           service.Category == "other" && len(service.Dependents) == 0 {
            unnecessaryCount++
        }
    }
    
    if unnecessaryCount > 10 {
        recommendations = append(recommendations, 
            "Considerar desabilitar serviços desnecessários em execução")
    }
    
    // Analisar dependências
    brokenDependencies := 0
    for _, service := range services {
        if service.State == "Running" && len(service.Dependencies) > 0 {
            // Verificar se dependências estão rodando
            // (implementação simplificada)
            brokenDependencies++
        }
    }
    
    if len(recommendations) == 0 {
        recommendations = append(recommendations, "Sistema de serviços funcionando adequadamente")
    }
    
    return recommendations
}
```

## 📋 Checklist de Implementação

### Funcionalidades de Serviços
- [ ] Coleta detalhada via WMI
- [ ] Categorização automática
- [ ] Análise de criticidade
- [ ] Métricas de performance
- [ ] Mapeamento de dependências

### Análise e Insights
- [ ] Identificação de serviços problemáticos
- [ ] Análise de uso de recursos
- [ ] Recomendações automáticas
- [ ] Estatísticas detalhadas

### Performance
- [ ] Queries WMI otimizadas
- [ ] Coleta paralela quando possível
- [ ] Cache de resultados
- [ ] Timeout adequado

## 🎯 Critérios de Sucesso
- [ ] Coleta completa de serviços Windows
- [ ] Categorização precisa
- [ ] Análise útil e acionável
- [ ] Performance adequada (<15s)

## 📚 Referências
- [Win32_Service Class](https://docs.microsoft.com/en-us/windows/win32/cimwin32prov/win32-service) - WMI Service class
- [Service Dependencies](https://docs.microsoft.com/en-us/windows/win32/services/service-dependencies) - Dependências de serviços
- [Service Control Manager](https://docs.microsoft.com/en-us/windows/win32/services/service-control-manager) - SCM

## ⏭️ Próxima Task
[09-windows-apps-discovery.md](09-windows-apps-discovery.md) - Descoberta avançada de aplicações Windows 