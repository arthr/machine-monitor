package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"machine-monitor-agent/internal/agent"
	"machine-monitor-agent/internal/config"
	"machine-monitor-agent/internal/types"

	"github.com/kardianos/service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	serviceName        = "MachineMonitorAgent"
	serviceDisplayName = "Machine Monitor Agent"
	serviceDescription = "Agent que monitora sistema, hardware e rede da máquina"
)

// Program implementa a interface service.Interface
type Program struct {
	agent      *agent.Agent
	configPath string
}

// Start inicia o serviço
func (p *Program) Start(s service.Service) error {
	log.Info().Msg("Iniciando serviço Machine Monitor Agent...")

	// Carrega configuração
	cfg, err := config.LoadConfig(p.configPath)
	if err != nil {
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	// Configura logging
	if err := p.setupLogging(cfg); err != nil {
		return fmt.Errorf("erro ao configurar logging: %w", err)
	}

	// Cria e inicia o agente
	p.agent = agent.NewAgent(cfg)
	if err := p.agent.Start(); err != nil {
		return fmt.Errorf("erro ao iniciar agente: %w", err)
	}

	// Inicia em goroutine para não bloquear
	go p.agent.Wait()

	log.Info().Msg("Serviço iniciado com sucesso")
	return nil
}

// Stop para o serviço
func (p *Program) Stop(s service.Service) error {
	log.Info().Msg("Parando serviço Machine Monitor Agent...")

	if p.agent != nil {
		if err := p.agent.Stop(); err != nil {
			log.Error().Err(err).Msg("Erro ao parar agente")
		}
	}

	log.Info().Msg("Serviço parado com sucesso")
	return nil
}

// setupLogging configura o sistema de logging
func (p *Program) setupLogging(cfg *types.Config) error {
	// Configura nível de log
	var level zerolog.Level
	switch cfg.Logging.Level {
	case types.LogLevelDebug:
		level = zerolog.DebugLevel
	case types.LogLevelInfo:
		level = zerolog.InfoLevel
	case types.LogLevelWarn:
		level = zerolog.WarnLevel
	case types.LogLevelError:
		level = zerolog.ErrorLevel
	default:
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)

	// Configura saída para arquivo se especificado
	if cfg.Logging.File != "" {
		// Cria diretório se não existir
		logDir := filepath.Dir(cfg.Logging.File)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("erro ao criar diretório de log: %w", err)
		}

		// Abre arquivo de log
		logFile, err := os.OpenFile(cfg.Logging.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("erro ao abrir arquivo de log: %w", err)
		}

		// Configura logger para escrever no arquivo
		log.Logger = log.Output(logFile)
	}

	// Adiciona timestamp e caller info
	log.Logger = log.Logger.With().
		Timestamp().
		Caller().
		Str("service", serviceName).
		Logger()

	return nil
}

func main() {
	// Flags da linha de comando
	var (
		configPath = flag.String("config", "", "Caminho para o arquivo de configuração")
		install    = flag.Bool("install", false, "Instala o serviço")
		uninstall  = flag.Bool("uninstall", false, "Remove o serviço")
		start      = flag.Bool("start", false, "Inicia o serviço")
		stop       = flag.Bool("stop", false, "Para o serviço")
		restart    = flag.Bool("restart", false, "Reinicia o serviço")
		console    = flag.Bool("console", false, "Executa em modo console (não como serviço)")
		version    = flag.Bool("version", false, "Mostra a versão")
	)
	flag.Parse()

	// Mostra versão
	if *version {
		fmt.Printf("Machine Monitor Agent v1.0.0\n")
		fmt.Printf("Build: %s\n", "development")
		return
	}

	// Configura logging básico
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Cria programa
	prg := &Program{
		configPath: *configPath,
	}

	// Configura serviço
	serviceConfig := &service.Config{
		Name:        serviceName,
		DisplayName: serviceDisplayName,
		Description: serviceDescription,
		Arguments:   []string{"-config", *configPath},
	}

	// Cria serviço
	s, err := service.New(prg, serviceConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Erro ao criar serviço")
	}

	// Trata comandos de controle do serviço
	if *install {
		err := s.Install()
		if err != nil {
			log.Fatal().Err(err).Msg("Erro ao instalar serviço")
		}
		log.Info().Msg("Serviço instalado com sucesso")
		return
	}

	if *uninstall {
		err := s.Uninstall()
		if err != nil {
			log.Fatal().Err(err).Msg("Erro ao remover serviço")
		}
		log.Info().Msg("Serviço removido com sucesso")
		return
	}

	if *start {
		err := s.Start()
		if err != nil {
			log.Fatal().Err(err).Msg("Erro ao iniciar serviço")
		}
		log.Info().Msg("Serviço iniciado com sucesso")
		return
	}

	if *stop {
		err := s.Stop()
		if err != nil {
			log.Fatal().Err(err).Msg("Erro ao parar serviço")
		}
		log.Info().Msg("Serviço parado com sucesso")
		return
	}

	if *restart {
		err := s.Restart()
		if err != nil {
			log.Fatal().Err(err).Msg("Erro ao reiniciar serviço")
		}
		log.Info().Msg("Serviço reiniciado com sucesso")
		return
	}

	// Executa em modo console ou como serviço
	if *console {
		// Modo console
		log.Info().Msg("Executando em modo console...")

		// Carrega configuração
		cfg, err := config.LoadConfig(*configPath)
		if err != nil {
			log.Fatal().Err(err).Msg("Erro ao carregar configuração")
		}

		// Configura logging
		if err := prg.setupLogging(cfg); err != nil {
			log.Fatal().Err(err).Msg("Erro ao configurar logging")
		}

		// Cria e inicia agente
		agentInstance := agent.NewAgent(cfg)
		if err := agentInstance.Start(); err != nil {
			log.Fatal().Err(err).Msg("Erro ao iniciar agente")
		}

		// Configura tratamento de sinais
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Aguarda sinal de parada
		log.Info().Msg("Agente executando. Pressione Ctrl+C para parar.")
		<-sigChan

		// Para o agente
		log.Info().Msg("Parando agente...")
		if err := agentInstance.Stop(); err != nil {
			log.Error().Err(err).Msg("Erro ao parar agente")
		}

		log.Info().Msg("Agente parado com sucesso")
	} else {
		// Modo serviço
		log.Info().Msg("Executando como serviço...")

		// Configura logger para serviço
		logger, err := s.Logger(nil)
		if err != nil {
			log.Fatal().Err(err).Msg("Erro ao configurar logger do serviço")
		}

		// Executa serviço
		err = s.Run()
		if err != nil {
			logger.Error(err)
			log.Fatal().Err(err).Msg("Erro ao executar serviço")
		}
	}
}
