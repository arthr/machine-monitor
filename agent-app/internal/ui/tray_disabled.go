//go:build linux && !cgo

package ui

import (
	"context"
	"machine-monitor-agent/internal/types"

	"github.com/rs/zerolog/log"
)

// TrayIcon representa o ícone na bandeja do sistema (versão disabled)
type TrayIcon struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewTrayIcon cria uma nova instância do ícone na bandeja (versão disabled)
func NewTrayIcon(onShowUI, onRestart, onExit func()) *TrayIcon {
	ctx, cancel := context.WithCancel(context.Background())

	log.Info().Msg("Tray icon desabilitado para esta plataforma")

	return &TrayIcon{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start inicia o ícone na bandeja (versão disabled)
func (t *TrayIcon) Start() {
	log.Info().Msg("Tray icon não disponível nesta plataforma")
}

// Stop para o ícone na bandeja (versão disabled)
func (t *TrayIcon) Stop() {
	t.cancel()
}

// UpdateStatus atualiza o status do agente (versão disabled)
func (t *TrayIcon) UpdateStatus(status *types.AgentStatus) {
	// Nada a fazer na versão disabled
}
