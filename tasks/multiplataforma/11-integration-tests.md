# Task 11: Testes de Integração Completos

## 📋 Objetivo
Implementar testes de integração abrangentes que validem o funcionamento completo do sistema multiplataforma, incluindo fluxos end-to-end, integração entre componentes e cenários reais de uso.

## 🎯 Entregáveis
- [ ] Testes de integração end-to-end
- [ ] Testes de compatibilidade entre componentes
- [ ] Cenários de teste realistas
- [ ] Testes de carga e stress
- [ ] Validação de dados em ambiente real
- [ ] Relatórios de qualidade automatizados

## 📊 Contexto
Após implementar testes específicos de plataforma, precisamos validar que todo o sistema funciona corretamente quando integrado, simulando cenários reais de uso e garantindo que a comunicação entre componentes seja robusta.

## 🔧 Implementação

### 1. Framework de Testes de Integração

#### `internal/integration/test_framework.go`
```go
package integration

import (
    "context"
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
    
    "github.com/gorilla/websocket"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "machine-monitor/internal/agent"
    "machine-monitor/internal/collector"
    "machine-monitor/internal/comms"
    "machine-monitor/internal/executor"
    "machine-monitor/internal/logging"
)

// IntegrationTestSuite coordena testes de integração completos
type IntegrationTestSuite struct {
    t           *testing.T
    logger      logging.Logger
    agent       *agent.Agent
    mockServer  *httptest.Server
    wsUpgrader  websocket.Upgrader
    testConfig  *TestConfig
}

type TestConfig struct {
    CollectionInterval time.Duration
    MaxTestDuration    time.Duration
    ExpectedApps       int
    ExpectedServices   int
    ServerURL          string
}

func NewIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
    logger := logging.NewLogger(logging.Config{
        Level:  "debug",
        Format: "json",
    })
    
    config := &TestConfig{
        CollectionInterval: 5 * time.Second,
        MaxTestDuration:    2 * time.Minute,
        ExpectedApps:       10,
        ExpectedServices:   5,
    }
    
    return &IntegrationTestSuite{
        t:          t,
        logger:     logger,
        testConfig: config,
        wsUpgrader: websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool { return true },
        },
    }
}

func (its *IntegrationTestSuite) SetupMockServer() {
    mux := http.NewServeMux()
    
    // Endpoint para receber dados do agente
    mux.HandleFunc("/agent/data", its.handleAgentData)
    
    // Endpoint WebSocket para comandos
    mux.HandleFunc("/agent/commands", its.handleWebSocketCommands)
    
    // Endpoint de health check
    mux.HandleFunc("/health", its.handleHealthCheck)
    
    its.mockServer = httptest.NewServer(mux)
    its.testConfig.ServerURL = its.mockServer.URL
}

func (its *IntegrationTestSuite) TearDown() {
    if its.mockServer != nil {
        its.mockServer.Close()
    }
    
    if its.agent != nil {
        its.agent.Stop()
    }
}

func (its *IntegrationTestSuite) RunFullIntegrationTest() {
    its.t.Run("Setup", its.testSetup)
    its.t.Run("AgentInitialization", its.testAgentInitialization)
    its.t.Run("DataCollection", its.testDataCollection)
    its.t.Run("Communication", its.testCommunication)
    its.t.Run("CommandExecution", its.testCommandExecution)
    its.t.Run("ErrorHandling", its.testErrorHandling)
    its.t.Run("Performance", its.testPerformance)
    its.t.Run("Cleanup", its.testCleanup)
}

func (its *IntegrationTestSuite) testSetup() {
    its.SetupMockServer()
    
    // Configurar agente
    agentConfig := &agent.Config{
        CollectionInterval: its.testConfig.CollectionInterval,
        ServerURL:          its.testConfig.ServerURL,
        WebSocketURL:       its.testConfig.ServerURL + "/agent/commands",
        MaxRetries:         3,
        RetryDelay:         time.Second,
    }
    
    var err error
    its.agent, err = agent.NewAgent(agentConfig, its.logger)
    require.NoError(its.t, err)
    assert.NotNil(its.t, its.agent)
}

func (its *IntegrationTestSuite) testAgentInitialization() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Inicializar agente
    err := its.agent.Initialize(ctx)
    require.NoError(its.t, err)
    
    // Verificar se componentes foram inicializados
    assert.True(its.t, its.agent.IsInitialized())
    
    // Verificar conectividade
    assert.True(its.t, its.agent.IsConnected())
}

func (its *IntegrationTestSuite) testDataCollection() {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()
    
    // Canal para receber dados coletados
    dataChan := make(chan *collector.SystemData, 1)
    
    // Configurar handler para capturar dados
    its.setupDataCapture(dataChan)
    
    // Iniciar coleta
    go its.agent.Start(ctx)
    
    // Aguardar primeira coleta
    select {
    case data := <-dataChan:
        its.validateCollectedData(data)
    case <-time.After(30 * time.Second):
        its.t.Fatal("Timeout aguardando coleta de dados")
    }
}

func (its *IntegrationTestSuite) validateCollectedData(data *collector.SystemData) {
    require.NotNil(its.t, data)
    
    // Validar informações básicas do sistema
    assert.NotEmpty(its.t, data.MachineID)
    assert.NotEmpty(its.t, data.Hostname)
    assert.NotEmpty(its.t, data.OS)
    assert.NotEmpty(its.t, data.OSVersion)
    assert.NotEmpty(its.t, data.Architecture)
    
    // Validar aplicações coletadas
    assert.NotEmpty(its.t, data.Applications)
    assert.GreaterOrEqual(its.t, len(data.Applications), its.testConfig.ExpectedApps)
    
    // Validar serviços coletados
    assert.NotEmpty(its.t, data.Services)
    assert.GreaterOrEqual(its.t, len(data.Services), its.testConfig.ExpectedServices)
    
    // Validar métricas de sistema
    assert.NotNil(its.t, data.SystemMetrics)
    assert.Greater(its.t, data.SystemMetrics.CPUUsage, 0.0)
    assert.Greater(its.t, data.SystemMetrics.MemoryUsage, 0.0)
    
    // Validar timestamp
    assert.WithinDuration(its.t, time.Now(), data.Timestamp, time.Minute)
}

func (its *IntegrationTestSuite) testCommunication() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Teste de envio de dados via HTTP
    its.t.Run("HTTP_Communication", func(t *testing.T) {
        its.testHTTPCommunication(ctx)
    })
    
    // Teste de comunicação WebSocket
    its.t.Run("WebSocket_Communication", func(t *testing.T) {
        its.testWebSocketCommunication(ctx)
    })
}

func (its *IntegrationTestSuite) testHTTPCommunication(ctx context.Context) {
    // Simular envio de dados
    testData := &collector.SystemData{
        MachineID:   "test-machine-id",
        Hostname:    "test-hostname",
        OS:          "test-os",
        OSVersion:   "test-version",
        Timestamp:   time.Now(),
    }
    
    // Configurar captura de dados no servidor
    receivedData := make(chan *collector.SystemData, 1)
    its.setupHTTPDataCapture(receivedData)
    
    // Enviar dados
    err := its.agent.SendData(ctx, testData)
    require.NoError(its.t, err)
    
    // Verificar recebimento
    select {
    case received := <-receivedData:
        assert.Equal(its.t, testData.MachineID, received.MachineID)
        assert.Equal(its.t, testData.Hostname, received.Hostname)
    case <-time.After(10 * time.Second):
        its.t.Fatal("Timeout aguardando dados via HTTP")
    }
}

func (its *IntegrationTestSuite) testWebSocketCommunication(ctx context.Context) {
    // Conectar WebSocket
    conn, err := its.agent.ConnectWebSocket(ctx)
    require.NoError(its.t, err)
    defer conn.Close()
    
    // Enviar comando de teste
    testCommand := &executor.Command{
        ID:      "test-command-1",
        Type:    "system_info",
        Command: "whoami",
        Args:    []string{},
    }
    
    err = conn.WriteJSON(testCommand)
    require.NoError(its.t, err)
    
    // Aguardar resposta
    var response executor.CommandResult
    err = conn.ReadJSON(&response)
    require.NoError(its.t, err)
    
    assert.Equal(its.t, testCommand.ID, response.CommandID)
    assert.Equal(its.t, "success", response.Status)
    assert.NotEmpty(its.t, response.Output)
}

func (its *IntegrationTestSuite) testCommandExecution() {
    ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
    defer cancel()
    
    // Testes de comandos específicos por plataforma
    its.testPlatformSpecificCommands(ctx)
    
    // Testes de validação de segurança
    its.testSecurityValidation(ctx)
    
    // Testes de timeout e limites
    its.testCommandLimits(ctx)
}

func (its *IntegrationTestSuite) testPlatformSpecificCommands(ctx context.Context) {
    commands := its.getPlatformCommands()
    
    for _, cmd := range commands {
        its.t.Run(fmt.Sprintf("Command_%s", cmd.Command), func(t *testing.T) {
            result, err := its.agent.ExecuteCommand(ctx, cmd)
            require.NoError(t, err)
            
            assert.Equal(t, cmd.ID, result.CommandID)
            assert.Equal(t, "success", result.Status)
            assert.NotEmpty(t, result.Output)
            assert.Greater(t, result.ExecutionTime, time.Duration(0))
        })
    }
}

func (its *IntegrationTestSuite) testSecurityValidation(ctx context.Context) {
    // Comandos que devem ser rejeitados
    dangerousCommands := []executor.Command{
        {ID: "dangerous-1", Command: "rm", Args: []string{"-rf", "/"}},
        {ID: "dangerous-2", Command: "format", Args: []string{"C:"}},
        {ID: "dangerous-3", Command: "shutdown", Args: []string{"-h", "now"}},
    }
    
    for _, cmd := range dangerousCommands {
        its.t.Run(fmt.Sprintf("Security_%s", cmd.Command), func(t *testing.T) {
            result, err := its.agent.ExecuteCommand(ctx, &cmd)
            
            // Deve retornar erro ou status de falha
            if err == nil {
                assert.Equal(t, "error", result.Status)
                assert.Contains(t, result.Error, "not allowed")
            } else {
                assert.Error(t, err)
            }
        })
    }
}

func (its *IntegrationTestSuite) testCommandLimits(ctx context.Context) {
    // Comando com timeout longo
    longCommand := &executor.Command{
        ID:      "long-command",
        Command: "ping",
        Args:    []string{"-c", "100", "127.0.0.1"},
        Timeout: 5 * time.Second,
    }
    
    start := time.Now()
    result, err := its.agent.ExecuteCommand(ctx, longCommand)
    duration := time.Since(start)
    
    // Deve respeitar timeout
    assert.Less(its.t, duration, 10*time.Second)
    
    if err != nil {
        assert.Contains(its.t, err.Error(), "timeout")
    } else {
        assert.Equal(its.t, "timeout", result.Status)
    }
}

func (its *IntegrationTestSuite) testErrorHandling() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Teste de reconexão automática
    its.t.Run("Auto_Reconnection", func(t *testing.T) {
        its.testAutoReconnection(ctx)
    })
    
    // Teste de recuperação de erros
    its.t.Run("Error_Recovery", func(t *testing.T) {
        its.testErrorRecovery(ctx)
    })
}

func (its *IntegrationTestSuite) testAutoReconnection(ctx context.Context) {
    // Simular desconexão do servidor
    its.mockServer.Close()
    
    // Aguardar tentativa de reconexão
    time.Sleep(2 * time.Second)
    
    // Recriar servidor
    its.SetupMockServer()
    
    // Verificar se reconectou
    assert.Eventually(its.t, func() bool {
        return its.agent.IsConnected()
    }, 15*time.Second, time.Second)
}

func (its *IntegrationTestSuite) testErrorRecovery(ctx context.Context) {
    // Simular erro de coleta
    its.agent.SimulateCollectionError()
    
    // Verificar se continua funcionando após erro
    time.Sleep(its.testConfig.CollectionInterval * 2)
    
    assert.True(its.t, its.agent.IsRunning())
}

func (its *IntegrationTestSuite) testPerformance() {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    
    // Métricas de performance
    metrics := &PerformanceMetrics{
        CollectionTimes: make([]time.Duration, 0),
        MemoryUsage:     make([]uint64, 0),
        CPUUsage:        make([]float64, 0),
    }
    
    // Monitorar performance durante execução
    go its.monitorPerformance(ctx, metrics)
    
    // Executar agente por período determinado
    its.agent.Start(ctx)
    
    // Validar métricas
    its.validatePerformanceMetrics(metrics)
}

type PerformanceMetrics struct {
    CollectionTimes []time.Duration
    MemoryUsage     []uint64
    CPUUsage        []float64
}

func (its *IntegrationTestSuite) monitorPerformance(ctx context.Context, metrics *PerformanceMetrics) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Coletar métricas de performance
            memUsage := its.getMemoryUsage()
            cpuUsage := its.getCPUUsage()
            
            metrics.MemoryUsage = append(metrics.MemoryUsage, memUsage)
            metrics.CPUUsage = append(metrics.CPUUsage, cpuUsage)
        }
    }
}

func (its *IntegrationTestSuite) validatePerformanceMetrics(metrics *PerformanceMetrics) {
    // Validar uso de memória
    avgMemory := its.calculateAverage(metrics.MemoryUsage)
    assert.Less(its.t, avgMemory, uint64(100*1024*1024), "Uso médio de memória deve ser < 100MB")
    
    // Validar uso de CPU
    avgCPU := its.calculateAverageFloat(metrics.CPUUsage)
    assert.Less(its.t, avgCPU, 10.0, "Uso médio de CPU deve ser < 10%")
    
    // Validar tempos de coleta
    if len(metrics.CollectionTimes) > 0 {
        avgCollectionTime := its.calculateAverageDuration(metrics.CollectionTimes)
        assert.Less(its.t, avgCollectionTime, 30*time.Second, "Tempo médio de coleta deve ser < 30s")
    }
}

// Handlers do servidor mock
func (its *IntegrationTestSuite) handleAgentData(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    
    var data collector.SystemData
    if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    // Enviar dados para canal de teste se configurado
    if its.dataCaptureChan != nil {
        select {
        case its.dataCaptureChan <- &data:
        default:
        }
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

func (its *IntegrationTestSuite) handleWebSocketCommands(w http.ResponseWriter, r *http.Request) {
    conn, err := its.wsUpgrader.Upgrade(w, r, nil)
    if err != nil {
        its.logger.Error("WebSocket upgrade failed", "error", err)
        return
    }
    defer conn.Close()
    
    for {
        var cmd executor.Command
        if err := conn.ReadJSON(&cmd); err != nil {
            break
        }
        
        // Simular processamento de comando
        result := executor.CommandResult{
            CommandID:     cmd.ID,
            Status:        "success",
            Output:        "Mock command executed successfully",
            ExecutionTime: 100 * time.Millisecond,
        }
        
        if err := conn.WriteJSON(result); err != nil {
            break
        }
    }
}

func (its *IntegrationTestSuite) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// Métodos auxiliares
func (its *IntegrationTestSuite) getPlatformCommands() []executor.Command {
    switch runtime.GOOS {
    case "windows":
        return []executor.Command{
            {ID: "cmd-1", Command: "whoami"},
            {ID: "cmd-2", Command: "systeminfo"},
            {ID: "cmd-3", Command: "tasklist"},
        }
    case "darwin":
        return []executor.Command{
            {ID: "cmd-1", Command: "whoami"},
            {ID: "cmd-2", Command: "system_profiler", Args: []string{"SPSoftwareDataType"}},
            {ID: "cmd-3", Command: "ps", Args: []string{"aux"}},
        }
    default:
        return []executor.Command{
            {ID: "cmd-1", Command: "whoami"},
            {ID: "cmd-2", Command: "uname", Args: []string{"-a"}},
            {ID: "cmd-3", Command: "ps", Args: []string{"aux"}},
        }
    }
}
```

