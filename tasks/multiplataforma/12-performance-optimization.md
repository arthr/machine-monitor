# Task 12: Otimização de Performance

## 📋 Objetivo
Otimizar o desempenho do agente multiplataforma, reduzindo uso de recursos, melhorando tempos de resposta e implementando estratégias de cache e paralelização eficientes.

## 🎯 Entregáveis
- [ ] Sistema de cache inteligente
- [ ] Paralelização de coletas
- [ ] Otimização de queries WMI/Registry
- [ ] Pool de conexões reutilizáveis
- [ ] Monitoramento de performance
- [ ] Benchmarks automatizados

## 📊 Contexto
Com a funcionalidade multiplataforma implementada, precisamos otimizar o desempenho para garantir que o agente seja eficiente em recursos e responsivo, especialmente em ambientes com hardware limitado ou alta carga.

## 🔧 Implementação

### 1. Sistema de Cache Inteligente

#### `internal/cache/smart_cache.go`
```go
package cache

import (
    "sync"
    "time"
    
    "machine-monitor/internal/collector"
)

// SmartCache implementa cache inteligente com TTL e invalidação
type SmartCache struct {
    mu          sync.RWMutex
    data        map[string]*CacheEntry
    cleanupTicker *time.Ticker
    stopCleanup chan bool
}

type CacheEntry struct {
    Value      interface{}
    ExpiresAt  time.Time
    AccessCount int64
    LastAccess time.Time
    Category   CacheCategory
}

type CacheCategory int

const (
    StaticData CacheCategory = iota // Dados que raramente mudam
    DynamicData                     // Dados que mudam frequentemente
    MetricsData                     // Métricas de sistema
)

func NewSmartCache() *SmartCache {
    cache := &SmartCache{
        data:        make(map[string]*CacheEntry),
        stopCleanup: make(chan bool),
    }
    
    // Iniciar limpeza automática
    cache.cleanupTicker = time.NewTicker(5 * time.Minute)
    go cache.cleanup()
    
    return cache
}

func (sc *SmartCache) Set(key string, value interface{}, category CacheCategory) {
    sc.mu.Lock()
    defer sc.mu.Unlock()
    
    ttl := sc.getTTLForCategory(category)
    
    sc.data[key] = &CacheEntry{
        Value:      value,
        ExpiresAt:  time.Now().Add(ttl),
        AccessCount: 0,
        LastAccess: time.Now(),
        Category:   category,
    }
}

func (sc *SmartCache) Get(key string) (interface{}, bool) {
    sc.mu.RLock()
    entry, exists := sc.data[key]
    sc.mu.RUnlock()
    
    if !exists {
        return nil, false
    }
    
    // Verificar expiração
    if time.Now().After(entry.ExpiresAt) {
        sc.Delete(key)
        return nil, false
    }
    
    // Atualizar estatísticas de acesso
    sc.mu.Lock()
    entry.AccessCount++
    entry.LastAccess = time.Now()
    sc.mu.Unlock()
    
    return entry.Value, true
}

func (sc *SmartCache) Delete(key string) {
    sc.mu.Lock()
    defer sc.mu.Unlock()
    delete(sc.data, key)
}

func (sc *SmartCache) getTTLForCategory(category CacheCategory) time.Duration {
    switch category {
    case StaticData:
        return 24 * time.Hour // Dados estáticos: 24 horas
    case DynamicData:
        return 5 * time.Minute // Dados dinâmicos: 5 minutos
    case MetricsData:
        return 30 * time.Second // Métricas: 30 segundos
    default:
        return 10 * time.Minute
    }
}

func (sc *SmartCache) cleanup() {
    for {
        select {
        case <-sc.cleanupTicker.C:
            sc.performCleanup()
        case <-sc.stopCleanup:
            return
        }
    }
}

func (sc *SmartCache) performCleanup() {
    sc.mu.Lock()
    defer sc.mu.Unlock()
    
    now := time.Now()
    
    for key, entry := range sc.data {
        // Remover entradas expiradas
        if now.After(entry.ExpiresAt) {
            delete(sc.data, key)
            continue
        }
        
        // Remover entradas pouco acessadas (LRU)
        if entry.AccessCount < 2 && now.Sub(entry.LastAccess) > time.Hour {
            delete(sc.data, key)
        }
    }
}

func (sc *SmartCache) GetStats() CacheStats {
    sc.mu.RLock()
    defer sc.mu.RUnlock()
    
    stats := CacheStats{
        TotalEntries: len(sc.data),
        Categories:   make(map[CacheCategory]int),
    }
    
    for _, entry := range sc.data {
        stats.Categories[entry.Category]++
        stats.TotalAccesses += entry.AccessCount
    }
    
    return stats
}

type CacheStats struct {
    TotalEntries  int
    TotalAccesses int64
    Categories    map[CacheCategory]int
}

func (sc *SmartCache) Stop() {
    sc.cleanupTicker.Stop()
    sc.stopCleanup <- true
}
```

