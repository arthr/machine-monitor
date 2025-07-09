🔧 Backend Simples para Debug - Node.js
*Servidor básico para testar e depurar o agente*

## 🎯 Objetivo

Criar um **servidor simples em Node.js** apenas para:
- ✅ Receber dados do agente (heartbeat, inventário)
- ✅ Enviar comandos via WebSocket
- ✅ Ver os dados em tempo real
- ✅ Logs para debug

**Tempo estimado**: 2-3 horas

⸻

## 📁 Estrutura Minimalista

```text
backend-debug/
├── server.js                       # Tudo em um arquivo
├── public/
│   └── index.html                  # Interface simples
├── package.json
└── README.md
```

⸻

## 🔧 Implementação Ultra-Simples

### **1. Package.json**

```json
{
  "name": "agent-debug-backend",
  "version": "1.0.0",
  "description": "Backend simples para debug do agente",
  "main": "server.js",
  "scripts": {
    "start": "node server.js",
    "dev": "nodemon server.js"
  },
  "dependencies": {
    "express": "^4.18.2",
    "ws": "^8.14.2",
    "cors": "^2.8.5"
  },
  "devDependencies": {
    "nodemon": "^3.0.1"
  }
}
```

### **2. Server.js - Tudo em um arquivo**

```javascript
const express = require('express');
const WebSocket = require('ws');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = 8080;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static('public'));

// Storage em memória (super simples)
const storage = {
    machines: new Map(),
    heartbeats: new Map(),
    inventories: new Map(),
    commands: new Map(),
    wsConnections: new Map()
};

// Função para log com timestamp
function log(message, data = null) {
    const timestamp = new Date().toISOString();
    console.log(`[${timestamp}] ${message}`);
    if (data) {
        console.log(JSON.stringify(data, null, 2));
    }
}

// Middleware de autenticação simples
function auth(req, res, next) {
    const token = req.headers.authorization;
    if (token !== 'Bearer dev-token-123') {
        return res.status(401).json({ error: 'Token inválido' });
    }
    next();
}

// ==================== ROTAS HTTP ====================

// Página principal
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'public', 'index.html'));
});

// Receber heartbeat
app.post('/heartbeat', auth, (req, res) => {
    const { machine_id, status = 'online', timestamp } = req.body;
    
    log(`Heartbeat recebido de ${machine_id}`, req.body);
    
    // Salvar heartbeat
    if (!storage.heartbeats.has(machine_id)) {
        storage.heartbeats.set(machine_id, []);
    }
    storage.heartbeats.get(machine_id).push({
        status,
        timestamp: timestamp || new Date().toISOString(),
        received_at: new Date().toISOString()
    });
    
    // Atualizar ou criar máquina
    storage.machines.set(machine_id, {
        id: machine_id,
        status,
        last_seen: new Date().toISOString(),
        hostname: req.body.hostname || 'unknown'
    });
    
    res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// Receber inventário
app.post('/inventory', auth, (req, res) => {
    const { machine_id } = req.body;
    
    log(`Inventário recebido de ${machine_id}`);
    log('Dados do inventário:', req.body);
    
    // Salvar inventário
    if (!storage.inventories.has(machine_id)) {
        storage.inventories.set(machine_id, []);
    }
    storage.inventories.get(machine_id).push({
        ...req.body,
        received_at: new Date().toISOString()
    });
    
    // Manter apenas os últimos 10 inventários
    const inventories = storage.inventories.get(machine_id);
    if (inventories.length > 10) {
        inventories.shift();
    }
    
    res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// Listar máquinas
app.get('/machines', auth, (req, res) => {
    const machines = Array.from(storage.machines.values());
    res.json(machines);
});

// Dados de uma máquina específica
app.get('/machines/:id', auth, (req, res) => {
    const machineId = req.params.id;
    const machine = storage.machines.get(machineId);
    
    if (!machine) {
        return res.status(404).json({ error: 'Máquina não encontrada' });
    }
    
    res.json({
        machine,
        heartbeats: storage.heartbeats.get(machineId) || [],
        inventories: storage.inventories.get(machineId) || [],
        connected: storage.wsConnections.has(machineId)
    });
});

// Enviar comando
app.post('/commands', auth, (req, res) => {
    const { machine_id, command, args = [] } = req.body;
    
    const commandId = `cmd_${Date.now()}`;
    const commandData = {
        id: commandId,
        machine_id,
        name: command,
        args,
        status: 'pending',
        created_at: new Date().toISOString()
    };
    
    storage.commands.set(commandId, commandData);
    
    // Enviar via WebSocket se conectado
    const wsConn = storage.wsConnections.get(machine_id);
    if (wsConn && wsConn.readyState === WebSocket.OPEN) {
        wsConn.send(JSON.stringify({
            id: commandId,
            name: command,
            args,
            timestamp: Date.now()
        }));
        log(`Comando enviado via WebSocket para ${machine_id}:`, commandData);
    } else {
        log(`Máquina ${machine_id} não conectada via WebSocket`);
    }
    
    res.json(commandData);
});

// Status do comando
app.get('/commands/:id', auth, (req, res) => {
    const command = storage.commands.get(req.params.id);
    if (!command) {
        return res.status(404).json({ error: 'Comando não encontrado' });
    }
    res.json(command);
});

// Debug - estatísticas
app.get('/debug/stats', (req, res) => {
    res.json({
        machines: storage.machines.size,
        websocket_connections: storage.wsConnections.size,
        total_commands: storage.commands.size,
        uptime: process.uptime()
    });
});

// Debug - limpar dados
app.delete('/debug/clear', (req, res) => {
    storage.machines.clear();
    storage.heartbeats.clear();
    storage.inventories.clear();
    storage.commands.clear();
    log('Todos os dados foram limpos');
    res.json({ message: 'Dados limpos' });
});

// ==================== WEBSOCKET ====================

const server = app.listen(PORT, () => {
    log(`Servidor rodando em http://localhost:${PORT}`);
});