### 2. Testes de Carga e Stress

#### `internal/integration/load_test.go`
```go
package integration

import (
    "context"
    "sync"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLoadAndStress(t *testing.T) {
    suite := NewIntegrationTestSuite(t)
    defer suite.TearDown()
    
    suite.SetupMockServer()
    
    t.Run("ConcurrentCollections", func(t *testing.T) {
        testConcurrentCollections(t, suite)
    })
    
    t.Run("HighFrequencyCollections", func(t *testing.T) {
        testHighFrequencyCollections(t, suite)
    })
    
    t.Run("LongRunningStability", func(t *testing.T) {
        testLongRunningStability(t, suite)
    })
    
    t.Run("MemoryLeakDetection", func(t *testing.T) {
        testMemoryLeakDetection(t, suite)
    })
}

func testConcurrentCollections(t *testing.T, suite *IntegrationTestSuite) {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    
    const numGoroutines = 10
    const collectionsPerGoroutine = 5
    
    var wg sync.WaitGroup
    errors := make(chan error, numGoroutines*collectionsPerGoroutine)
    
    // Executar coletas concorrentes
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(goroutineID int) {
            defer wg.Done()
            
            for j := 0; j < collectionsPerGoroutine; j++ {
                data, err := suite.agent.CollectData(ctx)
                if err != nil {
                    errors <- err
                    return
                }
                
                // Validar dados básicos
                if data.MachineID == "" || data.Hostname == "" {
                    errors <- fmt.Errorf("dados incompletos na goroutine %d, coleta %d", goroutineID, j)
                    return
                }
            }
        }(i)
    }
    
    // Aguardar conclusão
    wg.Wait()
    close(errors)
    
    // Verificar erros
    var errorCount int
    for err := range errors {
        t.Logf("Erro na coleta concorrente: %v", err)
        errorCount++
    }
    
    assert.Equal(t, 0, errorCount, "Não deve haver erros em coletas concorrentes")
}

func testHighFrequencyCollections(t *testing.T, suite *IntegrationTestSuite) {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
    defer cancel()
    
    // Configurar coleta a cada 1 segundo
    suite.agent.SetCollectionInterval(1 * time.Second)
    
    collectionCount := 0
    dataChan := make(chan *collector.SystemData, 100)
    
    // Capturar dados coletados
    suite.setupDataCapture(dataChan)
    
    // Iniciar agente
    go suite.agent.Start(ctx)
    
    // Contar coletas por 30 segundos
    timeout := time.After(30 * time.Second)
    
    for {
        select {
        case <-dataChan:
            collectionCount++
        case <-timeout:
            goto done
        case <-ctx.Done():
            goto done
        }
    }
    
done:
    // Deve ter pelo menos 25 coletas em 30 segundos (permitindo alguma margem)
    assert.GreaterOrEqual(t, collectionCount, 25, 
        "Deve ter pelo menos 25 coletas em 30 segundos, teve %d", collectionCount)
}

func testLongRunningStability(t *testing.T, suite *IntegrationTestSuite) {
    if testing.Short() {
        t.Skip("Pulando teste de longa duração em modo short")
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()
    
    // Configurar coleta a cada 10 segundos
    suite.agent.SetCollectionInterval(10 * time.Second)
    
    var (
        collectionCount int
        errorCount      int
        mu             sync.Mutex
    )
    
    // Monitorar coletas
    dataChan := make(chan *collector.SystemData, 100)
    errorChan := make(chan error, 100)
    
    suite.setupDataCapture(dataChan)
    suite.setupErrorCapture(errorChan)
    
    // Iniciar agente
    go suite.agent.Start(ctx)
    
    // Monitorar por 10 minutos
    done := make(chan bool)
    go func() {
        for {
            select {
            case <-dataChan:
                mu.Lock()
                collectionCount++
                mu.Unlock()
            case <-errorChan:
                mu.Lock()
                errorCount++
                mu.Unlock()
            case <-ctx.Done():
                done <- true
                return
            }
        }
    }()
    
    <-done
    
    // Validar estabilidade
    assert.GreaterOrEqual(t, collectionCount, 50, "Deve ter pelo menos 50 coletas")
    assert.LessOrEqual(t, errorCount, collectionCount/10, "Taxa de erro deve ser < 10%")
}

func testMemoryLeakDetection(t *testing.T, suite *IntegrationTestSuite) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    // Configurar coleta frequente
    suite.agent.SetCollectionInterval(2 * time.Second)
    
    var memoryUsages []uint64
    
    // Medir uso de memória inicial
    initialMemory := suite.getMemoryUsage()
    memoryUsages = append(memoryUsages, initialMemory)
    
    // Executar por 5 minutos, medindo memória a cada 30 segundos
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    go suite.agent.Start(ctx)
    
    for {
        select {
        case <-ticker.C:
            currentMemory := suite.getMemoryUsage()
            memoryUsages = append(memoryUsages, currentMemory)
            
            t.Logf("Uso de memória: %d MB", currentMemory/(1024*1024))
            
        case <-ctx.Done():
            goto analysis
        }
    }
    
analysis:
    // Analisar tendência de memória
    if len(memoryUsages) < 3 {
        t.Skip("Dados insuficientes para análise de vazamento")
    }
    
    // Verificar se há crescimento constante (possível vazamento)
    finalMemory := memoryUsages[len(memoryUsages)-1]
    memoryGrowth := finalMemory - initialMemory
    
    // Permitir crescimento até 50MB
    maxAllowedGrowth := uint64(50 * 1024 * 1024)
    
    assert.LessOrEqual(t, memoryGrowth, maxAllowedGrowth,
        "Crescimento de memória (%d MB) indica possível vazamento", memoryGrowth/(1024*1024))
}
```

