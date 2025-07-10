package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"agente-poc/internal/comms"
	"agente-poc/internal/logging"
)

// Executor implementa a execução segura de comandos remotos
type Executor struct {
	config    *Config
	logger    logging.Logger
	whitelist *CommandWhitelist
	semaphore chan struct{}
	metrics   *ExecutionMetrics
	mutex     sync.RWMutex
}

// Config contém a configuração do executor
type Config struct {
	MaxConcurrent   int                    `json:"max_concurrent"`
	DefaultTimeout  time.Duration          `json:"default_timeout"`
	MaxOutputSize   int                    `json:"max_output_size"`
	EnableMetrics   bool                   `json:"enable_metrics"`
	CustomWhitelist map[string]CommandSpec `json:"custom_whitelist,omitempty"`
	UserGroups      []string               `json:"user_groups,omitempty"`
	Logger          logging.Logger         `json:"-"`
}

// ExecutionMetrics coleta métricas de execução
type ExecutionMetrics struct {
	TotalExecutions  int64                   `json:"total_executions"`
	SuccessfulRuns   int64                   `json:"successful_runs"`
	FailedRuns       int64                   `json:"failed_runs"`
	RejectedCommands int64                   `json:"rejected_commands"`
	AverageTime      time.Duration           `json:"average_execution_time"`
	CommandStats     map[string]CommandStats `json:"command_stats"`
	LastExecution    time.Time               `json:"last_execution"`
	mutex            sync.RWMutex
}

// CommandStats estatísticas por comando
type CommandStats struct {
	Count         int64         `json:"count"`
	SuccessCount  int64         `json:"success_count"`
	FailureCount  int64         `json:"failure_count"`
	AverageTime   time.Duration `json:"average_time"`
	LastExecution time.Time     `json:"last_execution"`
}

// ExecutionResult resultado detalhado da execução
type ExecutionResult struct {
	Success       bool          `json:"success"`
	Output        string        `json:"output"`
	Error         string        `json:"error"`
	ExitCode      int           `json:"exit_code"`
	ExecutionTime time.Duration `json:"execution_time"`
	CommandSpec   CommandSpec   `json:"command_spec"`
	Sanitized     bool          `json:"sanitized"`
}

// New cria uma nova instância do executor
func New(config *Config) (*Executor, error) {
	if config == nil {
		config = &Config{
			MaxConcurrent:  5,
			DefaultTimeout: 30 * time.Second,
			MaxOutputSize:  1024 * 1024, // 1MB
			EnableMetrics:  true,
		}
	}

	if config.Logger == nil {
		logger, err := logging.NewLogger(nil)
		if err != nil {
			return nil, fmt.Errorf("erro ao criar logger: %w", err)
		}
		config.Logger = logger
	}

	// Obter whitelist baseada na plataforma
	var whitelist *CommandWhitelist
	switch runtime.GOOS {
	case "darwin":
		whitelist = GetMacOSWhitelist()
	case "linux":
		whitelist = GetMacOSWhitelist() // Usar mesma base por enquanto
	case "windows":
		whitelist = GetWindowsWhitelist()
	default:
		return nil, fmt.Errorf("plataforma não suportada: %s", runtime.GOOS)
	}

	// Adicionar comandos customizados se fornecidos
	if config.CustomWhitelist != nil {
		for name, spec := range config.CustomWhitelist {
			whitelist.Commands[name] = spec
		}
	}

	executor := &Executor{
		config:    config,
		logger:    config.Logger,
		whitelist: whitelist,
		semaphore: make(chan struct{}, config.MaxConcurrent),
		metrics: &ExecutionMetrics{
			CommandStats: make(map[string]CommandStats),
		},
	}

	executor.logger.WithField("platform", runtime.GOOS).Info("Executor inicializado")
	return executor, nil
}