#### `internal/cache/application_cache.go`
```go
package cache

import (
    "crypto/md5"
    "fmt"
    "time"
    
    "machine-monitor/internal/collector"
)

// ApplicationCache cache especializado para aplicações
type ApplicationCache struct {
    cache *SmartCache
}

func NewApplicationCache() *ApplicationCache {
    return &ApplicationCache{
        cache: NewSmartCache(),
    }
}

func (ac *ApplicationCache) GetApplications(platform string) ([]collector.Application, bool) {
    key := fmt.Sprintf("apps_%s", platform)
    
    if value, exists := ac.cache.Get(key); exists {
        if apps, ok := value.([]collector.Application); ok {
            return apps, true
        }
    }
    
    return nil, false
}

func (ac *ApplicationCache) SetApplications(platform string, apps []collector.Application) {
    key := fmt.Sprintf("apps_%s", platform)
    
    // Aplicações mudam raramente, usar cache de longa duração
    ac.cache.Set(key, apps, StaticData)
}

func (ac *ApplicationCache) GetApplicationMetadata(appPath string) (*collector.ApplicationMetadata, bool) {
    // Usar hash do caminho como chave
    hash := md5.Sum([]byte(appPath))
    key := fmt.Sprintf("metadata_%x", hash)
    
    if value, exists := ac.cache.Get(key); exists {
        if metadata, ok := value.(*collector.ApplicationMetadata); ok {
            return metadata, true
        }
    }
    
    return nil, false
}

func (ac *ApplicationCache) SetApplicationMetadata(appPath string, metadata *collector.ApplicationMetadata) {
    hash := md5.Sum([]byte(appPath))
    key := fmt.Sprintf("metadata_%x", hash)
    
    // Metadados de aplicação são estáticos
    ac.cache.Set(key, metadata, StaticData)
}

func (ac *ApplicationCache) InvalidateApplications(platform string) {
    key := fmt.Sprintf("apps_%s", platform)
    ac.cache.Delete(key)
}
```

### 2. Paralelização de Coletas

