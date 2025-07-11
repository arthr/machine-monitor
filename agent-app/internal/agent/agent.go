package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"machine-monitor-agent/internal/collector"
	"machine-monitor-agent/internal/communications"
	"machine-monitor-agent/internal/config"
	"machine-monitor-agent/internal/executor"
	"machine-monitor-agent/internal/types"
	"machine-monitor-agent/internal/ui"

	"github.com/rs/zerolog/log"
)

// Agent representa o agente principal
type Agent struct {
	config     *types.Config
	collector  *collector.Collector
	httpClient *communications.HTTPClient
	wsClient   *communications.WSClient
	executor   *executor.Executor
	trayIcon   *ui.TrayIcon
	webUI      *ui.WebUI

	// Estado
	status    *types.AgentStatus
	statusMu  sync.RWMutex
	startTime time.Time

	// Controle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Canais
	restartChan chan struct{}
}

// NewAgent cria uma nova instância do agente
func NewAgent(cfg *types.Config) *Agent {
	ctx, cancel := context.WithCancel(context.Background())

	return &Agent{
		config:      cfg,
		startTime:   time.Now(),
		ctx:         ctx,
		cancel:      cancel,
		restartChan: make(chan struct{}, 1),
		status: &types.AgentStatus{
			State:         types.StateStarting,
			LastHeartbeat: time.Time{},
			LastInventory: time.Time{},
			CommandsRun:   0,
			Errors:        0,
			Uptime:        0,
		},
	}
}

// Start inicia o agente
func (a *Agent) Start() error {
	log.Info().Msg("Iniciando Machine Monitor Agent...")

	// Cria diretórios necessários
	if err := config.EnsureDirectories(a.config); err != nil {
		return fmt.Errorf("erro ao criar diretórios: %w", err)
	}

	// Inicializa componentes
	if err := a.initializeComponents(); err != nil {
		return fmt.Errorf("erro ao inicializar componentes: %w", err)
	}

	// Atualiza status
	a.updateStatus(types.StateRunning)

	// Inicia loops principais
	a.startMainLoops()

	log.Info().Msg("Agent iniciado com sucesso")
	return nil
}

// Stop para o agente
func (a *Agent) Stop() error {
	log.Info().Msg("Parando Machine Monitor Agent...")

	a.updateStatus(types.StateStopping)

	// Cancela contexto
	a.cancel()

	// Para componentes
	if a.trayIcon != nil {
		a.trayIcon.Stop()
	}

	if a.wsClient != nil {
		a.wsClient.Disconnect()
	}

	if a.webUI != nil {
		a.webUI.Stop()
	}

	// Aguarda goroutines terminarem
	a.wg.Wait()

	a.updateStatus(types.StateStopped)

	log.Info().Msg("Agent parado com sucesso")
	return nil
}

// Wait aguarda o agente terminar
func (a *Agent) Wait() {
	<-a.ctx.Done()
}

// Restart reinicia o agente
func (a *Agent) Restart() error {
	log.Info().Msg("Reiniciando agente...")

	// Sinaliza restart
	select {
	case a.restartChan <- struct{}{}:
	default:
	}

	return nil
}

// initializeComponents inicializa todos os componentes
func (a *Agent) initializeComponents() error {
	// Inicializa collector
	cacheTTL := time.Duration(a.config.Agent.DataCacheTTL) * time.Second
	a.collector = collector.NewCollector(cacheTTL)

	// Inicializa HTTP client
	timeout := time.Duration(a.config.Server.Timeout) * time.Second
	a.httpClient = communications.NewHTTPClient(
		a.config.Server.BaseURL,
		a.config.Security.APIKey,
		timeout,
	)

	// Inicializa WebSocket client
	a.wsClient = communications.NewWSClient(
		a.config.Server.BaseURL,
		a.config.Security.APIKey,
		a.config.Agent.MachineID,
	)

	// Inicializa executor
	a.executor = executor.NewExecutor(
		a.config.Security.AllowedCommands,
		a.config.Agent.MaxConcurrency,
	)

	// Inicializa tray icon se habilitado
	if a.config.UI.ShowTrayIcon {
		a.trayIcon = ui.NewTrayIcon(
			a.showUI,
			func() { a.Restart() },
			func() { a.cancel() },
		)
		a.trayIcon.Start()
	}

	// Inicializa interface web
	a.webUI = ui.NewWebUI(a, a.config.UI.WebUIPort)
	if err := a.webUI.Start(); err != nil {
		return fmt.Errorf("erro ao iniciar interface web: %w", err)
	}

	return nil
}

