package agent

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"agente-poc/internal/collector"
	"agente-poc/internal/comms"
	"agente-poc/internal/executor"
	"agente-poc/internal/logging"
)

// AgentState representa o estado do agente
type AgentState int

const (
	StateStarting AgentState = iota
	StateRunning
	StateStopping
	StateStopped
	StateError
)

// String retorna a representação string do estado
func (s AgentState) String() string {
	switch s {
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// AgentMetrics contém métricas do agente
type AgentMetrics struct {
	StartTime          time.Time
	HeartbeatCount     int64
	InventoryCount     int64
	CommandsExecuted   int64
	CommandsSuccessful int64
	CommandsFailed     int64
	LastHeartbeat      time.Time
	LastInventory      time.Time
	LastCommand        time.Time
	ErrorCount         int64
	RetryCount         int64
	ConnectionAttempts int64
	ConnectionFailures int64
	mu                 sync.RWMutex
}

// RetryConfig contém configurações de retry
type RetryConfig struct {
	MaxRetries        int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
	JitterEnabled     bool
}

// CircuitBreakerConfig contém configurações do circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold int
	ResetTimeout     time.Duration
	HalfOpenMaxCalls int
}

// CircuitBreaker implementa um circuit breaker básico
type CircuitBreaker struct {
	config          CircuitBreakerConfig
	failures        int
	lastFailureTime time.Time
	state           string // "closed", "open", "half-open"
	halfOpenCalls   int
	mu              sync.RWMutex
}

// Agent representa a instância principal do agente
type Agent struct {
	config         *Config
	logger         logging.Logger
	collector      *collector.SystemCollector
	comms          *comms.Manager
	executor       *executor.Executor
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	mu             sync.RWMutex
	state          AgentState
	metrics        *AgentMetrics
	retryConfig    *RetryConfig
	circuitBreaker *CircuitBreaker
	commandChan    chan *comms.Command
	errorChan      chan error
	shutdownChan   chan struct{}
	healthStatus   *comms.SystemHealthStatus
}

// New cria uma nova instância do agente
func New(config *Config, logger logging.Logger) *Agent {
	ctx, cancel := context.WithCancel(context.Background())

	// Configuração de retry padrão
	retryConfig := &RetryConfig{
		MaxRetries:        config.MaxRetries,
		InitialBackoff:    config.RetryInterval,
		MaxBackoff:        config.RetryInterval * 10,
		BackoffMultiplier: 2.0,
		JitterEnabled:     true,
	}

	// Configuração do circuit breaker
	circuitBreakerConfig := CircuitBreakerConfig{
		FailureThreshold: 5,
		ResetTimeout:     30 * time.Second,
		HalfOpenMaxCalls: 3,
	}

	return &Agent{
		config:      config,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		state:       StateStarting,
		metrics:     &AgentMetrics{StartTime: time.Now()},
		retryConfig: retryConfig,
		circuitBreaker: &CircuitBreaker{
			config: circuitBreakerConfig,
			state:  "closed",
		},
		commandChan:  make(chan *comms.Command, 100),
		errorChan:    make(chan error, 100),
		shutdownChan: make(chan struct{}),
		healthStatus: &comms.SystemHealthStatus{
			Status: "healthy",
		},
	}
}

// Start inicia o agente e todos os seus componentes
func (a *Agent) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state != StateStarting {
		return fmt.Errorf("agent is not in starting state: %s", a.state)
	}

	a.logger.Info("Starting agent...")
	a.setState(StateStarting)

	// Inicializar collector
	a.collector = collector.New(a.config.CollectionInterval, a.logger)

	// Gerar machine_id automaticamente se não fornecido na configuração
	if a.config.MachineID == "" {
		a.logger.Info("Machine ID not provided in config, generating automatically...")

		// Coletar dados básicos para gerar machine_id
		inventory, err := a.collector.CollectInventory()
		if err != nil {
			a.logger.Warning("Failed to collect inventory for machine ID generation, using fallback: %v", err)

			// Fallback: usar informações básicas do sistema
			basicInfo, err := a.collector.CollectBasicInfo()
			if err != nil {
				a.setState(StateError)
				return fmt.Errorf("failed to generate machine ID: %w", err)
			}

			// Usar hostname como machine_id de fallback
			a.config.MachineID = fmt.Sprintf("auto-%s", basicInfo.Hostname)
		} else {
			// Usar machine_id gerado pelo collector
			a.config.MachineID = inventory.MachineID
		}

		a.logger.Info("Generated machine ID: %s", a.config.MachineID)
	} else {
		a.logger.Info("Using configured machine ID: %s", a.config.MachineID)
	}

	// Inicializar executor
	execConfig := &executor.Config{
		DefaultTimeout: a.config.CommandTimeout,
		MaxConcurrent:  10,
		Logger:         a.logger,
	}
	var err error
	a.executor, err = executor.New(execConfig)
	if err != nil {
		a.setState(StateError)
		return fmt.Errorf("failed to initialize executor: %w", err)
	}

	// Inicializar communications manager
	commConfig := &comms.Config{
		BackendURL:        a.config.BackendURL,
		WebSocketURL:      a.config.WebSocketURL,
		Token:             a.config.Token,
		MachineID:         a.config.MachineID,
		RetryInterval:     a.config.RetryInterval,
		HeartbeatInterval: a.config.HeartbeatInterval,
		Logger:            a.logger,
	}

	a.comms, err = comms.New(commConfig)
	if err != nil {
		a.setState(StateError)
		return fmt.Errorf("failed to initialize communications: %w", err)
	}

	// Marcar como running
	a.setState(StateRunning)

	// Iniciar goroutines
	a.wg.Add(5)

	// Goroutine para coleta de dados
	go a.runCollector()

	// Goroutine para comunicações
	go a.runCommunications()

	// Goroutine para loop principal
	go a.runMainLoop()

	// Goroutine para processamento de comandos
	go a.runCommandProcessor()

	// Goroutine para tratamento de erros
	go a.runErrorHandler()

	a.logger.Info("Agent started successfully")
	return nil
}