### 3. Testes de Cenários Reais

#### `internal/integration/scenarios_test.go`
```go
package integration

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestRealWorldScenarios(t *testing.T) {
    suite := NewIntegrationTestSuite(t)
    defer suite.TearDown()
    
    suite.SetupMockServer()
    
    t.Run("TypicalWorkstation", func(t *testing.T) {
        testTypicalWorkstation(t, suite)
    })
    
    t.Run("ServerEnvironment", func(t *testing.T) {
        testServerEnvironment(t, suite)
    })
    
    t.Run("NetworkInterruption", func(t *testing.T) {
        testNetworkInterruption(t, suite)
    })
    
    t.Run("SystemUnderLoad", func(t *testing.T) {
        testSystemUnderLoad(t, suite)
    })
}

func testTypicalWorkstation(t *testing.T, suite *IntegrationTestSuite) {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    
    // Simular ambiente típico de estação de trabalho
    suite.agent.SetCollectionInterval(30 * time.Second)
    
    collectedData := make([]*collector.SystemData, 0)
    dataChan := make(chan *collector.SystemData, 10)
    
    suite.setupDataCapture(dataChan)
    
    go suite.agent.Start(ctx)
    
    // Coletar dados por 2 minutos
    timeout := time.After(2 * time.Minute)
    
    for {
        select {
        case data := <-dataChan:
            collectedData = append(collectedData, data)
            
            // Validar dados de estação de trabalho
            assert.NotEmpty(t, data.Applications, "Estação deve ter aplicações")
            assert.Greater(t, len(data.Applications), 10, "Estação típica tem >10 aplicações")
            
            // Verificar aplicações comuns
            suite.verifyCommonApplications(t, data.Applications)
            
        case <-timeout:
            goto validation
        }
    }
    
validation:
    assert.GreaterOrEqual(t, len(collectedData), 3, "Deve ter pelo menos 3 coletas")
    
    // Validar consistência dos dados
    for i := 1; i < len(collectedData); i++ {
        assert.Equal(t, collectedData[0].MachineID, collectedData[i].MachineID)
        assert.Equal(t, collectedData[0].Hostname, collectedData[i].Hostname)
    }
}

func testServerEnvironment(t *testing.T, suite *IntegrationTestSuite) {
    ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
    defer cancel()
    
    // Simular ambiente de servidor
    suite.agent.SetCollectionInterval(60 * time.Second)
    
    var serverData *collector.SystemData
    dataChan := make(chan *collector.SystemData, 1)
    
    suite.setupDataCapture(dataChan)
    
    go suite.agent.Start(ctx)
    
    // Aguardar primeira coleta
    select {
    case data := <-dataChan:
        serverData = data
    case <-time.After(75 * time.Second):
        t.Fatal("Timeout aguardando coleta em ambiente servidor")
    }
    
    // Validar características de servidor
    assert.NotNil(t, serverData)
    assert.NotEmpty(t, serverData.Services, "Servidor deve ter serviços")
    assert.Greater(t, len(serverData.Services), 20, "Servidor típico tem >20 serviços")
    
    // Verificar serviços críticos
    suite.verifyServerServices(t, serverData.Services)
}

func testNetworkInterruption(t *testing.T, suite *IntegrationTestSuite) {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
    defer cancel()
    
    suite.agent.SetCollectionInterval(10 * time.Second)
    
    connectionStatus := make(chan bool, 10)
    
    // Monitorar status de conexão
    go func() {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                connectionStatus <- suite.agent.IsConnected()
            case <-ctx.Done():
                return
            }
        }
    }()
    
    go suite.agent.Start(ctx)
    
    // Aguardar conexão inicial
    time.Sleep(15 * time.Second)
    
    // Simular interrupção de rede
    suite.mockServer.Close()
    
    // Aguardar detecção de desconexão
    time.Sleep(10 * time.Second)
    
    // Restaurar conexão
    suite.SetupMockServer()
    
    // Verificar reconexão
    reconnected := false
    timeout := time.After(60 * time.Second)
    
    for {
        select {
        case connected := <-connectionStatus:
            if connected {
                reconnected = true
                goto validation
            }
        case <-timeout:
            goto validation
        }
    }
    
validation:
    assert.True(t, reconnected, "Agente deve reconectar após interrupção de rede")
}

func testSystemUnderLoad(t *testing.T, suite *IntegrationTestSuite) {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    
    // Simular carga do sistema executando múltiplas coletas
    suite.agent.SetCollectionInterval(5 * time.Second)
    
    var (
        successCount int
        errorCount   int
        mu          sync.Mutex
    )
    
    dataChan := make(chan *collector.SystemData, 50)
    errorChan := make(chan error, 50)
    
    suite.setupDataCapture(dataChan)
    suite.setupErrorCapture(errorChan)
    
    // Simular carga adicional no sistema
    go suite.simulateSystemLoad(ctx)
    
    go suite.agent.Start(ctx)
    
    // Monitorar resultados
    done := make(chan bool)
    go func() {
        for {
            select {
            case <-dataChan:
                mu.Lock()
                successCount++
                mu.Unlock()
            case <-errorChan:
                mu.Lock()
                errorCount++
                mu.Unlock()
            case <-ctx.Done():
                done <- true
                return
            }
        }
    }()
    
    <-done
    
    // Validar desempenho sob carga
    totalOperations := successCount + errorCount
    assert.Greater(t, totalOperations, 10, "Deve ter pelo menos 10 operações")
    
    // Taxa de sucesso deve ser >= 80%
    successRate := float64(successCount) / float64(totalOperations) * 100
    assert.GreaterOrEqual(t, successRate, 80.0, 
        "Taxa de sucesso deve ser >= 80%%, foi %.2f%%", successRate)
}

// Métodos auxiliares para validação
func (its *IntegrationTestSuite) verifyCommonApplications(t *testing.T, apps []collector.Application) {
    commonApps := its.getCommonApplications()
    foundCount := 0
    
    for _, app := range apps {
        for _, common := range commonApps {
            if strings.Contains(strings.ToLower(app.Name), strings.ToLower(common)) {
                foundCount++
                break
            }
        }
    }
    
    assert.GreaterOrEqual(t, foundCount, len(commonApps)/2, 
        "Deve encontrar pelo menos metade das aplicações comuns")
}

func (its *IntegrationTestSuite) verifyServerServices(t *testing.T, services []collector.Service) {
    serverServices := its.getServerServices()
    foundCount := 0
    
    for _, service := range services {
        for _, server := range serverServices {
            if strings.Contains(strings.ToLower(service.Name), strings.ToLower(server)) {
                foundCount++
                break
            }
        }
    }
    
    assert.GreaterOrEqual(t, foundCount, len(serverServices)/2,
        "Deve encontrar pelo menos metade dos serviços de servidor")
}

func (its *IntegrationTestSuite) simulateSystemLoad(ctx context.Context) {
    // Simular carga do sistema com operações intensivas
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // Operações que consomem CPU/memória
            time.Sleep(100 * time.Millisecond)
        }
    }
}
```