// startMainLoops inicia os loops principais
func (a *Agent) startMainLoops() {
	// Loop principal
	a.wg.Add(1)
	go a.mainLoop()

	// Loop de heartbeat
	a.wg.Add(1)
	go a.heartbeatLoop()

	// Loop de inventário
	a.wg.Add(1)
	go a.inventoryLoop()

	// Loop de comandos
	a.wg.Add(1)
	go a.commandLoop()

	// Loop de status
	a.wg.Add(1)
	go a.statusLoop()
}

// mainLoop loop principal do agente
func (a *Agent) mainLoop() {
	defer a.wg.Done()

	// Conecta WebSocket
	if err := a.wsClient.Connect(a.ctx); err != nil {
		log.Error().Err(err).Msg("Erro ao conectar WebSocket")
	}

	// Registra máquina
	if err := a.registerMachine(); err != nil {
		log.Error().Err(err).Msg("Erro ao registrar máquina")
	}

	// Loop principal
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-a.restartChan:
			a.handleRestart()
			return
		case <-time.After(time.Minute):
			// Verifica conexões e estado geral
			a.healthCheck()
		}
	}
}

// heartbeatLoop loop de heartbeat
func (a *Agent) heartbeatLoop() {
	defer a.wg.Done()

	interval := time.Duration(a.config.Agent.HeartbeatInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.sendHeartbeat()
		}
	}
}

// inventoryLoop loop de inventário
func (a *Agent) inventoryLoop() {
	defer a.wg.Done()

	interval := time.Duration(a.config.Agent.InventoryInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Envia inventário inicial
	a.sendInventory()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.sendInventory()
		}
	}
}

// commandLoop loop de processamento de comandos
func (a *Agent) commandLoop() {
	defer a.wg.Done()

	commandChan := a.wsClient.GetCommandChannel()

	for {
		select {
		case <-a.ctx.Done():
			return
		case command := <-commandChan:
			a.processCommand(command)
		}
	}
}

// statusLoop loop de atualização de status
func (a *Agent) statusLoop() {
	defer a.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.updateUptime()
			if a.trayIcon != nil {
				a.trayIcon.UpdateStatus(a.getStatus())
			}
		}
	}
}

// registerMachine registra a máquina no backend
func (a *Agent) registerMachine() error {
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	inventory, err := a.collector.CollectInventory(ctx, a.config.Agent.MachineID)
	if err != nil {
		return fmt.Errorf("erro ao coletar inventário: %w", err)
	}

	if err := a.httpClient.RegisterMachine(ctx, a.config.Agent.MachineID, inventory); err != nil {
		return fmt.Errorf("erro ao registrar máquina: %w", err)
	}

	log.Info().Str("machine_id", a.config.Agent.MachineID).Msg("Máquina registrada com sucesso")
	return nil
}

// sendHeartbeat envia heartbeat para o backend
func (a *Agent) sendHeartbeat() {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	// Coleta informações básicas
	cpuUsage := 0.0
	memUsage := 0.0

	if hwInfo, err := a.collector.CollectHardwareInfo(ctx); err == nil {
		cpuUsage = hwInfo.CPU.Usage
		memUsage = hwInfo.Memory.UsedPercent
	}

	heartbeat := &types.HeartbeatData{
		MachineID: a.config.Agent.MachineID,
		Status:    a.getStatus().State,
		Uptime:    uint64(time.Since(a.startTime).Seconds()),
		CPUUsage:  cpuUsage,
		MemUsage:  memUsage,
		Timestamp: time.Now(),
	}

	if err := a.httpClient.SendHeartbeat(ctx, heartbeat); err != nil {
		log.Error().Err(err).Msg("Erro ao enviar heartbeat")
		a.incrementErrors()
	} else {
		a.statusMu.Lock()
		a.status.LastHeartbeat = time.Now()
		a.statusMu.Unlock()
	}
}

// sendInventory envia inventário para o backend
func (a *Agent) sendInventory() {
	ctx, cancel := context.WithTimeout(a.ctx, 60*time.Second)
	defer cancel()

	inventory, err := a.collector.CollectInventory(ctx, a.config.Agent.MachineID)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao coletar inventário")
		a.incrementErrors()
		return
	}

	if err := a.httpClient.SendInventory(ctx, inventory); err != nil {
		log.Error().Err(err).Msg("Erro ao enviar inventário")
		a.incrementErrors()
	} else {
		a.statusMu.Lock()
		a.status.LastInventory = time.Now()
		a.statusMu.Unlock()
		log.Info().Msg("Inventário enviado com sucesso")
	}
}

