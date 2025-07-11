# Machine Monitor Agent

## ✅ Status: FUNCIONAL

Agente multiplataforma para monitoramento de sistema desenvolvido em Go que executa como serviço em background, coletando informações de hardware, sistema operacional e rede, com comunicação via API REST e WebSocket.

## 🚀 Funcionalidades Implementadas

### ✅ Coleta de Dados
- **Sistema**: OS, hostname, uptime, usuários, processos
- **Hardware**: CPU (modelo, cores, uso), memória, discos, interfaces de rede
- **Cache inteligente**: TTL configurável para otimização de performance
- **Coleta paralela**: Goroutines para melhor performance

### ✅ Comunicação
- **HTTP REST API**: Registro, heartbeat, inventário, comandos
- **WebSocket**: Comunicação em tempo real com reconexão automática
- **Suporte a timeout e retry**: Configurável por ambiente

### ✅ Interface de Usuário
- **Ícone na bandeja**: Menu contextual com status, abrir interface, reiniciar, sair
- **Interface web**: Dashboard HTML/CSS/JavaScript responsivo
- **APIs REST**: `/api/status`, `/api/system`, `/api/hardware`
- **Atualização automática**: Dashboard se atualiza a cada 10 segundos

### ✅ Execução de Comandos
- **Comandos shell**: Execução segura com sanitização
- **Comandos especiais**: info, ping, restart
- **Controle de concorrência**: Semáforo para limitar execuções simultâneas
- **Timeout por comando**: Configurável

### ✅ Serviço Multiplataforma
- **Windows**: Serviço do Windows
- **macOS**: LaunchAgent
- **Linux**: systemd service
- **Instalação/desinstalação**: Automática via linha de comando

### ✅ Configuração
- **Arquivo JSON**: Configuração centralizada
- **Validação**: Valores padrão e validação de configuração
- **Machine ID**: Geração automática de identificador único
- **Caminhos multiplataforma**: Configuração e logs em locais apropriados

## 🛠️ Problemas Resolvidos

### ✅ Imports e Dependências
- Corrigido import do pacote `net` do gopsutil
- Removida dependência problemática do webview
- Build tags para separar código dependente de GUI

### ✅ Compilação Multiplataforma
- **Linux**: Build com CGO_ENABLED=0 e tray desabilitado
- **Windows/macOS**: Build com CGO habilitado para suporte a tray
- **Build tags**: Separação de código GUI/headless

### ✅ Tray Icon
- Versão completa para Windows/macOS com systray
- Versão disabled para Linux headless
- Build tags condicionais: `//go:build !linux || (linux && cgo)`

## 📦 Compilação

```bash
# Build para plataforma atual
make build

# Build para Linux (headless)
make build-linux

# Build para Windows
make build-windows

# Build para macOS
make build-darwin

# Limpar builds
make clean
```

## 🔧 Configuração

O agente usa um arquivo de configuração JSON localizado em:
- **Windows**: `%APPDATA%\MachineMonitor\config.json`
- **macOS**: `~/Library/Application Support/MachineMonitor/config.json`
- **Linux**: `~/.config/MachineMonitor/config.json`

### Configuração Padrão

```json
{
  "server": {
    "base_url": "http://localhost:3000",
    "timeout": 30
  },
  "agent": {
    "machine_id": "auto-generated",
    "heartbeat_interval": 30,
    "inventory_interval": 300,
    "data_cache_ttl": 10,
    "max_concurrency": 5
  },
  "logging": {
    "level": "info",
    "file_enabled": true,
    "console_enabled": true
  },
  "ui": {
    "show_tray_icon": true,
    "web_ui_port": 8080
  },
  "security": {
    "api_key": "",
    "enable_tls": false,
    "allowed_commands": ["info", "ping", "restart"]
  }
}
```

## 🚀 Uso

### Modo Console (Desenvolvimento)
```bash
./machine-monitor-agent -console
```

### Instalação como Serviço
```bash
# Instalar
sudo ./machine-monitor-agent -install

# Iniciar
sudo ./machine-monitor-agent -start

# Parar
sudo ./machine-monitor-agent -stop

# Reiniciar
sudo ./machine-monitor-agent -restart

# Desinstalar
sudo ./machine-monitor-agent -uninstall
```

### Interface Web
Acesse `http://localhost:8080` para ver o dashboard com:
- Status do agente em tempo real
- Informações do sistema
- Métricas de hardware
- Gráficos de CPU e memória

### APIs REST
- `GET /api/status` - Status do agente
- `GET /api/system` - Informações do sistema
- `GET /api/hardware` - Informações de hardware

## 🏗️ Arquitetura

```
cmd/main.go                 # Ponto de entrada
├── internal/types/         # Definições de tipos
├── internal/config/        # Sistema de configuração
├── internal/collector/     # Coleta de dados do sistema
├── internal/communications/# Cliente HTTP/WebSocket
├── internal/executor/      # Execução de comandos
├── internal/agent/         # Agente principal
└── internal/ui/           # Interface (tray + web)
    ├── tray.go            # Tray para Windows/macOS
    ├── tray_disabled.go   # Tray disabled para Linux
    └── webui.go           # Interface web
```

## 📋 Dependências

- `github.com/shirou/gopsutil/v3` - Informações do sistema
- `github.com/gorilla/websocket` - Cliente WebSocket
- `github.com/getlantern/systray` - Ícone na bandeja (Windows/macOS)
- `github.com/kardianos/service` - Serviço multiplataforma
- `github.com/rs/zerolog` - Logging estruturado

## 🎯 Próximos Passos

1. **Servidor backend** - Implementar API REST e WebSocket server
2. **Dashboard web** - Interface de gerenciamento de múltiplos agentes
3. **Alertas** - Sistema de notificações baseado em métricas
4. **Plugins** - Sistema de extensões para coleta customizada
5. **Criptografia** - Implementar TLS e autenticação robusta

## 📝 Notas Técnicas

- O agente foi testado e está funcionando corretamente no macOS
- Build para Linux funciona sem interface gráfica (headless)
- Build para Windows requer ambiente Windows para teste completo
- Interface web responsiva funciona em qualquer navegador moderno
- Logs são salvos em arquivos rotativos com níveis configuráveis

## 🔍 Teste Realizado

O agente foi testado com sucesso:
- ✅ Compilação para Linux, Windows e macOS
- ✅ Execução em modo console
- ✅ Coleta de dados do sistema (CPU, memória, disco, rede)
- ✅ Interface web funcionando em http://localhost:8080
- ✅ APIs REST retornando dados corretos
- ✅ Configuração automática de diretórios

O projeto está **pronto para uso** e pode ser estendido conforme necessário. 