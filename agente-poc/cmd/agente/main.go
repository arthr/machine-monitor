package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"agente-poc/internal/agent"
	"agente-poc/internal/logging"
)

// Versão do agente
const (
	Version = "1.0.0"
	AppName = "agente-poc"
)

// Flags de linha de comando
var (
	configFile = flag.String("config", "configs/config.json", "Caminho para o arquivo de configuração")
	logLevel   = flag.String("log-level", "", "Nível de log (debug, info, warning, error)")
	verbose    = flag.Bool("verbose", false, "Modo verboso (equivalente a -log-level=debug)")
	version    = flag.Bool("version", false, "Mostrar versão e sair")
	help       = flag.Bool("help", false, "Mostrar ajuda e sair")
)

func main() {
	// Configurar recovery de panic
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "PANIC: %v\n", r)
			os.Exit(1)
		}
	}()

	// Parse das flags
	flag.Parse()

	// Mostrar versão
	if *version {
		fmt.Printf("%s versão %s\n", AppName, Version)
		os.Exit(0)
	}

	// Mostrar ajuda
	if *help {
		printHelp()
		os.Exit(0)
	}

	// Configurar logging inicial
	initialLogger, err := logging.NewLogger(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao criar logger inicial: %v\n", err)
		os.Exit(1)
	}

	if *verbose || *logLevel == "debug" {
		initialLogger.SetLevel(logging.DEBUG)
	} else if *logLevel != "" {
		level := logging.ParseLogLevel(*logLevel)
		initialLogger.SetLevel(level)
	}

	// Log inicial
	initialLogger.Info("Iniciando agente...")
	initialLogger.WithField("version", Version).Info("Versão do agente")

	// Determinar caminho do arquivo de configuração
	configPath := *configFile
	if !filepath.IsAbs(configPath) {
		// Se o caminho é relativo, fazer relativo ao diretório do executável
		exePath, err := os.Executable()
		if err != nil {
			initialLogger.WithField("error", err).Error("Erro ao obter caminho do executável")
			os.Exit(1)
		}
		configPath = filepath.Join(filepath.Dir(exePath), configPath)
	}

	// Carregar configuração
	initialLogger.WithField("config_path", configPath).Info("Carregando configuração")
	config, err := agent.LoadConfig(configPath)
	if err != nil {
		initialLogger.WithField("error", err).Error("Erro ao carregar configuração")
		os.Exit(1)
	}

	// Override de configuração com flags
	if *logLevel != "" {
		config.LogLevel = *logLevel
	}
	if *verbose {
		config.Debug = true
		config.LogLevel = "debug"
	}

	// Configurar logger final
	logger, err := logging.NewLogger(nil)
	if err != nil {
		initialLogger.WithField("error", err).Error("Erro ao criar logger final")
		os.Exit(1)
	}

	level := logging.ParseLogLevel(config.LogLevel)
	logger.SetLevel(level)

	if config.Debug {
		logger.WithField("config", config.String()).Debug("Configuração carregada")
	}

	// Criar contexto com cancelamento
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Configurar tratamento de sinais
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Criar instância do agente
	logger.Info("Criando instância do agente...")
	agentInstance := agent.New(config, logger)

	// Canal para controlar o shutdown
	shutdownChan := make(chan struct{})

	// Goroutine para tratamento de sinais
	go func() {
		sig := <-signalChan
		logger.WithField("signal", sig.String()).Info("Sinal recebido, iniciando shutdown...")

		// Cancelar contexto (usado no agente)
		cancel()

		// Sinalizar shutdown
		close(shutdownChan)
	}()

	// Iniciar agente
	logger.Info("Iniciando agente...")
	if err := agentInstance.Start(); err != nil {
		logger.WithField("error", err).Error("Erro ao iniciar agente")
		os.Exit(1)
	}

	logger.Info("Agente iniciado com sucesso - aguardando sinal de parada...")

	// Aguardar shutdown
	<-shutdownChan

	// Shutdown graceful com timeout
	logger.Info("Iniciando shutdown graceful...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	shutdownComplete := make(chan error)
	go func() {
		shutdownComplete <- agentInstance.Stop()
	}()

	select {
	case err := <-shutdownComplete:
		if err != nil {
			logger.WithField("error", err).Error("Erro durante shutdown")
			os.Exit(1)
		}
		logger.Info("Shutdown concluído com sucesso")
	case <-shutdownCtx.Done():
		logger.Warning("Timeout durante shutdown - forçando saída")
		os.Exit(1)
	}

	logger.Info("Agente finalizado")
}

// printHelp exibe informações de ajuda
func printHelp() {
	fmt.Printf(`%s - Agente de Monitoramento de Sistema

USAGE:
    %s [FLAGS]

FLAGS:
    -config string
        Caminho para o arquivo de configuração (default: "configs/config.json")
    
    -log-level string
        Nível de log (debug, info, warning, error)
    
    -verbose
        Modo verboso (equivalente a -log-level=debug)
    
    -version
        Mostrar versão e sair
    
    -help
        Mostrar esta ajuda e sair

VARIABLES DE AMBIENTE:
    AGENTE_CONFIG_PATH
        Caminho para o arquivo de configuração (sobrescreve -config)
    
    AGENTE_LOG_LEVEL
        Nível de log (sobrescreve -log-level)
    
    AGENTE_DEBUG
        Ativar modo debug (sobrescreve -verbose)

EXEMPLOS:
    # Executar com configuração padrão
    %s

    # Executar com arquivo de configuração específico
    %s -config /path/to/config.json

    # Executar em modo debug
    %s -verbose

    # Executar com nível de log específico
    %s -log-level warning

ARQUIVOS:
    configs/config.json     Arquivo de configuração padrão
    logs/                   Diretório de logs (se configurado)

Para mais informações, consulte a documentação.
`, AppName, AppName, AppName, AppName, AppName, AppName)
}

// init configura variáveis de ambiente
func init() {
	// Override de configuração com variáveis de ambiente
	if envConfig := os.Getenv("AGENTE_CONFIG_PATH"); envConfig != "" {
		*configFile = envConfig
	}

	if envLogLevel := os.Getenv("AGENTE_LOG_LEVEL"); envLogLevel != "" {
		*logLevel = envLogLevel
	}

	if envDebug := os.Getenv("AGENTE_DEBUG"); envDebug == "true" || envDebug == "1" {
		*verbose = true
	}
}