// Execute executa um comando de forma segura
func (e *Executor) Execute(ctx context.Context, command *comms.Command) (*comms.CommandResult, error) {
	if command == nil {
		return nil, fmt.Errorf("comando não pode ser nulo")
	}

	startTime := time.Now()
	e.updateMetrics(func(m *ExecutionMetrics) {
		m.TotalExecutions++
		m.LastExecution = startTime
	})

	// Log da tentativa de execução
	e.logger.WithFields(map[string]interface{}{
		"command_id":   command.ID,
		"command_type": command.Type,
		"command":      command.Command,
		"args":         command.Args,
	}).Info("Iniciando execução de comando")

	// Controle de concorrência
	select {
	case e.semaphore <- struct{}{}:
		defer func() { <-e.semaphore }()
	case <-ctx.Done():
		e.updateMetrics(func(m *ExecutionMetrics) { m.RejectedCommands++ })
		return e.createErrorResult(command, "timeout na fila de execução", -1, startTime), ctx.Err()
	}

	// Executar comando baseado no tipo
	var result *comms.CommandResult
	var err error

	switch command.Type {
	case "shell":
		result, err = e.executeShellCommand(ctx, command, startTime)
	case "info":
		result, err = e.executeInfoCommand(ctx, command, startTime)
	case "ping":
		result, err = e.executePingCommand(ctx, command, startTime)
	default:
		e.updateMetrics(func(m *ExecutionMetrics) { m.RejectedCommands++ })
		return e.createErrorResult(command, "tipo de comando não suportado: "+command.Type, -1, startTime),
			fmt.Errorf("tipo de comando não suportado: %s", command.Type)
	}

	// Atualizar métricas
	duration := time.Since(startTime)
	if err != nil {
		e.updateMetrics(func(m *ExecutionMetrics) { m.FailedRuns++ })
		e.updateCommandStats(command.Command, duration, false)
	} else {
		e.updateMetrics(func(m *ExecutionMetrics) { m.SuccessfulRuns++ })
		e.updateCommandStats(command.Command, duration, true)
	}

	return result, err
}

// executeShellCommand executa um comando shell com validação de segurança
func (e *Executor) executeShellCommand(ctx context.Context, command *comms.Command, startTime time.Time) (*comms.CommandResult, error) {
	// Validar comando contra whitelist
	if err := e.whitelist.ValidateCommand(command.Command, command.Args); err != nil {
		e.logger.WithFields(map[string]interface{}{
			"command": command.Command,
			"args":    command.Args,
			"error":   err.Error(),
		}).Warning("Comando rejeitado pela whitelist")

		return e.createErrorResult(command, "comando rejeitado: "+err.Error(), -1, startTime), err
	}

	// Verificação adicional de segurança
	if !IsCommandSafe(command.Command, command.Args) {
		e.logger.WithFields(map[string]interface{}{
			"command": command.Command,
			"args":    command.Args,
		}).Warning("Comando rejeitado pela verificação de segurança")

		return e.createErrorResult(command, "comando considerado inseguro", -1, startTime),
			fmt.Errorf("comando considerado inseguro")
	}

	// Sanitizar argumentos
	sanitizedArgs := SanitizeArguments(command.Args)
	sanitized := !equalSlices(command.Args, sanitizedArgs)

	if sanitized {
		e.logger.WithFields(map[string]interface{}{
			"original":  command.Args,
			"sanitized": sanitizedArgs,
		}).Info("Argumentos sanitizados")
	}

	// Obter especificações do comando
	spec, exists := e.whitelist.GetCommandSpec(command.Command)
	if !exists {
		return e.createErrorResult(command, "especificações do comando não encontradas", -1, startTime),
			fmt.Errorf("especificações do comando não encontradas")
	}

	// Configurar timeout
	timeout := e.config.DefaultTimeout
	if spec.TimeoutSeconds > 0 {
		timeout = time.Duration(spec.TimeoutSeconds) * time.Second
	}
	if command.Timeout > 0 {
		timeout = time.Duration(command.Timeout) * time.Second
	}

	// Criar contexto com timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Executar comando
	e.logger.WithFields(map[string]interface{}{
		"command": command.Command,
		"args":    sanitizedArgs,
		"timeout": timeout.String(),
	}).Debug("Executando comando shell")

	cmd := exec.CommandContext(execCtx, command.Command, sanitizedArgs...)

	// Configurar ambiente limitado
	cmd.Env = []string{
		"PATH=/usr/bin:/bin:/usr/sbin:/sbin",
		"HOME=/tmp",
		"USER=nobody",
	}

	// Executar e capturar saída
	output, err := cmd.CombinedOutput()

	// Limitar tamanho da saída
	outputStr := string(output)
	if len(outputStr) > e.config.MaxOutputSize {
		outputStr = outputStr[:e.config.MaxOutputSize] + "\n... (saída truncada)"
	}

	// Determinar código de saída
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}

	// Criar resultado
	result := &comms.CommandResult{
		ID:            command.ID,
		CommandID:     command.ID,
		Status:        "success",
		Output:        outputStr,
		ExitCode:      exitCode,
		ExecutionTime: time.Since(startTime).Milliseconds(),
		Timestamp:     time.Now(),
	}

	if err != nil {
		result.Status = "error"
		result.Error = err.Error()

		e.logger.WithFields(map[string]interface{}{
			"command":   command.Command,
			"exit_code": exitCode,
			"error":     err.Error(),
		}).Error("Erro na execução do comando")
	} else {
		e.logger.WithFields(map[string]interface{}{
			"command":        command.Command,
			"exit_code":      exitCode,
			"execution_time": result.ExecutionTime,
			"output_size":    len(outputStr),
		}).Info("Comando executado com sucesso")
	}

	return result, nil
}