#### `internal/collector/parallel_collector.go`
```go
package collector

import (
    "context"
    "sync"
    "time"
    
    "golang.org/x/sync/errgroup"
)

// ParallelCollector executa coletas em paralelo para melhor performance
type ParallelCollector struct {
    platformCollector PlatformCollector
    cache            *cache.SmartCache
    logger           logging.Logger
    maxConcurrency   int
}

func NewParallelCollector(platformCollector PlatformCollector, logger logging.Logger) *ParallelCollector {
    return &ParallelCollector{
        platformCollector: platformCollector,
        cache:            cache.NewSmartCache(),
        logger:           logger,
        maxConcurrency:   4, // Número de goroutines paralelas
    }
}

func (pc *ParallelCollector) CollectAll(ctx context.Context) (*SystemData, error) {
    start := time.Now()
    
    // Usar errgroup para coletas paralelas
    g, ctx := errgroup.WithContext(ctx)
    
    var (
        systemInfo   *PlatformInfo
        applications []Application
        services     []Service
        metrics      *SystemMetrics
        machineID    string
    )
    
    // Coleta de informações do sistema
    g.Go(func() error {
        var err error
        systemInfo, err = pc.collectSystemInfoCached(ctx)
        return err
    })
    
    // Coleta de aplicações
    g.Go(func() error {
        var err error
        applications, err = pc.collectApplicationsCached(ctx)
        return err
    })
    
    // Coleta de serviços
    g.Go(func() error {
        var err error
        services, err = pc.collectServicesCached(ctx)
        return err
    })
    
    // Coleta de métricas
    g.Go(func() error {
        var err error
        metrics, err = pc.collectMetrics(ctx)
        return err
    })
    
    // Coleta de Machine ID
    g.Go(func() error {
        var err error
        machineID, err = pc.getMachineIDCached(ctx)
        return err
    })
    
    // Aguardar todas as coletas
    if err := g.Wait(); err != nil {
        return nil, err
    }
    
    duration := time.Since(start)
    pc.logger.Info("Coleta paralela concluída", "duration", duration)
    
    return &SystemData{
        MachineID:     machineID,
        Hostname:      systemInfo.Hostname,
        OS:            systemInfo.OS,
        OSVersion:     systemInfo.OSVersion,
        Architecture:  systemInfo.Architecture,
        Applications:  applications,
        Services:      services,
        SystemMetrics: metrics,
        Timestamp:     time.Now(),
    }, nil
}

func (pc *ParallelCollector) collectSystemInfoCached(ctx context.Context) (*PlatformInfo, error) {
    cacheKey := "system_info"
    
    // Tentar cache primeiro
    if cached, exists := pc.cache.Get(cacheKey); exists {
        if info, ok := cached.(*PlatformInfo); ok {
            return info, nil
        }
    }
    
    // Coletar dados frescos
    info, err := pc.platformCollector.CollectPlatformSpecific(ctx)
    if err != nil {
        return nil, err
    }
    
    // Armazenar no cache
    pc.cache.Set(cacheKey, info, cache.StaticData)
    
    return info, nil
}

func (pc *ParallelCollector) collectApplicationsCached(ctx context.Context) ([]Application, error) {
    cacheKey := "applications"
    
    // Tentar cache primeiro
    if cached, exists := pc.cache.Get(cacheKey); exists {
        if apps, ok := cached.([]Application); ok {
            return apps, nil
        }
    }
    
    // Coletar aplicações com paralelização interna
    apps, err := pc.collectApplicationsParallel(ctx)
    if err != nil {
        return nil, err
    }
    
    // Armazenar no cache
    pc.cache.Set(cacheKey, apps, cache.StaticData)
    
    return apps, nil
}

func (pc *ParallelCollector) collectApplicationsParallel(ctx context.Context) ([]Application, error) {
    // Dividir coleta de aplicações em chunks paralelos
    var allApps []Application
    var mu sync.Mutex
    
    g, ctx := errgroup.WithContext(ctx)
    
    // Registry apps
    g.Go(func() error {
        apps, err := pc.platformCollector.CollectInstalledApps(ctx)
        if err != nil {
            return err
        }
        
        mu.Lock()
        allApps = append(allApps, apps...)
        mu.Unlock()
        
        return nil
    })
    
    // Se for Windows, coletar UWP apps em paralelo
    if windowsCollector, ok := pc.platformCollector.(*WindowsCollector); ok {
        g.Go(func() error {
            apps, err := windowsCollector.getUWPApps()
            if err != nil {
                // UWP apps são opcionais, não falhar por isso
                pc.logger.Warn("Falha ao coletar UWP apps", "error", err)
                return nil
            }
            
            mu.Lock()
            allApps = append(allApps, apps...)
            mu.Unlock()
            
            return nil
        })
        
        // Aplicações portáveis
        g.Go(func() error {
            scanner := NewPortableAppScanner(pc.logger)
            apps, err := scanner.ScanPortableApps(ctx)
            if err != nil {
                pc.logger.Warn("Falha ao coletar apps portáveis", "error", err)
                return nil
            }
            
            mu.Lock()
            allApps = append(allApps, apps...)
            mu.Unlock()
            
            return nil
        })
    }
    
    if err := g.Wait(); err != nil {
        return nil, err
    }
    
    // Remover duplicatas
    return pc.removeDuplicateApps(allApps), nil
}

func (pc *ParallelCollector) collectServicesCached(ctx context.Context) ([]Service, error) {
    cacheKey := "services"
    
    // Tentar cache primeiro
    if cached, exists := pc.cache.Get(cacheKey); exists {
        if services, ok := cached.([]Service); ok {
            return services, nil
        }
    }
    
    // Coletar serviços
    services, err := pc.platformCollector.CollectSystemServices(ctx)
    if err != nil {
        return nil, err
    }
    
    // Serviços mudam com menos frequência
    pc.cache.Set(cacheKey, services, cache.DynamicData)
    
    return services, nil
}

func (pc *ParallelCollector) collectMetrics(ctx context.Context) (*SystemMetrics, error) {
    cacheKey := "metrics"
    
    // Métricas devem ser sempre frescas, mas verificar cache muito recente
    if cached, exists := pc.cache.Get(cacheKey); exists {
        if metrics, ok := cached.(*SystemMetrics); ok {
            return metrics, nil
        }
    }
    
    // Coletar métricas usando gopsutil
    metrics, err := pc.collectSystemMetrics(ctx)
    if err != nil {
        return nil, err
    }
    
    // Cache muito curto para métricas
    pc.cache.Set(cacheKey, metrics, cache.MetricsData)
    
    return metrics, nil
}

func (pc *ParallelCollector) getMachineIDCached(ctx context.Context) (string, error) {
    cacheKey := "machine_id"
    
    // Machine ID nunca muda, cache longo
    if cached, exists := pc.cache.Get(cacheKey); exists {
        if machineID, ok := cached.(string); ok {
            return machineID, nil
        }
    }
    
    // Gerar Machine ID
    machineID, err := pc.platformCollector.GetMachineID(ctx)
    if err != nil {
        return "", err
    }
    
    // Cache permanente para Machine ID
    pc.cache.Set(cacheKey, machineID, cache.StaticData)
    
    return machineID, nil
}
```