const wss = new WebSocket.Server({ server });

wss.on('connection', (ws, req) => {
    log('Nova conexão WebSocket estabelecida');
    
    let machineId = null;
    
    ws.on('message', (data) => {
        try {
            const message = JSON.parse(data.toString());
            
            // Primeiro mensagem deve conter machine_id
            if (!machineId && message.machine_id) {
                machineId = message.machine_id;
                storage.wsConnections.set(machineId, ws);
                log(`WebSocket registrado para máquina: ${machineId}`);
                return;
            }
            
            // Processar resultados de comandos
            if (message.id && (message.output || message.error)) {
                log(`Resultado de comando recebido:`, message);
                
                const command = storage.commands.get(message.id);
                if (command) {
                    command.output = message.output;
                    command.error = message.error;
                    command.status = message.error ? 'failed' : 'completed';
                    command.completed_at = new Date().toISOString();
                    storage.commands.set(message.id, command);
                }
            }
            
        } catch (error) {
            log('Erro ao processar mensagem WebSocket:', error.message);
        }
    });
    
    ws.on('close', () => {
        if (machineId) {
            storage.wsConnections.delete(machineId);
            log(`WebSocket desconectado para máquina: ${machineId}`);
        }
    });
    
    ws.on('error', (error) => {
        log('Erro WebSocket:', error.message);
    });
});