// executeInfoCommand executa comandos de coleta de informações
func (e *Executor) executeInfoCommand(ctx context.Context, command *comms.Command, startTime time.Time) (*comms.CommandResult, error) {
	e.logger.WithField("command_id", command.ID).Debug("Executando comando de informação")

	// Simular coleta de informações do sistema
	info := map[string]interface{}{
		"hostname":     getHostname(),
		"platform":     runtime.GOOS,
		"architecture": runtime.GOARCH,
		"uptime":       getUptime(),
		"timestamp":    time.Now().Unix(),
	}

	output := fmt.Sprintf("Informações do sistema coletadas: %+v", info)

	return &comms.CommandResult{
		ID:            command.ID,
		CommandID:     command.ID,
		Status:        "success",
		Output:        output,
		ExitCode:      0,
		ExecutionTime: time.Since(startTime).Milliseconds(),
		Timestamp:     time.Now(),
	}, nil
}

// executePingCommand executa comando de ping
func (e *Executor) executePingCommand(ctx context.Context, command *comms.Command, startTime time.Time) (*comms.CommandResult, error) {
	e.logger.WithField("command_id", command.ID).Debug("Executando comando de ping")

	return &comms.CommandResult{
		ID:            command.ID,
		CommandID:     command.ID,
		Status:        "success",
		Output:        "pong",
		ExitCode:      0,
		ExecutionTime: time.Since(startTime).Milliseconds(),
		Timestamp:     time.Now(),
	}, nil
}

// createErrorResult cria um resultado de erro padronizado
func (e *Executor) createErrorResult(command *comms.Command, errorMsg string, exitCode int, startTime time.Time) *comms.CommandResult {
	return &comms.CommandResult{
		ID:            command.ID,
		CommandID:     command.ID,
		Status:        "error",
		Error:         errorMsg,
		ExitCode:      exitCode,
		ExecutionTime: time.Since(startTime).Milliseconds(),
		Timestamp:     time.Now(),
	}
}

// GetMetrics retorna as métricas de execução
func (e *Executor) GetMetrics() ExecutionMetrics {
	e.metrics.mutex.RLock()
	defer e.metrics.mutex.RUnlock()

	// Fazer uma cópia das métricas
	metrics := ExecutionMetrics{
		TotalExecutions:  e.metrics.TotalExecutions,
		SuccessfulRuns:   e.metrics.SuccessfulRuns,
		FailedRuns:       e.metrics.FailedRuns,
		RejectedCommands: e.metrics.RejectedCommands,
		AverageTime:      e.metrics.AverageTime,
		LastExecution:    e.metrics.LastExecution,
		CommandStats:     make(map[string]CommandStats),
	}

	// Copiar estatísticas de comandos
	for k, v := range e.metrics.CommandStats {
		metrics.CommandStats[k] = v
	}

	return metrics
}

// IsSupported verifica se um comando é suportado
func (e *Executor) IsSupported(command *comms.Command) bool {
	if command == nil {
		return false
	}

	switch command.Type {
	case "shell":
		return e.whitelist.ValidateCommand(command.Command, command.Args) == nil
	case "info", "ping":
		return true
	default:
		return false
	}
}

// GetTimeout retorna o timeout configurado
func (e *Executor) GetTimeout() time.Duration {
	return e.config.DefaultTimeout
}

// GetWhitelist retorna a whitelist atual
func (e *Executor) GetWhitelist() *CommandWhitelist {
	return e.whitelist
}

// updateMetrics atualiza as métricas de forma thread-safe
func (e *Executor) updateMetrics(updateFunc func(*ExecutionMetrics)) {
	if !e.config.EnableMetrics {
		return
	}

	e.metrics.mutex.Lock()
	defer e.metrics.mutex.Unlock()
	updateFunc(e.metrics)
}

// updateCommandStats atualiza estatísticas de um comando específico
func (e *Executor) updateCommandStats(command string, duration time.Duration, success bool) {
	if !e.config.EnableMetrics {
		return
	}

	e.metrics.mutex.Lock()
	defer e.metrics.mutex.Unlock()

	stats, exists := e.metrics.CommandStats[command]
	if !exists {
		stats = CommandStats{}
	}

	stats.Count++
	stats.LastExecution = time.Now()

	if success {
		stats.SuccessCount++
	} else {
		stats.FailureCount++
	}

	// Calcular média móvel simples
	if stats.Count == 1 {
		stats.AverageTime = duration
	} else {
		stats.AverageTime = (stats.AverageTime + duration) / 2
	}

	e.metrics.CommandStats[command] = stats
}

// Funções auxiliares
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func getUptime() int64 {
	// Implementação simplificada - retorna tempo desde o início do processo
	return int64(time.Since(time.Now().Add(-time.Hour)).Seconds())
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