// Stop para o agente gracefully
func (a *Agent) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state == StateStopped || a.state == StateStopping {
		return nil
	}

	a.logger.Info("Stopping agent...")
	a.setState(StateStopping)

	// Cancelar contexto
	a.cancel()

	// Sinalizar shutdown
	close(a.shutdownChan)

	// Aguardar todas as goroutines terminarem
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	// Timeout para shutdown
	select {
	case <-done:
		a.logger.Info("Agent stopped successfully")
	case <-time.After(30 * time.Second):
		a.logger.Warning("Agent shutdown timeout - forcing stop")
	}

	a.setState(StateStopped)
	return nil
}

// IsRunning retorna se o agente está rodando
func (a *Agent) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state == StateRunning
}

// GetState retorna o estado atual do agente
func (a *Agent) GetState() AgentState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

// GetMetrics retorna as métricas do agente
func (a *Agent) GetMetrics() *AgentMetrics {
	a.metrics.mu.RLock()
	defer a.metrics.mu.RUnlock()

	// Retornar cópia das métricas
	return &AgentMetrics{
		StartTime:          a.metrics.StartTime,
		HeartbeatCount:     a.metrics.HeartbeatCount,
		InventoryCount:     a.metrics.InventoryCount,
		CommandsExecuted:   a.metrics.CommandsExecuted,
		CommandsSuccessful: a.metrics.CommandsSuccessful,
		CommandsFailed:     a.metrics.CommandsFailed,
		LastHeartbeat:      a.metrics.LastHeartbeat,
		LastInventory:      a.metrics.LastInventory,
		LastCommand:        a.metrics.LastCommand,
		ErrorCount:         a.metrics.ErrorCount,
		RetryCount:         a.metrics.RetryCount,
		ConnectionAttempts: a.metrics.ConnectionAttempts,
		ConnectionFailures: a.metrics.ConnectionFailures,
	}
}

// setState define o estado do agente
func (a *Agent) setState(state AgentState) {
	a.state = state
	a.logger.WithField("state", state.String()).Debug("Agent state changed")
}

// runCollector executa o loop de coleta de dados
func (a *Agent) runCollector() {
	defer a.wg.Done()

	a.logger.Info("Starting data collector...")

	ticker := time.NewTicker(a.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			a.logger.Info("Collector stopped")
			return
		case <-ticker.C:
			a.collectAndSendInventory()
		}
	}
}

// runCommunications executa o loop de comunicações
func (a *Agent) runCommunications() {
	defer a.wg.Done()

	a.logger.Info("Starting communications...")

	if err := a.comms.Start(a.ctx); err != nil {
		a.logger.WithField("error", err).Error("Failed to start communications")
		a.errorChan <- err
		return
	}

	a.logger.Info("Communications stopped")
}