### 3. Otimização de Queries Windows

#### `internal/collector/optimized_wmi_windows.go`
```go
//go:build windows

package collector

import (
    "context"
    "fmt"
    "strings"
    "sync"
    "time"
    
    "github.com/go-ole/go-ole"
    "github.com/go-ole/go-ole/oleutil"
)

// OptimizedWMIClient cliente WMI otimizado com pool de conexões
type OptimizedWMIClient struct {
    connectionPool *WMIConnectionPool
    queryCache     *cache.SmartCache
    logger         logging.Logger
}

type WMIConnectionPool struct {
    connections chan *WMIConnection
    maxSize     int
    mu          sync.Mutex
    created     int
}

type WMIConnection struct {
    service    *ole.IDispatch
    lastUsed   time.Time
    inUse      bool
}

func NewOptimizedWMIClient(logger logging.Logger) *OptimizedWMIClient {
    return &OptimizedWMIClient{
        connectionPool: NewWMIConnectionPool(5), // Pool de 5 conexões
        queryCache:     cache.NewSmartCache(),
        logger:         logger,
    }
}

func NewWMIConnectionPool(maxSize int) *WMIConnectionPool {
    return &WMIConnectionPool{
        connections: make(chan *WMIConnection, maxSize),
        maxSize:     maxSize,
    }
}

func (pool *WMIConnectionPool) Get() (*WMIConnection, error) {
    select {
    case conn := <-pool.connections:
        conn.inUse = true
        conn.lastUsed = time.Now()
        return conn, nil
    default:
        // Criar nova conexão se pool não estiver cheio
        pool.mu.Lock()
        if pool.created < pool.maxSize {
            pool.created++
            pool.mu.Unlock()
            return pool.createConnection()
        }
        pool.mu.Unlock()
        
        // Aguardar conexão disponível
        conn := <-pool.connections
        conn.inUse = true
        conn.lastUsed = time.Now()
        return conn, nil
    }
}

func (pool *WMIConnectionPool) Put(conn *WMIConnection) {
    conn.inUse = false
    conn.lastUsed = time.Now()
    
    select {
    case pool.connections <- conn:
    default:
        // Pool cheio, fechar conexão
        conn.service.Release()
        pool.mu.Lock()
        pool.created--
        pool.mu.Unlock()
    }
}

func (pool *WMIConnectionPool) createConnection() (*WMIConnection, error) {
    ole.CoInitialize(0)
    
    unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
    if err != nil {
        return nil, err
    }
    
    wmi, err := unknown.QueryInterface(ole.IID_IDispatch)
    if err != nil {
        unknown.Release()
        return nil, err
    }
    unknown.Release()
    
    serviceRaw, err := oleutil.CallMethod(wmi, "ConnectServer")
    if err != nil {
        wmi.Release()
        return nil, err
    }
    
    service := serviceRaw.ToIDispatch()
    wmi.Release()
    
    return &WMIConnection{
        service:  service,
        lastUsed: time.Now(),
        inUse:    false,
    }, nil
}

func (client *OptimizedWMIClient) QueryWMI(ctx context.Context, query string) ([]map[string]interface{}, error) {
    // Verificar cache primeiro
    cacheKey := fmt.Sprintf("wmi_%x", md5.Sum([]byte(query)))
    
    if cached, exists := client.queryCache.Get(cacheKey); exists {
        if results, ok := cached.([]map[string]interface{}); ok {
            return results, nil
        }
    }
    
    // Executar query
    results, err := client.executeQuery(ctx, query)
    if err != nil {
        return nil, err
    }
    
    // Determinar tipo de cache baseado na query
    cacheCategory := client.determineCacheCategory(query)
    client.queryCache.Set(cacheKey, results, cacheCategory)
    
    return results, nil
}

func (client *OptimizedWMIClient) executeQuery(ctx context.Context, query string) ([]map[string]interface{}, error) {
    conn, err := client.connectionPool.Get()
    if err != nil {
        return nil, err
    }
    defer client.connectionPool.Put(conn)
    
    // Executar query com timeout
    resultChan := make(chan []map[string]interface{}, 1)
    errorChan := make(chan error, 1)
    
    go func() {
        results, err := client.doQuery(conn, query)
        if err != nil {
            errorChan <- err
        } else {
            resultChan <- results
        }
    }()
    
    select {
    case results := <-resultChan:
        return results, nil
    case err := <-errorChan:
        return nil, err
    case <-ctx.Done():
        return nil, ctx.Err()
    case <-time.After(30 * time.Second):
        return nil, fmt.Errorf("WMI query timeout: %s", query)
    }
}

func (client *OptimizedWMIClient) doQuery(conn *WMIConnection, query string) ([]map[string]interface{}, error) {
    resultRaw, err := oleutil.CallMethod(conn.service, "ExecQuery", query)
    if err != nil {
        return nil, err
    }
    
    result := resultRaw.ToIDispatch()
    defer result.Release()
    
    return client.parseResults(result)
}

func (client *OptimizedWMIClient) parseResults(result *ole.IDispatch) ([]map[string]interface{}, error) {
    var results []map[string]interface{}
    
    countRaw, err := oleutil.GetProperty(result, "Count")
    if err != nil {
        return nil, err
    }
    
    count := int(countRaw.Val)
    
    for i := 0; i < count; i++ {
        itemRaw, err := oleutil.CallMethod(result, "ItemIndex", i)
        if err != nil {
            continue
        }
        
        item := itemRaw.ToIDispatch()
        properties, err := client.getItemProperties(item)
        item.Release()
        
        if err != nil {
            continue
        }
        
        results = append(results, properties)
    }
    
    return results, nil
}

func (client *OptimizedWMIClient) getItemProperties(item *ole.IDispatch) (map[string]interface{}, error) {
    properties := make(map[string]interface{})
    
    // Obter lista de propriedades
    propsRaw, err := oleutil.GetProperty(item, "Properties_")
    if err != nil {
        return nil, err
    }
    
    props := propsRaw.ToIDispatch()
    defer props.Release()
    
    countRaw, err := oleutil.GetProperty(props, "Count")
    if err != nil {
        return nil, err
    }
    
    count := int(countRaw.Val)
    
    for i := 0; i < count; i++ {
        propRaw, err := oleutil.CallMethod(props, "ItemIndex", i)
        if err != nil {
            continue
        }
        
        prop := propRaw.ToIDispatch()
        
        nameRaw, err := oleutil.GetProperty(prop, "Name")
        if err != nil {
            prop.Release()
            continue
        }
        
        valueRaw, err := oleutil.GetProperty(prop, "Value")
        if err != nil {
            prop.Release()
            continue
        }
        
        name := nameRaw.ToString()
        value := valueRaw.Value()
        
        properties[name] = value
        
        prop.Release()
    }
    
    return properties, nil
}

func (client *OptimizedWMIClient) determineCacheCategory(query string) cache.CacheCategory {
    queryLower := strings.ToLower(query)
    
    // Queries de sistema são mais estáticas
    if strings.Contains(queryLower, "win32_computersystem") ||
       strings.Contains(queryLower, "win32_operatingsystem") ||
       strings.Contains(queryLower, "win32_bios") {
        return cache.StaticData
    }
    
    // Queries de processo/performance são dinâmicas
    if strings.Contains(queryLower, "win32_process") ||
       strings.Contains(queryLower, "win32_perfrawdata") {
        return cache.MetricsData
    }
    
    // Queries de serviço são semi-estáticas
    if strings.Contains(queryLower, "win32_service") {
        return cache.DynamicData
    }
    
    // Default para dados dinâmicos
    return cache.DynamicData
}

// Queries otimizadas pré-definidas
var OptimizedWMIQueries = map[string]string{
    "system_info": `SELECT Name, Version, BuildNumber, Architecture, TotalPhysicalMemory FROM Win32_OperatingSystem`,
    "computer_info": `SELECT Name, Manufacturer, Model, TotalPhysicalMemory FROM Win32_ComputerSystem`,
    "bios_info": `SELECT SerialNumber, Manufacturer, Version FROM Win32_BIOS`,
    "cpu_info": `SELECT Name, NumberOfCores, NumberOfLogicalProcessors, MaxClockSpeed FROM Win32_Processor`,
    "memory_info": `SELECT Capacity, Speed, DeviceLocator FROM Win32_PhysicalMemory`,
    "disk_info": `SELECT Size, FreeSpace, DriveType, FileSystem FROM Win32_LogicalDisk WHERE DriveType = 3`,
    "services": `SELECT Name, DisplayName, State, StartMode, PathName FROM Win32_Service`,
    "installed_programs": `SELECT Name, Version, Vendor, InstallDate FROM Win32_Product WHERE Name IS NOT NULL`,
}

func (client *OptimizedWMIClient) GetSystemInfo(ctx context.Context) (map[string]interface{}, error) {
    results, err := client.QueryWMI(ctx, OptimizedWMIQueries["system_info"])
    if err != nil {
        return nil, err
    }
    
    if len(results) > 0 {
        return results[0], nil
    }
    
    return nil, fmt.Errorf("no system info found")
}

func (client *OptimizedWMIClient) GetInstalledPrograms(ctx context.Context) ([]map[string]interface{}, error) {
    return client.QueryWMI(ctx, OptimizedWMIQueries["installed_programs"])
}

func (client *OptimizedWMIClient) GetServices(ctx context.Context) ([]map[string]interface{}, error) {
    return client.QueryWMI(ctx, OptimizedWMIQueries["services"])
}

func (client *OptimizedWMIClient) Close() {
    // Fechar todas as conexões do pool
    close(client.connectionPool.connections)
    
    for conn := range client.connectionPool.connections {
        if conn.service != nil {
            conn.service.Release()
        }
    }
    
    client.queryCache.Stop()
}
```

