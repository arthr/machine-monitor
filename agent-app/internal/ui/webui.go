package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"machine-monitor-agent/internal/types"

	"github.com/rs/zerolog/log"
)

// WebUI representa a interface web
type WebUI struct {
	server *http.Server
	agent  AgentInterface
	port   int
	ctx    context.Context
	cancel context.CancelFunc
}

// AgentInterface interface para acessar dados do agente
type AgentInterface interface {
	GetConfig() *types.Config
	GetStatus() *types.AgentStatus
	CollectSystemInfo(ctx context.Context) (*types.SystemInfo, error)
	CollectHardwareInfo(ctx context.Context) (*types.HardwareInfo, error)
	CollectSystemInfoFresh(ctx context.Context) (*types.SystemInfo, error)
	CollectHardwareInfoFresh(ctx context.Context) (*types.HardwareInfo, error)
}

// NewWebUI cria uma nova instância da interface web
func NewWebUI(agent AgentInterface, port int) *WebUI {
	ctx, cancel := context.WithCancel(context.Background())

	return &WebUI{
		agent:  agent,
		port:   port,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start inicia o servidor web
func (w *WebUI) Start() error {
	mux := http.NewServeMux()

	// Rotas
	mux.HandleFunc("/", w.handleHome)
	mux.HandleFunc("/api/status", w.handleAPIStatus)
	mux.HandleFunc("/api/system", w.handleAPISystem)
	mux.HandleFunc("/api/system/fresh", w.handleAPISystemFresh)
	mux.HandleFunc("/api/hardware", w.handleAPIHardware)
	mux.HandleFunc("/api/hardware/fresh", w.handleAPIHardwareFresh)
	mux.HandleFunc("/static/", w.handleStatic)

	// Configura servidor
	w.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", w.port),
		Handler: mux,
	}

	// Inicia servidor em goroutine
	go func() {
		log.Info().Int("port", w.port).Msg("Iniciando servidor web")
		if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("Erro no servidor web")
		}
	}()

	return nil
}

// Stop para o servidor web
func (w *WebUI) Stop() error {
	w.cancel()

	if w.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := w.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("erro ao parar servidor web: %w", err)
		}
	}

	log.Info().Msg("Servidor web parado")
	return nil
}