## ✅ Critérios de Sucesso

### Funcionalidade
- [ ] Todos os testes de integração passando
- [ ] Cenários reais funcionando corretamente
- [ ] Comunicação robusta entre componentes
- [ ] Recuperação automática de erros

### Performance
- [ ] Coleta completa em < 30 segundos
- [ ] Uso de memória estável (< 100MB)
- [ ] Taxa de sucesso > 95% em operações normais
- [ ] Reconexão automática em < 30 segundos

### Qualidade
- [ ] Cobertura de testes > 80%
- [ ] Zero vazamentos de memória detectados
- [ ] Estabilidade em execução prolongada
- [ ] Resiliência a falhas de rede

## 🧪 Execução dos Testes

### Localmente
```bash
# Testes de integração completos
go test -v ./internal/integration/... -timeout=10m

# Testes de carga (apenas se necessário)
go test -v ./internal/integration/... -run=TestLoadAndStress -timeout=15m

# Testes de cenários reais
go test -v ./internal/integration/... -run=TestRealWorldScenarios -timeout=10m
```

### CI/CD
```yaml
# Adicionar ao pipeline existente
- name: Integration Tests
  run: |
    go test -v ./internal/integration/... -timeout=10m -coverprofile=integration-coverage.out
    go tool cover -html=integration-coverage.out -o integration-coverage.html
```

## 📚 Referências

### Testing
- [Go Integration Testing](https://golang.org/doc/tutorial/add-a-test)
- [Testify Documentation](https://github.com/stretchr/testify)
- [WebSocket Testing](https://github.com/gorilla/websocket/tree/master/examples)

### Performance
- [Go Performance Testing](https://golang.org/pkg/testing/#hdr-Benchmarks)
- [Memory Profiling](https://golang.org/blog/pprof)

## 🔄 Próximos Passos
Após completar esta task, prosseguir para:
- **Task 12**: Otimização de performance
- **Task 13**: Documentação final 