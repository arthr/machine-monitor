package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"agente-poc/internal/collector"
	"agente-poc/internal/comms"
	"agente-poc/internal/logging"
)

// Agent representa a instância principal do agente
type Agent struct {
	config    *Config
	logger    logging.Logger
	collector *collector.SystemCollector
	comms     *comms.Manager
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.RWMutex
	running   bool
}

// New cria uma nova instância do agente
func New(config *Config, logger logging.Logger) *Agent {
	ctx, cancel := context.WithCancel(context.Background())

	return &Agent{
		config: config,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start inicia o agente e todos os seus componentes
func (a *Agent) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return fmt.Errorf("agent already running")
	}

	a.logger.Info("Starting agent...")

	// Inicializar collector
	a.collector = collector.New(a.config.CollectionInterval, a.logger)

	// Inicializar communications manager
	commConfig := &comms.Config{
		BackendURL:    a.config.BackendURL,
		WebSocketURL:  a.config.WebSocketURL,
		Token:         a.config.Token,
		MachineID:     a.config.MachineID,
		RetryInterval: a.config.RetryInterval,
		Logger:        a.logger,
	}

	var err error
	a.comms, err = comms.New(commConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize communications: %w", err)
	}

	// Marcar como running
	a.running = true

	// Iniciar componentes em goroutines separadas
	a.wg.Add(3)

	// Goroutine para coleta de dados
	go a.runCollector()

	// Goroutine para comunicações
	go a.runCommunications()

	// Goroutine para loop principal
	go a.runMainLoop()

	a.logger.Info("Agent started successfully")
	return nil
}

// Stop para o agente gracefully
func (a *Agent) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	a.logger.Info("Stopping agent...")

	// Cancelar contexto
	a.cancel()

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

	a.running = false
	return nil
}

// IsRunning retorna se o agente está rodando
func (a *Agent) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
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
			// Coletar dados do sistema
			data, err := a.collector.CollectInventory()
			if err != nil {
				a.logger.WithField("error", err).Error("Failed to collect inventory data")
				continue
			}

			// Enviar dados via communications
			if err := a.comms.SendInventory(data); err != nil {
				a.logger.WithField("error", err).Error("Failed to send inventory data")
			}
		}
	}
}

// runCommunications executa o loop de comunicações
func (a *Agent) runCommunications() {
	defer a.wg.Done()

	a.logger.Info("Starting communications...")

	if err := a.comms.Start(a.ctx); err != nil {
		a.logger.WithField("error", err).Error("Failed to start communications")
		return
	}

	a.logger.Info("Communications stopped")
}

// runMainLoop executa o loop principal do agente
func (a *Agent) runMainLoop() {
	defer a.wg.Done()

	a.logger.Info("Starting main loop...")

	heartbeatTicker := time.NewTicker(a.config.HeartbeatInterval)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			a.logger.Info("Main loop stopped")
			return
		case <-heartbeatTicker.C:
			// Enviar heartbeat
			if err := a.comms.SendHeartbeat(); err != nil {
				a.logger.WithField("error", err).Error("Failed to send heartbeat")
			}
		}
	}
}

// Health retorna informações de saúde do agente
func (a *Agent) Health() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return map[string]interface{}{
		"running":     a.running,
		"machine_id":  a.config.MachineID,
		"backend_url": a.config.BackendURL,
		"uptime":      time.Since(time.Now()).String(), // Será implementado corretamente depois
	}
}
