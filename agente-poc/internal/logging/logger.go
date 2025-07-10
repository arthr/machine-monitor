package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// LogLevel representa o nível de log
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

// String retorna a representação string do nível
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger define a interface para logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warning(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
	SetLevel(level LogLevel)
	GetLevel() LogLevel
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

// Config representa a configuração do logger
type Config struct {
	Level      LogLevel `json:"level"`
	Format     string   `json:"format"`      // "text", "json"
	Output     string   `json:"output"`      // "stdout", "stderr", "file"
	FilePath   string   `json:"file_path"`   // caminho do arquivo se output = "file"
	MaxSize    int      `json:"max_size"`    // tamanho máximo do arquivo em MB
	MaxBackups int      `json:"max_backups"` // número máximo de backups
	MaxAge     int      `json:"max_age"`     // idade máxima em dias
	Compress   bool     `json:"compress"`    // compactar arquivos antigos
}

// DefaultConfig retorna a configuração padrão
func DefaultConfig() *Config {
	return &Config{
		Level:      INFO,
		Format:     "text",
		Output:     "stdout",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}
}

// StandardLogger implementa a interface Logger
type StandardLogger struct {
	level  LogLevel
	config *Config
	logger *log.Logger
	fields map[string]interface{}
}

// NewLogger cria um novo logger com a configuração especificada
func NewLogger(config *Config) (Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	var output *os.File
	var err error

	switch config.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	case "file":
		if config.FilePath == "" {
			return nil, fmt.Errorf("file_path é obrigatório quando output = file")
		}
		output, err = os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("erro ao abrir arquivo de log: %w", err)
		}
	default:
		output = os.Stdout
	}

	logger := log.New(output, "", 0)

	return &StandardLogger{
		level:  config.Level,
		config: config,
		logger: logger,
		fields: make(map[string]interface{}),
	}, nil
}

// ParseLogLevel converte uma string em LogLevel
func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARNING", "WARN":
		return WARNING
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

// SetLevel define o nível de log
func (l *StandardLogger) SetLevel(level LogLevel) {
	l.level = level
}

// GetLevel retorna o nível atual de log
func (l *StandardLogger) GetLevel() LogLevel {
	return l.level
}

// WithField adiciona um campo ao contexto do log
func (l *StandardLogger) WithField(key string, value interface{}) Logger {
	newLogger := &StandardLogger{
		level:  l.level,
		config: l.config,
		logger: l.logger,
		fields: make(map[string]interface{}),
	}

	// Copiar campos existentes
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Adicionar novo campo
	newLogger.fields[key] = value

	return newLogger
}

// WithFields adiciona múltiplos campos ao contexto do log
func (l *StandardLogger) WithFields(fields map[string]interface{}) Logger {
	newLogger := &StandardLogger{
		level:  l.level,
		config: l.config,
		logger: l.logger,
		fields: make(map[string]interface{}),
	}

	// Copiar campos existentes
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Adicionar novos campos
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// Debug registra uma mensagem de debug
func (l *StandardLogger) Debug(msg string, args ...interface{}) {
	if l.level <= DEBUG {
		l.log(DEBUG, msg, args...)
	}
}

// Info registra uma mensagem de informação
func (l *StandardLogger) Info(msg string, args ...interface{}) {
	if l.level <= INFO {
		l.log(INFO, msg, args...)
	}
}

// Warning registra uma mensagem de aviso
func (l *StandardLogger) Warning(msg string, args ...interface{}) {
	if l.level <= WARNING {
		l.log(WARNING, msg, args...)
	}
}

// Error registra uma mensagem de erro
func (l *StandardLogger) Error(msg string, args ...interface{}) {
	if l.level <= ERROR {
		l.log(ERROR, msg, args...)
	}
}

// Fatal registra uma mensagem fatal e encerra o programa
func (l *StandardLogger) Fatal(msg string, args ...interface{}) {
	l.log(FATAL, msg, args...)
	os.Exit(1)
}

// log é o método interno para registrar mensagens
func (l *StandardLogger) log(level LogLevel, msg string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Formatar mensagem com argumentos
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	// Construir campos
	fieldsStr := ""
	if len(l.fields) > 0 {
		var parts []string
		for k, v := range l.fields {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		fieldsStr = " [" + strings.Join(parts, ", ") + "]"
	}

	// Formato da mensagem
	var logMsg string
	switch l.config.Format {
	case "json":
		logMsg = fmt.Sprintf(`{"timestamp":"%s","level":"%s","message":"%s","fields":%s}`,
			timestamp, level.String(), msg, l.fieldsToJSON())
	default:
		logMsg = fmt.Sprintf("[%s] %s: %s%s", timestamp, level.String(), msg, fieldsStr)
	}

	l.logger.Println(logMsg)
}

// fieldsToJSON converte campos para JSON
func (l *StandardLogger) fieldsToJSON() string {
	if len(l.fields) == 0 {
		return "{}"
	}

	var parts []string
	for k, v := range l.fields {
		parts = append(parts, fmt.Sprintf(`"%s":"%v"`, k, v))
	}

	return "{" + strings.Join(parts, ",") + "}"
}

// Global logger instance
var globalLogger Logger

// InitGlobalLogger inicializa o logger global
func InitGlobalLogger(config *Config) error {
	var err error
	globalLogger, err = NewLogger(config)
	return err
}

// GetGlobalLogger retorna o logger global
func GetGlobalLogger() Logger {
	if globalLogger == nil {
		// Inicializar com configuração padrão se não foi inicializado
		globalLogger, _ = NewLogger(DefaultConfig())
	}
	return globalLogger
}

// Funções de conveniência para o logger global
func Debug(msg string, args ...interface{}) {
	GetGlobalLogger().Debug(msg, args...)
}

func Info(msg string, args ...interface{}) {
	GetGlobalLogger().Info(msg, args...)
}

func Warning(msg string, args ...interface{}) {
	GetGlobalLogger().Warning(msg, args...)
}

func Error(msg string, args ...interface{}) {
	GetGlobalLogger().Error(msg, args...)
}

func Fatal(msg string, args ...interface{}) {
	GetGlobalLogger().Fatal(msg, args...)
}
