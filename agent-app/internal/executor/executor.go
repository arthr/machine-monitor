package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"machine-monitor-agent/internal/types"

	"github.com/rs/zerolog/log"
)

// Executor responsável pela execução de comandos
type Executor struct {
	allowedCommands []string
	maxConcurrency  int
	semaphore       chan struct{}
}

// NewExecutor cria uma nova instância do executor
func NewExecutor(allowedCommands []string, maxConcurrency int) *Executor {
	return &Executor{
		allowedCommands: allowedCommands,
		maxConcurrency:  maxConcurrency,
		semaphore:       make(chan struct{}, maxConcurrency),
	}
}

// ExecuteCommand executa um comando
func (e *Executor) ExecuteCommand(ctx context.Context, command types.Command) types.CommandResult {
	startTime := time.Now()

	result := types.CommandResult{
		ID:        command.ID,
		Timestamp: startTime,
	}

	// Verifica se o comando é permitido
	if !e.isCommandAllowed(command.Type) {
		result.Success = false
		result.Error = fmt.Sprintf("comando não permitido: %s", command.Type)
		result.Duration = time.Since(startTime).Milliseconds()
		return result
	}

	// Adquire semáforo para controlar concorrência
	select {
	case e.semaphore <- struct{}{}:
		defer func() { <-e.semaphore }()
	case <-ctx.Done():
		result.Success = false
		result.Error = "timeout ao aguardar slot de execução"
		result.Duration = time.Since(startTime).Milliseconds()
		return result
	}

	// Executa o comando baseado no tipo
	switch command.Type {
	case types.CommandTypeShell:
		result = e.executeShellCommand(ctx, command)
	case types.CommandTypeInfo:
		result = e.executeInfoCommand(ctx, command)
	case types.CommandTypePing:
		result = e.executePingCommand(ctx, command)
	case types.CommandTypeRestart:
		result = e.executeRestartCommand(ctx, command)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("tipo de comando desconhecido: %s", command.Type)
	}

	result.Duration = time.Since(startTime).Milliseconds()

	log.Info().
		Str("command_id", command.ID).
		Str("type", command.Type).
		Bool("success", result.Success).
		Int64("duration_ms", result.Duration).
		Msg("Comando executado")

	return result
}

// executeShellCommand executa um comando shell
func (e *Executor) executeShellCommand(ctx context.Context, command types.Command) types.CommandResult {
	result := types.CommandResult{
		ID:        command.ID,
		Timestamp: time.Now(),
	}

	// Valida o comando
	if command.Command == "" {
		result.Success = false
		result.Error = "comando vazio"
		return result
	}

	// Sanitiza o comando
	sanitizedCmd := e.sanitizeCommand(command.Command)
	if sanitizedCmd == "" {
		result.Success = false
		result.Error = "comando contém caracteres perigosos"
		return result
	}

	// Configura timeout
	timeout := 30 * time.Second
	if command.Timeout > 0 {
		timeout = time.Duration(command.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Executa o comando
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "cmd", "/C", sanitizedCmd)
	default:
		cmd = exec.CommandContext(ctx, "sh", "-c", sanitizedCmd)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		}
	} else {
		result.Success = true
		result.ExitCode = 0
	}

	result.Output = string(output)
	return result
}

// executeInfoCommand executa comando de informação
func (e *Executor) executeInfoCommand(ctx context.Context, command types.Command) types.CommandResult {
	result := types.CommandResult{
		ID:        command.ID,
		Timestamp: time.Now(),
		Success:   true,
		ExitCode:  0,
	}

	// Informações básicas do sistema
	info := map[string]interface{}{
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"go_version":    runtime.Version(),
		"num_cpu":       runtime.NumCPU(),
		"num_goroutine": runtime.NumGoroutine(),
	}

	// Adiciona informações específicas baseadas nos args
	if len(command.Args) > 0 {
		switch command.Args[0] {
		case "memory":
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			info["memory"] = map[string]interface{}{
				"alloc":       m.Alloc,
				"total_alloc": m.TotalAlloc,
				"sys":         m.Sys,
				"num_gc":      m.NumGC,
			}
		case "version":
			info = map[string]interface{}{
				"agent_version": "1.0.0",
				"go_version":    runtime.Version(),
				"build_time":    time.Now().Format("2006-01-02 15:04:05"),
			}
		}
	}

	// Converte para JSON
	output, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("erro ao serializar informações: %v", err)
		return result
	}

	result.Output = string(output)
	return result
}

// executePingCommand executa comando de ping
func (e *Executor) executePingCommand(ctx context.Context, command types.Command) types.CommandResult {
	result := types.CommandResult{
		ID:        command.ID,
		Timestamp: time.Now(),
		Success:   true,
		ExitCode:  0,
	}

	target := "8.8.8.8"
	if len(command.Args) > 0 {
		target = command.Args[0]
	}

	// Sanitiza o target
	target = strings.TrimSpace(target)
	if target == "" {
		result.Success = false
		result.Error = "target de ping vazio"
		return result
	}

	// Configura timeout
	timeout := 10 * time.Second
	if command.Timeout > 0 {
		timeout = time.Duration(command.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Executa ping
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "ping", "-n", "4", target)
	default:
		cmd = exec.CommandContext(ctx, "ping", "-c", "4", target)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		}
	} else {
		result.Success = true
		result.ExitCode = 0
	}

	result.Output = string(output)
	return result
}

// executeRestartCommand executa comando de restart do agente
func (e *Executor) executeRestartCommand(ctx context.Context, command types.Command) types.CommandResult {
	result := types.CommandResult{
		ID:        command.ID,
		Timestamp: time.Now(),
		Success:   true,
		ExitCode:  0,
		Output:    "Comando de restart recebido. O agente será reiniciado.",
	}

	// Nota: O restart será tratado pelo agente principal
	// Este comando apenas sinaliza que o restart foi solicitado

	return result
}

// isCommandAllowed verifica se o comando é permitido
func (e *Executor) isCommandAllowed(commandType string) bool {
	for _, allowed := range e.allowedCommands {
		if allowed == commandType {
			return true
		}
	}
	return false
}

// sanitizeCommand sanitiza o comando removendo caracteres perigosos
func (e *Executor) sanitizeCommand(command string) string {
	// Remove caracteres perigosos
	dangerous := []string{
		";", "&&", "||", "|", ">", "<", ">>", "<<",
		"$(", "`", "&", "rm -rf", "del /f", "format",
		"sudo", "su", "passwd", "chmod 777",
	}

	cmd := strings.TrimSpace(command)

	for _, danger := range dangerous {
		if strings.Contains(strings.ToLower(cmd), strings.ToLower(danger)) {
			return ""
		}
	}

	return cmd
}

// GetStats retorna estatísticas do executor
func (e *Executor) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"max_concurrency":     e.maxConcurrency,
		"current_concurrency": e.maxConcurrency - len(e.semaphore),
		"allowed_commands":    e.allowedCommands,
	}
}