### 4. Monitoramento de Performance

#### `internal/monitoring/performance_monitor.go`
```go
package monitoring

import (
    "context"
    "runtime"
    "sync"
    "time"
)

// PerformanceMonitor monitora métricas de performance do agente
type PerformanceMonitor struct {
    mu                sync.RWMutex
    metrics           *PerformanceMetrics
    collectionTimes   []time.Duration
    memoryUsages      []uint64
    goroutineCount    []int
    startTime         time.Time
    logger            logging.Logger
}

type PerformanceMetrics struct {
    TotalCollections    int64
    SuccessfulCollections int64
    FailedCollections   int64
    AverageCollectionTime time.Duration
    PeakMemoryUsage     uint64
    CurrentMemoryUsage  uint64
    AverageMemoryUsage  uint64
    PeakGoroutines      int
    CurrentGoroutines   int
    Uptime             time.Duration
    CacheHitRate       float64
    CacheMissRate      float64
}

func NewPerformanceMonitor(logger logging.Logger) *PerformanceMonitor {
    return &PerformanceMonitor{
        metrics:         &PerformanceMetrics{},
        collectionTimes: make([]time.Duration, 0, 1000),
        memoryUsages:    make([]uint64, 0, 1000),
        goroutineCount:  make([]int, 0, 1000),
        startTime:       time.Now(),
        logger:          logger,
    }
}

func (pm *PerformanceMonitor) Start(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            pm.collectMetrics()
        }
    }
}

func (pm *PerformanceMonitor) collectMetrics() {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    // Coletar métricas de runtime
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    currentMemory := m.Alloc
    currentGoroutines := runtime.NumGoroutine()
    
    // Atualizar métricas
    pm.memoryUsages = append(pm.memoryUsages, currentMemory)
    pm.goroutineCount = append(pm.goroutineCount, currentGoroutines)
    
    // Manter apenas últimas 1000 medições
    if len(pm.memoryUsages) > 1000 {
        pm.memoryUsages = pm.memoryUsages[1:]
    }
    if len(pm.goroutineCount) > 1000 {
        pm.goroutineCount = pm.goroutineCount[1:]
    }
    
    // Calcular métricas agregadas
    pm.metrics.CurrentMemoryUsage = currentMemory
    pm.metrics.CurrentGoroutines = currentGoroutines
    pm.metrics.Uptime = time.Since(pm.startTime)
    
    // Calcular picos
    if currentMemory > pm.metrics.PeakMemoryUsage {
        pm.metrics.PeakMemoryUsage = currentMemory
    }
    if currentGoroutines > pm.metrics.PeakGoroutines {
        pm.metrics.PeakGoroutines = currentGoroutines
    }
    
    // Calcular médias
    pm.metrics.AverageMemoryUsage = pm.calculateAverageMemory()
    pm.metrics.AverageCollectionTime = pm.calculateAverageCollectionTime()
}

func (pm *PerformanceMonitor) RecordCollection(duration time.Duration, success bool) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    pm.metrics.TotalCollections++
    
    if success {
        pm.metrics.SuccessfulCollections++
    } else {
        pm.metrics.FailedCollections++
    }
    
    pm.collectionTimes = append(pm.collectionTimes, duration)
    
    // Manter apenas últimas 1000 medições
    if len(pm.collectionTimes) > 1000 {
        pm.collectionTimes = pm.collectionTimes[1:]
    }
}

func (pm *PerformanceMonitor) RecordCacheHit() {
    // Implementar contadores de cache
}

func (pm *PerformanceMonitor) RecordCacheMiss() {
    // Implementar contadores de cache
}

func (pm *PerformanceMonitor) GetMetrics() *PerformanceMetrics {
    pm.mu.RLock()
    defer pm.mu.RUnlock()
    
    // Retornar cópia das métricas
    metrics := *pm.metrics
    return &metrics
}

func (pm *PerformanceMonitor) calculateAverageMemory() uint64 {
    if len(pm.memoryUsages) == 0 {
        return 0
    }
    
    var total uint64
    for _, usage := range pm.memoryUsages {
        total += usage
    }
    
    return total / uint64(len(pm.memoryUsages))
}

func (pm *PerformanceMonitor) calculateAverageCollectionTime() time.Duration {
    if len(pm.collectionTimes) == 0 {
        return 0
    }
    
    var total time.Duration
    for _, duration := range pm.collectionTimes {
        total += duration
    }
    
    return total / time.Duration(len(pm.collectionTimes))
}

func (pm *PerformanceMonitor) GetHealthStatus() HealthStatus {
    metrics := pm.GetMetrics()
    
    status := HealthStatus{
        Overall: "healthy",
        Checks:  make(map[string]string),
    }
    
    // Verificar uso de memória
    if metrics.CurrentMemoryUsage > 200*1024*1024 { // 200MB
        status.Checks["memory"] = "warning"
        if metrics.CurrentMemoryUsage > 500*1024*1024 { // 500MB
            status.Checks["memory"] = "critical"
            status.Overall = "unhealthy"
        }
    } else {
        status.Checks["memory"] = "healthy"
    }
    
    // Verificar número de goroutines
    if metrics.CurrentGoroutines > 100 {
        status.Checks["goroutines"] = "warning"
        if metrics.CurrentGoroutines > 500 {
            status.Checks["goroutines"] = "critical"
            status.Overall = "unhealthy"
        }
    } else {
        status.Checks["goroutines"] = "healthy"
    }
    
    // Verificar taxa de sucesso
    if metrics.TotalCollections > 0 {
        successRate := float64(metrics.SuccessfulCollections) / float64(metrics.TotalCollections)
        if successRate < 0.8 {
            status.Checks["success_rate"] = "warning"
            if successRate < 0.5 {
                status.Checks["success_rate"] = "critical"
                status.Overall = "unhealthy"
            }
        } else {
            status.Checks["success_rate"] = "healthy"
        }
    }
    
    return status
}

type HealthStatus struct {
    Overall string            `json:"overall"`
    Checks  map[string]string `json:"checks"`
}
```