// Graceful shutdown
process.on('SIGTERM', () => {
    log('Parando servidor...');
    server.close(() => {
        log('Servidor parado');
        process.exit(0);
    });
});
```

### **3. Interface Web (public/index.html)**

```html
<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Debug do Agente</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .card {
            background: white;
            padding: 20px;
            margin: 10px 0;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .machine {
            border-left: 4px solid #28a745;
            margin: 10px 0;
        }
        .machine.offline {
            border-left-color: #dc3545;
        }
        .btn {
            background: #007bff;
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 4px;
            cursor: pointer;
            margin: 5px;
        }
        .btn:hover {
            background: #0056b3;
        }
        .btn-danger {
            background: #dc3545;
        }
        .btn-danger:hover {
            background: #c82333;
        }
        .logs {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 4px;
            font-family: monospace;
            font-size: 12px;
            max-height: 300px;
            overflow-y: auto;
        }
        .stats {
            display: flex;
            gap: 20px;
            margin: 20px 0;
        }
        .stat {
            background: #e9ecef;
            padding: 10px;
            border-radius: 4px;
            text-align: center;
            flex: 1;
        }
        .command-result {
            background: #f8f9fa;
            padding: 10px;
            margin: 10px 0;
            border-radius: 4px;
            border-left: 3px solid #17a2b8;
        }
        pre {
            white-space: pre-wrap;
            word-wrap: break-word;
        }
        .status-online { color: #28a745; font-weight: bold; }
        .status-offline { color: #dc3545; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔧 Debug do Agente</h1>
        
        <div class="stats">
            <div class="stat">
                <div id="machine-count">0</div>
                <small>Máquinas</small>
            </div>
            <div class="stat">
                <div id="connection-count">0</div>
                <small>Conexões WS</small>
            </div>
            <div class="stat">
                <div id="command-count">0</div>
                <small>Comandos</small>
            </div>
            <div class="stat">
                <div id="uptime">0</div>
                <small>Uptime</small>
            </div>
        </div>
        
        <div class="card">
            <h2>Controles</h2>
            <button class="btn" onclick="refreshData()">🔄 Atualizar</button>
            <button class="btn btn-danger" onclick="clearData()">🗑️ Limpar Dados</button>
            <button class="btn" onclick="toggleLogs()">📋 Toggle Logs</button>
        </div>
        
        <div class="card">
            <h2>Máquinas Conectadas</h2>
            <div id="machines">
                <p>Carregando...</p>
            </div>
        </div>
        
        <div class="card" id="logs-container" style="display: none;">
            <h2>Logs em Tempo Real</h2>
            <div class="logs" id="logs">
                Logs aparecerão aqui...
            </div>
        </div>
    </div>

    <script>
        const AUTH_TOKEN = 'Bearer dev-token-123';
        let logsVisible = false;
        
        // Atualizar dados
        async function refreshData() {
            try {
                // Estatísticas
                const statsResponse = await fetch('/debug/stats');
                const stats = await statsResponse.json();
                
                document.getElementById('machine-count').textContent = stats.machines;
                document.getElementById('connection-count').textContent = stats.websocket_connections;
                document.getElementById('command-count').textContent = stats.total_commands;
                document.getElementById('uptime').textContent = Math.floor(stats.uptime) + 's';
                
                // Máquinas
                const machinesResponse = await fetch('/machines', {
                    headers: { 'Authorization': AUTH_TOKEN }
                });
                const machines = await machinesResponse.json();
                
                const machinesDiv = document.getElementById('machines');
                
                if (machines.length === 0) {
                    machinesDiv.innerHTML = '<p>Nenhuma máquina conectada</p>';
                    return;
                }
                
                machinesDiv.innerHTML = machines.map(machine => `
                    <div class="machine ${machine.status === 'online' ? 'online' : 'offline'}">
                        <h3>${machine.hostname || machine.id}</h3>
                        <p>
                            <strong>ID:</strong> ${machine.id} |
                            <strong>Status:</strong> <span class="status-${machine.status}">${machine.status}</span> |
                            <strong>Última atividade:</strong> ${new Date(machine.last_seen).toLocaleString()}
                        </p>
                        <div>
                            <select id="cmd-${machine.id}">
                                <option value="ps aux">ps aux</option>
                                <option value="system_profiler SPHardwareDataType">system_profiler SPHardwareDataType</option>
                                <option value="uptime">uptime</option>
                                <option value="whoami">whoami</option>
                                <option value="uname -a">uname -a</option>
                                <option value="df -h">df -h</option>
                            </select>
                            <button class="btn" onclick="sendCommand('${machine.id}')">Enviar Comando</button>
                            <button class="btn" onclick="viewMachine('${machine.id}')">Ver Detalhes</button>
                        </div>
                        <div id="result-${machine.id}"></div>
                    </div>
                `).join('');
                
            } catch (error) {
                console.error('Erro ao atualizar dados:', error);
            }
        }
        
        // Enviar comando
        async function sendCommand(machineId) {
            const select = document.getElementById(`cmd-${machineId}`);
            const command = select.value;
            const [cmd, ...args] = command.split(' ');
            
            try {
                const response = await fetch('/commands', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': AUTH_TOKEN
                    },
                    body: JSON.stringify({
                        machine_id: machineId,
                        command: cmd,
                        args: args
                    })
                });
                
                const result = await response.json();
                
                const resultDiv = document.getElementById(`result-${machineId}`);
                resultDiv.innerHTML = `
                    <div class="command-result">
                        <strong>Comando enviado:</strong> ${command}<br>
                        <strong>ID:</strong> ${result.id}<br>
                        <strong>Status:</strong> ${result.status}
                    </div>
                `;
                
                // Verificar resultado
                setTimeout(() => checkCommandResult(result.id, machineId), 2000);
                
            } catch (error) {
                console.error('Erro ao enviar comando:', error);
            }
        }
        
        // Verificar resultado do comando
        async function checkCommandResult(commandId, machineId) {
            try {
                const response = await fetch(`/commands/${commandId}`, {
                    headers: { 'Authorization': AUTH_TOKEN }
                });
                const command = await response.json();
                
                const resultDiv = document.getElementById(`result-${machineId}`);
                
                if (command.status === 'completed' || command.status === 'failed') {
                    resultDiv.innerHTML = `
                        <div class="command-result">
                            <strong>Comando:</strong> ${command.name}<br>
                            <strong>Status:</strong> ${command.status}<br>
                            <strong>Resultado:</strong><br>
                            <pre>${command.output || command.error || 'Sem saída'}</pre>
                        </div>
                    `;
                } else {
                    setTimeout(() => checkCommandResult(commandId, machineId), 1000);
                }
            } catch (error) {
                console.error('Erro ao verificar comando:', error);
            }
        }
        
        // Ver detalhes da máquina
        async function viewMachine(machineId) {
            try {
                const response = await fetch(`/machines/${machineId}`, {
                    headers: { 'Authorization': AUTH_TOKEN }
                });
                const data = await response.json();
                
                alert(`Detalhes da máquina:\n\n${JSON.stringify(data, null, 2)}`);
            } catch (error) {
                console.error('Erro ao buscar detalhes:', error);
            }
        }
        
        // Limpar dados
        async function clearData() {
            if (!confirm('Limpar todos os dados?')) return;
            
            try {
                await fetch('/debug/clear', { method: 'DELETE' });
                refreshData();
            } catch (error) {
                console.error('Erro ao limpar dados:', error);
            }
        }
        
        // Toggle logs
        function toggleLogs() {
            logsVisible = !logsVisible;
            const container = document.getElementById('logs-container');
            container.style.display = logsVisible ? 'block' : 'none';
        }
        
        // Atualizar automaticamente
        refreshData();
        setInterval(refreshData, 5000);
    </script>
</body>
</html>
```

### **4. README.md**

```markdown
# Backend Debug - Agente

Servidor Node.js simples para depurar o agente.

## Como usar

1. **Instalar dependências:**
   ```bash
   npm install
   ```

2. **Executar:**
   ```bash
   npm start
   # ou para desenvolvimento:
   npm run dev
   ```

3. **Acessar:**
   - Interface: http://localhost:8080
   - API: http://localhost:8080/machines

## Endpoints

- `POST /heartbeat` - Receber heartbeat do agente
- `POST /inventory` - Receber inventário do agente  
- `POST /commands` - Enviar comando para agente
- `GET /machines` - Listar máquinas
- `WebSocket /ws` - Comunicação em tempo real

## Autenticação

Token: `Bearer dev-token-123`

## Teste rápido

```bash
# Simular heartbeat
curl -X POST http://localhost:8080/heartbeat \
  -H "Authorization: Bearer dev-token-123" \
  -H "Content-Type: application/json" \
  -d '{"machine_id":"test","status":"online"}'
```
```

⸻

## 🚀 Como Executar

### **Instalação Ultra-Rápida**
```bash
# Criar projeto
mkdir backend-debug
cd backend-debug

# Criar package.json
npm init -y

# Instalar dependências
npm install express ws cors
npm install -D nodemon

# Copiar os arquivos acima
# Executar
npm start
```

### **Uso**
1. **Executar backend**: `npm start`
2. **Abrir navegador**: http://localhost:8080
3. **Executar agente POC**
4. **Ver dados em tempo real** na interface

⸻

## 🎯 Funcionalidades

### **✅ Essenciais**
- Receber heartbeat e inventário do agente
- WebSocket para comandos em tempo real
- Interface web para visualizar dados
- Logs no console para debug

### **✅ Debug**
- Ver dados JSON completos
- Enviar comandos de teste
- Limpar dados facilmente
- Estatísticas em tempo real

### **✅ Simples**
- Tudo em um arquivo `server.js`
- Interface web básica mas funcional
- Zero configuração complexa
- Pronto para usar em minutos

Este backend é **perfeito** para desenvolver e depurar o agente rapidamente! 