// runMainLoop executa o loop principal do agente
func (a *Agent) runMainLoop() {
	defer a.wg.Done()

	a.logger.Info("Starting main loop...")

	// Remover heartbeat daqui pois o manager já tem seu próprio timer
	// heartbeatTicker := time.NewTicker(a.config.HeartbeatInterval)
	// defer heartbeatTicker.Stop()

	healthCheckTicker := time.NewTicker(10 * time.Second)
	defer healthCheckTicker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			a.logger.Info("Main loop stopped")
			return
		// case <-heartbeatTicker.C:
		// 	a.sendHeartbeatWithRetry()
		case <-healthCheckTicker.C:
			a.updateHealthStatus()
		}
	}
}

// runCommandProcessor executa o loop de processamento de comandos
func (a *Agent) runCommandProcessor() {
	defer a.wg.Done()

	a.logger.Info("Starting command processor...")

	for {
		select {
		case <-a.ctx.Done():
			a.logger.Info("Command processor stopped")
			return
		case command := <-a.commandChan:
			a.handleCommand(command)
		}
	}
}

// runErrorHandler executa o loop de tratamento de erros
func (a *Agent) runErrorHandler() {
	defer a.wg.Done()

	a.logger.Info("Starting error handler...")

	for {
		select {
		case <-a.ctx.Done():
			a.logger.Info("Error handler stopped")
			return
		case err := <-a.errorChan:
			a.handleError(err)
		}
	}
}

// collectAndSendInventory coleta e envia dados de inventário
func (a *Agent) collectAndSendInventory() {
	a.logger.Debug("Collecting and sending inventory...")

	// Coletar dados do sistema
	data, err := a.collector.CollectInventory()
	if err != nil {
		a.logger.WithField("error", err).Error("Failed to collect inventory data")
		a.errorChan <- err
		return
	}

	// Usar machine_id da configuração (que já foi resolvido no Start)
	// Se o inventory não tiver machine_id, usar o da configuração
	if data.MachineID == "" {
		data.MachineID = a.config.MachineID
	}

	// Enviar dados via communications
	if err := a.sendInventoryWithRetry(data); err != nil {
		a.logger.WithField("error", err).Error("Failed to send inventory data")
		a.errorChan <- err
		return
	}

	// Atualizar métricas
	a.metrics.mu.Lock()
	a.metrics.InventoryCount++
	a.metrics.LastInventory = time.Now()
	a.metrics.mu.Unlock()

	a.logger.Debug("Inventory sent successfully")
}

// sendInventoryWithRetry envia inventário com retry
func (a *Agent) sendInventoryWithRetry(data *collector.InventoryData) error {
	if !a.circuitBreaker.canExecute() {
		return fmt.Errorf("circuit breaker is open")
	}

	err := a.retryWithBackoff(func() error {
		return a.comms.SendInventory(data)
	})

	if err != nil {
		a.circuitBreaker.recordFailure()
		return err
	}

	a.circuitBreaker.recordSuccess()
	return nil
}

// handleCommand processa um comando recebido
func (a *Agent) handleCommand(command *comms.Command) {
	a.logger.WithFields(map[string]interface{}{
		"command_id":   command.ID,
		"command_type": command.Type,
		"command":      command.Command,
	}).Info("Processing command")

	// Verificar se o comando é suportado
	if !a.executor.IsSupported(command) {
		a.logger.WithField("command_type", command.Type).Warning("Unsupported command type")
		result := &comms.CommandResult{
			CommandID: command.ID,
			Status:    "error",
			Error:     fmt.Sprintf("Unsupported command type: %s", command.Type),
		}
		a.sendCommandResult(result)
		return
	}

	// Executar comando
	ctx, cancel := context.WithTimeout(a.ctx, a.executor.GetTimeout())
	defer cancel()

	result, err := a.executor.Execute(ctx, command)

	// Atualizar métricas
	a.metrics.mu.Lock()
	a.metrics.CommandsExecuted++
	a.metrics.LastCommand = time.Now()
	if err != nil {
		a.metrics.CommandsFailed++
	} else {
		a.metrics.CommandsSuccessful++
	}
	a.metrics.mu.Unlock()

	if err != nil {
		a.logger.WithFields(map[string]interface{}{
			"command_id": command.ID,
			"error":      err,
		}).Error("Command execution failed")
		a.errorChan <- err
	}

	// Enviar resultado
	if result != nil {
		a.sendCommandResult(result)
	}
}

// sendCommandResult envia resultado do comando
func (a *Agent) sendCommandResult(result *comms.CommandResult) {
	if err := a.comms.SendCommandResult(result); err != nil {
		a.logger.WithFields(map[string]interface{}{
			"command_id": result.CommandID,
			"error":      err,
		}).Error("Failed to send command result")
		a.errorChan <- err
	}
}