### 5. Benchmarks Automatizados

#### `internal/collector/benchmark_test.go`
```go
package collector

import (
    "context"
    "testing"
    "time"
)

func BenchmarkParallelCollection(b *testing.B) {
    logger := logging.NewLogger(logging.Config{Level: "error"})
    
    // Criar collector para plataforma atual
    platformCollector := createCollectorForCurrentPlatform(logger)
    parallelCollector := NewParallelCollector(platformCollector, logger)
    
    ctx := context.Background()
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, err := parallelCollector.CollectAll(ctx)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkSequentialCollection(b *testing.B) {
    logger := logging.NewLogger(logging.Config{Level: "error"})
    platformCollector := createCollectorForCurrentPlatform(logger)
    
    ctx := context.Background()
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        // Coleta sequencial
        _, err := platformCollector.CollectPlatformSpecific(ctx)
        if err != nil {
            b.Fatal(err)
        }
        
        _, err = platformCollector.CollectInstalledApps(ctx)
        if err != nil {
            b.Fatal(err)
        }
        
        _, err = platformCollector.CollectSystemServices(ctx)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkCachePerformance(b *testing.B) {
    cache := cache.NewSmartCache()
    defer cache.Stop()
    
    testData := []collector.Application{
        {Name: "Test App 1", Version: "1.0"},
        {Name: "Test App 2", Version: "2.0"},
    }
    
    b.ResetTimer()
    
    b.Run("Cache_Set", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            cache.Set(fmt.Sprintf("key_%d", i), testData, cache.StaticData)
        }
    })
    
    b.Run("Cache_Get", func(b *testing.B) {
        // Pré-popular cache
        for i := 0; i < 1000; i++ {
            cache.Set(fmt.Sprintf("key_%d", i), testData, cache.StaticData)
        }
        
        b.ResetTimer()
        
        for i := 0; i < b.N; i++ {
            cache.Get(fmt.Sprintf("key_%d", i%1000))
        }
    })
}

func BenchmarkWMIOptimized(b *testing.B) {
    if runtime.GOOS != "windows" {
        b.Skip("WMI benchmark apenas no Windows")
    }
    
    logger := logging.NewLogger(logging.Config{Level: "error"})
    client := NewOptimizedWMIClient(logger)
    defer client.Close()
    
    ctx := context.Background()
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, err := client.GetSystemInfo(ctx)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkMemoryUsage(b *testing.B) {
    logger := logging.NewLogger(logging.Config{Level: "error"})
    platformCollector := createCollectorForCurrentPlatform(logger)
    parallelCollector := NewParallelCollector(platformCollector, logger)
    
    ctx := context.Background()
    
    // Medir uso de memória
    var m1, m2 runtime.MemStats
    runtime.ReadMemStats(&m1)
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        data, err := parallelCollector.CollectAll(ctx)
        if err != nil {
            b.Fatal(err)
        }
        
        // Evitar otimização do compilador
        _ = data
    }
    
    runtime.ReadMemStats(&m2)
    
    b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
    b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "total-bytes/op")
}
```