// processCommand processa um comando recebido
func (a *Agent) processCommand(command types.Command) {
	log.Info().Str("command_id", command.ID).Str("type", command.Type).Msg("Processando comando")

	ctx, cancel := context.WithTimeout(a.ctx, time.Duration(command.Timeout)*time.Second)
	defer cancel()

	// Executa comando
	result := a.executor.ExecuteCommand(ctx, command)

	// Atualiza estatísticas
	a.statusMu.Lock()
	a.status.CommandsRun++
	if !result.Success {
		a.status.Errors++
	}
	a.statusMu.Unlock()

	// Envia resultado via WebSocket
	if err := a.wsClient.SendResult(result); err != nil {
		log.Error().Err(err).Str("command_id", command.ID).Msg("Erro ao enviar resultado via WebSocket")

		// Fallback: envia via HTTP
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()

		if err := a.httpClient.SendCommandResult(ctx, a.config.Agent.MachineID, &result); err != nil {
			log.Error().Err(err).Str("command_id", command.ID).Msg("Erro ao enviar resultado via HTTP")
			a.incrementErrors()
		}
	}

	// Trata comando especial de restart
	if command.Type == types.CommandTypeRestart && result.Success {
		a.Restart()
	}
}

// healthCheck verifica saúde do agente
func (a *Agent) healthCheck() {
	// Verifica conexão WebSocket
	if !a.wsClient.IsConnected() {
		log.Warn().Msg("WebSocket desconectado, tentando reconectar...")
		if err := a.wsClient.Connect(a.ctx); err != nil {
			log.Error().Err(err).Msg("Erro ao reconectar WebSocket")
		}
	}

	// Verifica conectividade HTTP
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	if err := a.httpClient.Ping(ctx); err != nil {
		log.Error().Err(err).Msg("Erro ao fazer ping HTTP")
		a.incrementErrors()
	}
}

// handleRestart trata o restart do agente
func (a *Agent) handleRestart() {
	log.Info().Msg("Executando restart do agente...")

	// Para o agente atual
	a.Stop()

	// Reinicia o processo
	executable, err := os.Executable()
	if err != nil {
		log.Error().Err(err).Msg("Erro ao obter caminho do executável")
		return
	}

	cmd := exec.Command(executable, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg("Erro ao reiniciar processo")
		return
	}

	log.Info().Msg("Novo processo iniciado, finalizando atual...")
	os.Exit(0)
}

// showUI abre a interface web
func (a *Agent) showUI() {
	url := fmt.Sprintf("http://localhost:%d", a.config.UI.WebUIPort)

	var cmd *exec.Cmd
	switch {
	case os.Getenv("DISPLAY") != "":
		cmd = exec.Command("xdg-open", url)
	case os.Getenv("TERM_PROGRAM") == "Apple_Terminal":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("cmd", "/c", "start", url)
	}

	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg("Erro ao abrir interface web")
	}
}

// updateStatus atualiza o status do agente
func (a *Agent) updateStatus(state string) {
	a.statusMu.Lock()
	defer a.statusMu.Unlock()

	a.status.State = state
	a.status.Uptime = time.Since(a.startTime)
}

// updateUptime atualiza o uptime
func (a *Agent) updateUptime() {
	a.statusMu.Lock()
	defer a.statusMu.Unlock()

	a.status.Uptime = time.Since(a.startTime)
}

// incrementErrors incrementa contador de erros
func (a *Agent) incrementErrors() {
	a.statusMu.Lock()
	defer a.statusMu.Unlock()

	a.status.Errors++
}

// getStatus retorna cópia do status atual
func (a *Agent) getStatus() *types.AgentStatus {
	a.statusMu.RLock()
	defer a.statusMu.RUnlock()

	status := *a.status
	return &status
}

// GetConfig retorna a configuração atual
func (a *Agent) GetConfig() *types.Config {
	return a.config
}

// GetStatus retorna o status atual (método público para interface)
func (a *Agent) GetStatus() *types.AgentStatus {
	return a.getStatus()
}

// CollectSystemInfo coleta informações do sistema (método público para interface)
func (a *Agent) CollectSystemInfo(ctx context.Context) (*types.SystemInfo, error) {
	return a.collector.CollectSystemInfo(ctx)
}

// CollectHardwareInfo coleta informações de hardware (método público para interface)
func (a *Agent) CollectHardwareInfo(ctx context.Context) (*types.HardwareInfo, error) {
	return a.collector.CollectHardwareInfo(ctx)
}

// CollectSystemInfoFresh coleta informações do sistema sem cache
func (a *Agent) CollectSystemInfoFresh(ctx context.Context) (*types.SystemInfo, error) {
	a.collector.ClearCache()
	return a.collector.CollectSystemInfo(ctx)
}

// CollectHardwareInfoFresh coleta informações de hardware sem cache
func (a *Agent) CollectHardwareInfoFresh(ctx context.Context) (*types.HardwareInfo, error) {
	a.collector.ClearCache()
	return a.collector.CollectHardwareInfo(ctx)
}