// handleError trata erros do agente
func (a *Agent) handleError(err error) {
	a.logger.WithField("error", err).Error("Handling agent error")

	// Atualizar métricas
	a.metrics.mu.Lock()
	a.metrics.ErrorCount++
	a.metrics.mu.Unlock()

	// Aqui pode implementar lógica específica de tratamento de erro
	// Por exemplo, notificar o backend sobre o erro
}

// updateHealthStatus atualiza o status de saúde do sistema
func (a *Agent) updateHealthStatus() {
	// TODO: Implementar coleta real de métricas de saúde
	a.healthStatus.CPUUsage = 25.0    // Simulado
	a.healthStatus.MemoryUsage = 60.0 // Simulado
	a.healthStatus.DiskUsage = 45.0   // Simulado

	// Determinar status geral
	if a.healthStatus.CPUUsage > 80 || a.healthStatus.MemoryUsage > 90 || a.healthStatus.DiskUsage > 95 {
		a.healthStatus.Status = "critical"
	} else if a.healthStatus.CPUUsage > 60 || a.healthStatus.MemoryUsage > 80 || a.healthStatus.DiskUsage > 85 {
		a.healthStatus.Status = "warning"
	} else {
		a.healthStatus.Status = "healthy"
	}
}

// retryWithBackoff executa uma função com retry e backoff exponencial
func (a *Agent) retryWithBackoff(fn func() error) error {
	var lastErr error
	backoff := a.retryConfig.InitialBackoff

	for attempt := 0; attempt <= a.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Aplicar jitter se habilitado
			if a.retryConfig.JitterEnabled {
				jitter := time.Duration(float64(backoff) * 0.1 * (2*rand.Float64() - 1))
				backoff += jitter
			}

			a.logger.WithFields(map[string]interface{}{
				"attempt": attempt,
				"backoff": backoff.String(),
			}).Debug("Retrying operation")

			// Atualizar métricas
			a.metrics.mu.Lock()
			a.metrics.RetryCount++
			a.metrics.mu.Unlock()

			select {
			case <-time.After(backoff):
			case <-a.ctx.Done():
				return a.ctx.Err()
			}

			// Calcular próximo backoff
			backoff = time.Duration(float64(backoff) * a.retryConfig.BackoffMultiplier)
			if backoff > a.retryConfig.MaxBackoff {
				backoff = a.retryConfig.MaxBackoff
			}
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		a.logger.WithFields(map[string]interface{}{
			"attempt": attempt,
			"error":   lastErr,
		}).Warning("Operation failed, will retry")
	}

	return fmt.Errorf("operation failed after %d attempts: %w", a.retryConfig.MaxRetries, lastErr)
}

// canExecute verifica se o circuit breaker permite execução
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case "closed":
		return true
	case "open":
		if now.Sub(cb.lastFailureTime) > cb.config.ResetTimeout {
			cb.state = "half-open"
			cb.halfOpenCalls = 0
			return true
		}
		return false
	case "half-open":
		return cb.halfOpenCalls < cb.config.HalfOpenMaxCalls
	default:
		return false
	}
}

// recordSuccess registra um sucesso no circuit breaker
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	if cb.state == "half-open" {
		cb.state = "closed"
	}
}

// recordFailure registra uma falha no circuit breaker
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.state == "half-open" {
		cb.state = "open"
	} else if cb.failures >= cb.config.FailureThreshold {
		cb.state = "open"
	}
}

// Health retorna informações de saúde do agente
func (a *Agent) Health() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	metrics := a.GetMetrics()

	return map[string]interface{}{
		"state":               a.state.String(),
		"machine_id":          a.config.MachineID,
		"backend_url":         a.config.BackendURL,
		"uptime":              time.Since(metrics.StartTime).String(),
		"heartbeat_count":     metrics.HeartbeatCount,
		"inventory_count":     metrics.InventoryCount,
		"commands_executed":   metrics.CommandsExecuted,
		"commands_successful": metrics.CommandsSuccessful,
		"commands_failed":     metrics.CommandsFailed,
		"error_count":         metrics.ErrorCount,
		"retry_count":         metrics.RetryCount,
		"last_heartbeat":      metrics.LastHeartbeat.Format(time.RFC3339),
		"last_inventory":      metrics.LastInventory.Format(time.RFC3339),
		"system_health":       a.healthStatus,
		"circuit_breaker":     a.circuitBreaker.state,
	}
}

// SubmitCommand submete um comando para execução
func (a *Agent) SubmitCommand(command *comms.Command) error {
	select {
	case a.commandChan <- command:
		return nil
	default:
		return fmt.Errorf("command queue is full")
	}
}
