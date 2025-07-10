package executor

import (
	"context"
	"fmt"
	"time"

	"agente-poc/internal/comms"
	"agente-poc/internal/logging"
)

// Executor define a interface para execução de comandos
type Executor interface {
	Execute(ctx context.Context, command *comms.Command) (*comms.CommandResult, error)
	IsSupported(command *comms.Command) bool
	GetTimeout() time.Duration
}

// Config contém a configuração do executor
type Config struct {
	DefaultTimeout time.Duration
	MaxConcurrent  int
	Logger         logging.Logger
}

// CommandExecutor implementa a interface Executor
type CommandExecutor struct {
	config    *Config
	logger    logging.Logger
	semaphore chan struct{}
}

// New cria uma nova instância do executor
func New(config *Config) *CommandExecutor {
	if config == nil {
		config = &Config{
			DefaultTimeout: 30 * time.Second,
			MaxConcurrent:  10,
		}
	}

	if config.Logger == nil {
		// Usar um logger padrão se não fornecido
		logger, _ := logging.NewLogger(nil)
		config.Logger = logger
	}

	return &CommandExecutor{
		config:    config,
		logger:    config.Logger,
		semaphore: make(chan struct{}, config.MaxConcurrent),
	}
}

// Execute executa um comando específico
func (e *CommandExecutor) Execute(ctx context.Context, command *comms.Command) (*comms.CommandResult, error) {
	if command == nil {
		return nil, fmt.Errorf("command cannot be nil")
	}

	// Controle de concorrência
	select {
	case e.semaphore <- struct{}{}:
		defer func() { <-e.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	e.logger.WithFields(map[string]interface{}{
		"command_id":   command.ID,
		"command_type": command.Type,
		"command":      command.Command,
	}).Info("Executing command")

	// Contexto com timeout
	timeout := e.config.DefaultTimeout
	if command.Timeout > 0 {
		timeout = time.Duration(command.Timeout) * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Executar comando baseado no tipo
	result, err := e.executeByType(execCtx, command)
	if err != nil {
		e.logger.WithFields(map[string]interface{}{
			"command_id": command.ID,
			"error":      err,
		}).Error("Command execution failed")

		return &comms.CommandResult{
			CommandID:     command.ID,
			Status:        "error",
			Error:         err.Error(),
			ExitCode:      -1,
			ExecutionTime: 0,
			Timestamp:     time.Now(),
		}, err
	}

	e.logger.WithFields(map[string]interface{}{
		"command_id": command.ID,
		"exit_code":  result.ExitCode,
	}).Info("Command executed successfully")

	return result, nil
}

// IsSupported verifica se o comando é suportado
func (e *CommandExecutor) IsSupported(command *comms.Command) bool {
	if command == nil {
		return false
	}

	switch command.Type {
	case "shell", "info", "ping", "restart":
		return true
	default:
		return false
	}
}

// GetTimeout retorna o timeout padrão
func (e *CommandExecutor) GetTimeout() time.Duration {
	return e.config.DefaultTimeout
}

// executeByType executa o comando baseado no tipo
func (e *CommandExecutor) executeByType(ctx context.Context, command *comms.Command) (*comms.CommandResult, error) {
	startTime := time.Now()

	var result *comms.CommandResult
	var err error

	switch command.Type {
	case "shell":
		result, err = e.executeShellCommand(ctx, command)
	case "info":
		result, err = e.executeInfoCommand(ctx, command)
	case "ping":
		result, err = e.executePingCommand(ctx, command)
	case "restart":
		result, err = e.executeRestartCommand(ctx, command)
	default:
		return nil, fmt.Errorf("unsupported command type: %s", command.Type)
	}

	// Calcular duração
	duration := time.Since(startTime)
	e.logger.WithFields(map[string]interface{}{
		"command_id": command.ID,
		"duration":   duration.String(),
	}).Debug("Command execution completed")

	return result, err
}

// executeShellCommand executa um comando shell
func (e *CommandExecutor) executeShellCommand(ctx context.Context, command *comms.Command) (*comms.CommandResult, error) {
	// TODO: Implementar execução real de comando shell
	// Por enquanto, simular
	e.logger.WithField("command", command.Command).Debug("Simulating shell command execution")

	return &comms.CommandResult{
		CommandID:     command.ID,
		Status:        "success",
		Output:        "Command executed successfully (simulated)",
		ExitCode:      0,
		ExecutionTime: 1000, // 1 segundo simulado em ms
		Timestamp:     time.Now(),
	}, nil
}

// executeInfoCommand executa um comando de informação
func (e *CommandExecutor) executeInfoCommand(ctx context.Context, command *comms.Command) (*comms.CommandResult, error) {
	// TODO: Implementar coleta de informações do sistema
	e.logger.Debug("Executing info command")

	return &comms.CommandResult{
		CommandID:     command.ID,
		Status:        "success",
		Output:        "System info collected successfully (simulated)",
		ExitCode:      0,
		ExecutionTime: 500, // 0.5 segundos simulado em ms
		Timestamp:     time.Now(),
	}, nil
}

// executePingCommand executa um comando de ping
func (e *CommandExecutor) executePingCommand(ctx context.Context, command *comms.Command) (*comms.CommandResult, error) {
	e.logger.Debug("Executing ping command")

	return &comms.CommandResult{
		CommandID:     command.ID,
		Status:        "success",
		Output:        "pong",
		ExitCode:      0,
		ExecutionTime: 100, // 0.1 segundos em ms
		Timestamp:     time.Now(),
	}, nil
}

// executeRestartCommand executa um comando de reinicialização
func (e *CommandExecutor) executeRestartCommand(ctx context.Context, command *comms.Command) (*comms.CommandResult, error) {
	e.logger.Warning("Restart command received - scheduling restart")

	// TODO: Implementar reinicialização real do agente
	return &comms.CommandResult{
		CommandID:     command.ID,
		Status:        "success",
		Output:        "Restart scheduled",
		ExitCode:      0,
		ExecutionTime: 200, // 0.2 segundos em ms
		Timestamp:     time.Now(),
	}, nil
}