// handleHome trata a página inicial
func (w *WebUI) handleHome(rw http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(rw, r)
		return
	}

	tmpl := `
<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Machine Monitor Agent</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
            color: #333;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .header h1 {
            color: #2c3e50;
            margin-bottom: 10px;
        }
        .status {
            display: inline-block;
            padding: 5px 15px;
            border-radius: 20px;
            font-weight: bold;
            text-transform: uppercase;
            font-size: 12px;
        }
        .status.running { background-color: #27ae60; color: white; }
        .status.error { background-color: #e74c3c; color: white; }
        .status.starting { background-color: #f39c12; color: white; }
        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .card {
            background: white;
            border-radius: 10px;
            padding: 20px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .card h3 {
            margin-top: 0;
            color: #2c3e50;
            border-bottom: 2px solid #3498db;
            padding-bottom: 10px;
        }
        .metric {
            display: flex;
            justify-content: space-between;
            margin: 10px 0;
            padding: 5px 0;
            border-bottom: 1px solid #eee;
        }
        .metric:last-child {
            border-bottom: none;
        }
        .metric-label {
            font-weight: 500;
        }
        .metric-value {
            font-family: monospace;
            color: #2c3e50;
        }
        .progress-bar {
            width: 100%;
            height: 20px;
            background-color: #ecf0f1;
            border-radius: 10px;
            overflow: hidden;
            margin: 5px 0;
        }
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #3498db, #2ecc71);
            transition: width 0.3s ease;
        }
        .refresh-btn {
            background: #3498db;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 5px;
            cursor: pointer;
            font-size: 14px;
        }
        .refresh-btn:hover {
            background: #2980b9;
        }
        .loading {
            text-align: center;
            color: #7f8c8d;
            font-style: italic;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Machine Monitor Agent</h1>
            <div id="status" class="status">Carregando...</div>
            <button class="refresh-btn" onclick="refreshData()">Atualizar</button>
        </div>
        
        <div class="grid">
            <div class="card">
                <h3>Status do Agente</h3>
                <div id="agent-status" class="loading">Carregando...</div>
            </div>
            
            <div class="card">
                <h3>Sistema</h3>
                <div id="system-info" class="loading">Carregando...</div>
            </div>
            
            <div class="card">
                <h3>CPU</h3>
                <div id="cpu-info" class="loading">Carregando...</div>
            </div>
            
            <div class="card">
                <h3>Memória</h3>
                <div id="memory-info" class="loading">Carregando...</div>
            </div>
            
            <div class="card">
                <h3>Disco</h3>
                <div id="disk-info" class="loading">Carregando...</div>
            </div>
            
            <div class="card">
                <h3>Rede</h3>
                <div id="network-info" class="loading">Carregando...</div>
            </div>
        </div>
    </div>

    <script>
        function formatBytes(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        function formatDuration(seconds) {
            const days = Math.floor(seconds / 86400);
            const hours = Math.floor((seconds % 86400) / 3600);
            const minutes = Math.floor((seconds % 3600) / 60);
            
            if (days > 0) return days + 'd ' + hours + 'h';
            if (hours > 0) return hours + 'h ' + minutes + 'm';
            return minutes + 'm';
        }

        function createMetric(label, value) {
            return '<div class="metric"><span class="metric-label">' + label + '</span><span class="metric-value">' + value + '</span></div>';
        }

        function createProgressBar(percentage) {
            return '<div class="progress-bar"><div class="progress-fill" style="width: ' + percentage + '%"></div></div>';
        }

        async function loadStatus() {
            try {
                const response = await fetch('/api/status');
                const data = await response.json();
                
                const statusEl = document.getElementById('status');
                statusEl.textContent = data.state;
                statusEl.className = 'status ' + data.state.toLowerCase();
                
                const agentStatusEl = document.getElementById('agent-status');
                agentStatusEl.innerHTML = 
                    createMetric('Estado', data.state) +
                    createMetric('Uptime', formatDuration(data.uptime / 1000000000)) +
                    createMetric('Comandos Executados', data.commands_run) +
                    createMetric('Erros', data.errors) +
                    createMetric('Último Heartbeat', data.last_heartbeat ? new Date(data.last_heartbeat).toLocaleString() : 'Nunca') +
                    createMetric('Último Inventário', data.last_inventory ? new Date(data.last_inventory).toLocaleString() : 'Nunca');
            } catch (error) {
                console.error('Erro ao carregar status:', error);
            }
        }

        async function loadSystemInfo() {
            try {
                const response = await fetch('/api/system/fresh');
                const data = await response.json();
                
                const systemInfoEl = document.getElementById('system-info');
                systemInfoEl.innerHTML = 
                    createMetric('Sistema Operacional', data.os) +
                    createMetric('Plataforma', data.platform) +
                    createMetric('Hostname', data.hostname) +
                    createMetric('Uptime', formatDuration(data.uptime)) +
                    createMetric('Processos', data.procs) +
                    createMetric('Usuários Logados', data.users ? data.users.length : 0);
            } catch (error) {
                console.error('Erro ao carregar info do sistema:', error);
            }
        }

        async function loadHardwareInfo() {
            try {
                const response = await fetch('/api/hardware/fresh');
                const data = await response.json();
                
                // CPU
                const cpuInfoEl = document.getElementById('cpu-info');
                cpuInfoEl.innerHTML = 
                    createMetric('Modelo', data.cpu.model_name || 'N/A') +
                    createMetric('Cores', data.cpu.cores) +
                    createMetric('Threads', data.cpu.threads) +
                    createMetric('Frequência', data.cpu.frequency.toFixed(2) + ' MHz') +
                    createMetric('Uso', data.cpu.usage.toFixed(1) + '%') +
                    createProgressBar(data.cpu.usage);
                
                // Memória
                const memoryInfoEl = document.getElementById('memory-info');
                memoryInfoEl.innerHTML = 
                    createMetric('Total', formatBytes(data.memory.total)) +
                    createMetric('Usado', formatBytes(data.memory.used)) +
                    createMetric('Disponível', formatBytes(data.memory.available)) +
                    createMetric('Uso', data.memory.used_percent.toFixed(1) + '%') +
                    createProgressBar(data.memory.used_percent);
                
                // Disco
                const diskInfoEl = document.getElementById('disk-info');
                let diskHtml = '';
                data.disk.forEach(disk => {
                    diskHtml += '<div style="margin-bottom: 15px; padding-bottom: 15px; border-bottom: 1px solid #eee;">';
                    diskHtml += createMetric('Dispositivo', disk.device);
                    diskHtml += createMetric('Ponto de Montagem', disk.mountpoint);
                    diskHtml += createMetric('Tipo', disk.fstype);
                    diskHtml += createMetric('Total', formatBytes(disk.total));
                    diskHtml += createMetric('Usado', formatBytes(disk.used));
                    diskHtml += createMetric('Livre', formatBytes(disk.free));
                    diskHtml += createMetric('Uso', disk.used_percent.toFixed(1) + '%');
                    diskHtml += createProgressBar(disk.used_percent);
                    diskHtml += '</div>';
                });
                diskInfoEl.innerHTML = diskHtml;
                
                // Rede
                const networkInfoEl = document.getElementById('network-info');
                let networkHtml = '';
                data.network.forEach(net => {
                    networkHtml += '<div style="margin-bottom: 15px; padding-bottom: 15px; border-bottom: 1px solid #eee;">';
                    networkHtml += createMetric('Interface', net.name);
                    networkHtml += createMetric('MAC', net.hardware_addr || 'N/A');
                    networkHtml += createMetric('Endereços', net.addrs ? net.addrs.join(', ') : 'N/A');
                    networkHtml += createMetric('Bytes Enviados', formatBytes(net.bytes_sent));
                    networkHtml += createMetric('Bytes Recebidos', formatBytes(net.bytes_recv));
                    networkHtml += '</div>';
                });
                networkInfoEl.innerHTML = networkHtml;
                
            } catch (error) {
                console.error('Erro ao carregar info de hardware:', error);
            }
        }

        function refreshData() {
            loadStatus();
            loadSystemInfo();
            loadHardwareInfo();
        }

        // Carrega dados iniciais
        refreshData();
        
        // Atualiza automaticamente a cada 10 segundos
        setInterval(refreshData, 10000);
    </script>
</body>
</html>
`

	t, err := template.New("home").Parse(tmpl)
	if err != nil {
		http.Error(rw, "Erro no template", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(rw, nil); err != nil {
		log.Error().Err(err).Msg("Erro ao executar template")
	}
}

// handleAPIStatus trata a API de status
func (w *WebUI) handleAPIStatus(rw http.ResponseWriter, r *http.Request) {
	status := w.agent.GetStatus()

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(status)
}

// handleAPISystem trata a API de informações do sistema
func (w *WebUI) handleAPISystem(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	info, err := w.agent.CollectSystemInfo(ctx)
	if err != nil {
		http.Error(rw, "Erro ao coletar informações do sistema", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(info)
}

// handleAPISystemFresh trata a API de informações do sistema sem cache
func (w *WebUI) handleAPISystemFresh(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	info, err := w.agent.CollectSystemInfoFresh(ctx)
	if err != nil {
		http.Error(rw, "Erro ao coletar informações do sistema", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(info)
}

// handleAPIHardware trata a API de informações de hardware
func (w *WebUI) handleAPIHardware(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	info, err := w.agent.CollectHardwareInfo(ctx)
	if err != nil {
		http.Error(rw, "Erro ao coletar informações de hardware", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(info)
}

// handleAPIHardwareFresh trata a API de informações de hardware sem cache
func (w *WebUI) handleAPIHardwareFresh(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	info, err := w.agent.CollectHardwareInfoFresh(ctx)
	if err != nil {
		http.Error(rw, "Erro ao coletar informações de hardware", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(info)
}

// handleStatic trata arquivos estáticos
func (w *WebUI) handleStatic(rw http.ResponseWriter, r *http.Request) {
	http.NotFound(rw, r)
}