## ✅ Critérios de Sucesso

### Performance
- [ ] Coleta completa em < 15 segundos (melhoria de 50%)
- [ ] Uso de memória < 50MB em operação normal
- [ ] Cache hit rate > 80% para dados estáticos
- [ ] Paralelização reduz tempo de coleta em 40%

### Eficiência
- [ ] Reutilização de conexões WMI funcionando
- [ ] Cache inteligente com TTL apropriado
- [ ] Cleanup automático de recursos
- [ ] Monitoramento de performance ativo

### Qualidade
- [ ] Benchmarks automatizados passando
- [ ] Métricas de health check implementadas
- [ ] Detecção de vazamentos de memória
- [ ] Alertas para degradação de performance

## 🧪 Testes e Benchmarks

### Execução Local
```bash
# Benchmarks de performance
go test -bench=. -benchmem ./internal/collector/...

# Testes de carga
go test -v ./internal/collector/... -run=TestLoad

# Profile de memória
go test -bench=BenchmarkMemoryUsage -memprofile=mem.prof ./internal/collector/...
go tool pprof mem.prof
```

### Monitoramento Contínuo
```bash
# Executar com profiling
go run -cpuprofile=cpu.prof -memprofile=mem.prof ./cmd/agente/

# Analisar profiles
go tool pprof cpu.prof
go tool pprof mem.prof
```

## 📚 Referências

### Performance em Go
- [Go Performance Tips](https://golang.org/doc/effective_go.html#performance)
- [Memory Profiling](https://golang.org/blog/pprof)
- [Benchmarking](https://golang.org/pkg/testing/#hdr-Benchmarks)

### Otimização
- [Go Memory Model](https://golang.org/ref/mem)
- [Sync Package](https://golang.org/pkg/sync/)
- [Context Package](https://golang.org/pkg/context/)

## 🔄 Próximos Passos
Após completar esta task, prosseguir para:
- **Task 13**: Documentação final e entrega 