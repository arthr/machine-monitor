//go:build !linux || (linux && cgo)

package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"machine-monitor-agent/internal/types"

	"github.com/getlantern/systray"
	"github.com/rs/zerolog/log"
)

// TrayIcon representa o ícone na bandeja do sistema
type TrayIcon struct {
	status    *types.AgentStatus
	onShowUI  func()
	onRestart func()
	onExit    func()

	// Menu items
	statusItem  *systray.MenuItem
	showUIItem  *systray.MenuItem
	restartItem *systray.MenuItem
	exitItem    *systray.MenuItem

	// Controle
	updateChan chan *types.AgentStatus
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewTrayIcon cria uma nova instância do ícone na bandeja
func NewTrayIcon(onShowUI, onRestart, onExit func()) *TrayIcon {
	ctx, cancel := context.WithCancel(context.Background())

	return &TrayIcon{
		onShowUI:   onShowUI,
		onRestart:  onRestart,
		onExit:     onExit,
		updateChan: make(chan *types.AgentStatus, 10),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start inicia o ícone na bandeja
func (t *TrayIcon) Start() {
	go func() {
		systray.Run(t.onReady, t.onExit)
	}()
}

// Stop para o ícone na bandeja
func (t *TrayIcon) Stop() {
	t.cancel()
	systray.Quit()
}

// UpdateStatus atualiza o status do agente
func (t *TrayIcon) UpdateStatus(status *types.AgentStatus) {
	select {
	case t.updateChan <- status:
	default:
		// Canal cheio, ignora a atualização
	}
}

// onReady callback chamado quando o systray está pronto
func (t *TrayIcon) onReady() {
	// Define o ícone inicial
	iconData := getIconData()
	if len(iconData) > 0 {
		systray.SetIcon(iconData)
	}
	systray.SetTitle("Machine Monitor")
	systray.SetTooltip("Machine Monitor Agent")

	// Cria os itens do menu
	t.statusItem = systray.AddMenuItem("Status: Iniciando...", "Status atual do agente")
	t.statusItem.Disable()

	systray.AddSeparator()

	t.showUIItem = systray.AddMenuItem("Abrir Interface", "Abre a interface web do agente")
	t.restartItem = systray.AddMenuItem("Reiniciar Agente", "Reinicia o agente")

	systray.AddSeparator()

	t.exitItem = systray.AddMenuItem("Sair", "Fecha o agente")

	// Inicia o loop de eventos
	go t.eventLoop()
}

// eventLoop loop principal de eventos
func (t *TrayIcon) eventLoop() {
	for {
		select {
		case <-t.ctx.Done():
			return

		case status := <-t.updateChan:
			t.status = status
			t.updateStatusDisplay()

		case <-t.showUIItem.ClickedCh:
			log.Info().Msg("Menu: Abrir Interface clicado")
			if t.onShowUI != nil {
				go t.onShowUI()
			}

		case <-t.restartItem.ClickedCh:
			log.Info().Msg("Menu: Reiniciar Agente clicado")
			if t.onRestart != nil {
				go t.onRestart()
			}

		case <-t.exitItem.ClickedCh:
			log.Info().Msg("Menu: Sair clicado")
			if t.onExit != nil {
				go t.onExit()
			}
			return
		}
	}
}

// updateStatusDisplay atualiza a exibição do status
func (t *TrayIcon) updateStatusDisplay() {
	if t.status == nil || t.statusItem == nil {
		return
	}

	statusText := fmt.Sprintf("Status: %s", t.getStatusText(t.status.State))
	t.statusItem.SetTitle(statusText)

	// Atualiza tooltip com informações detalhadas
	tooltip := fmt.Sprintf("Machine Monitor Agent\nStatus: %s\nUptime: %s\nComandos: %d\nErros: %d",
		t.getStatusText(t.status.State),
		t.formatDuration(t.status.Uptime),
		t.status.CommandsRun,
		t.status.Errors,
	)
	systray.SetTooltip(tooltip)

	// Atualiza ícone baseado no status - com tratamento de erro
	iconData := t.getStatusIcon(t.status.State)
	if len(iconData) > 0 {
		systray.SetIcon(iconData)
	}
}

// getStatusText retorna texto amigável para o status
func (t *TrayIcon) getStatusText(state string) string {
	switch state {
	case types.StateStarting:
		return "Iniciando"
	case types.StateRunning:
		return "Executando"
	case types.StateStopping:
		return "Parando"
	case types.StateStopped:
		return "Parado"
	case types.StateError:
		return "Erro"
	default:
		return "Desconhecido"
	}
}

// formatDuration formata duração de forma amigável
func (t *TrayIcon) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}

// getStatusIcon retorna ícone baseado no status
func (t *TrayIcon) getStatusIcon(state string) []byte {
	switch state {
	case types.StateRunning:
		return getGreenIconData()
	case types.StateError:
		return getRedIconData()
	case types.StateStarting, types.StateStopping:
		return getYellowIconData()
	default:
		return getGrayIconData()
	}
}

// getExecutableDir retorna o diretório onde está o executável
func getExecutableDir() string {
	executable, err := os.Executable()
	if err != nil {
		log.Error().Err(err).Msg("Erro ao obter caminho do executável")
		return "."
	}
	return filepath.Dir(executable)
}

// getIconPath retorna o caminho absoluto para um arquivo de ícone
func getIconPath(filename string) string {
	execDir := getExecutableDir()
	return filepath.Join(execDir, "assets", filename)
}

// getIconData retorna dados do ícone padrão (azul)
func getIconData() []byte {
	return getIcon("blue.ico")
}

// getGreenIconData retorna ícone verde (executando)
func getGreenIconData() []byte {
	return getIcon("green.ico")
}

// getRedIconData retorna ícone vermelho (erro)
func getRedIconData() []byte {
	return getIcon("red.ico")
}

// getYellowIconData retorna ícone amarelo (transição)
func getYellowIconData() []byte {
	return getIcon("yellow.ico")
}

// getGrayIconData retorna ícone cinza (parado)
func getGrayIconData() []byte {
	return getIcon("gray.ico")
}

// getIcon carrega um arquivo de ícone com tratamento de erro
func getIcon(filename string) []byte {
	iconPath := getIconPath(filename)

	log.Debug().Str("path", iconPath).Msg("Tentando carregar ícone")

	data, err := os.ReadFile(iconPath)
	if err != nil {
		log.Error().Err(err).Str("path", iconPath).Msg("Erro ao carregar ícone")
		return getIcon("../../assets/blue.ico")
	}

	log.Debug().Str("path", iconPath).Int("size", len(data)).Msg("Ícone carregado com sucesso")
	return data
}
