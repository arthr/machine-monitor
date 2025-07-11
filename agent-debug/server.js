const express = require('express');
const WebSocket = require('ws');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = 3000;

// Middleware
app.use(cors());
app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true }));
app.use(express.static(path.join(__dirname, 'public')));

// Storage para mensagens
const messages = [];
const machines = new Map();

// Função para adicionar mensagem ao log
function addMessage(type, endpoint, data, headers = {}) {
  const message = {
    id: Date.now() + Math.random(),
    timestamp: new Date().toISOString(),
    type,
    endpoint,
    data,
    headers,
    size: JSON.stringify(data).length
  };
  
  messages.unshift(message); // Adiciona no início
  
  // Manter apenas as últimas 1000 mensagens
  if (messages.length > 1000) {
    messages.splice(1000);
  }
  
  console.log(`[${message.timestamp}] ${type} ${endpoint} - ${message.size} bytes`);
  
  // Broadcast para clientes WebSocket conectados
  broadcastToClients({
    type: 'new_message',
    message
  });
  
  return message;
}

// Broadcast para clientes WebSocket
function broadcastToClients(data) {
  if (wss) {
    wss.clients.forEach(client => {
      if (client.readyState === WebSocket.OPEN) {
        client.send(JSON.stringify(data));
      }
    });
  }
}

// Routes HTTP que o agente vai usar

// Ping - teste de conectividade
app.get('/api/ping', (req, res) => {
  addMessage('GET', '/api/ping', { status: 'pong' }, req.headers);
  res.json({ status: 'pong', timestamp: new Date().toISOString() });
});

// Registro de máquina
app.post('/api/machines/register', (req, res) => {
  const machineData = req.body;
  addMessage('POST', '/api/machines/register', machineData, req.headers);
  
  // Armazena informações da máquina
  if (machineData.machine_id) {
    machines.set(machineData.machine_id, {
      ...machineData,
      registered_at: new Date().toISOString(),
      last_seen: new Date().toISOString()
    });
  }
  
  res.json({
    success: true,
    machine_id: machineData.machine_id,
    message: 'Máquina registrada com sucesso'
  });
});

// Heartbeat
app.post('/api/machines/:machineId/heartbeat', (req, res) => {
  const { machineId } = req.params;
  const heartbeatData = req.body;
  
  addMessage('POST', `/api/machines/${machineId}/heartbeat`, heartbeatData, req.headers);
  
  // Atualiza last_seen da máquina
  if (machines.has(machineId)) {
    const machine = machines.get(machineId);
    machine.last_seen = new Date().toISOString();
    machine.last_heartbeat = heartbeatData;
    machines.set(machineId, machine);
  }
  
  res.json({
    success: true,
    timestamp: new Date().toISOString()
  });
});

// Inventário
app.post('/api/machines/:machineId/inventory', (req, res) => {
  const { machineId } = req.params;
  const inventoryData = req.body;
  
  addMessage('POST', `/api/machines/${machineId}/inventory`, inventoryData, req.headers);
  
  // Atualiza inventário da máquina
  if (machines.has(machineId)) {
    const machine = machines.get(machineId);
    machine.last_inventory = inventoryData;
    machine.inventory_updated_at = new Date().toISOString();
    machines.set(machineId, machine);
  }
  
  res.json({
    success: true,
    timestamp: new Date().toISOString()
  });
});

// Resultado de comando
app.post('/api/machines/:machineId/commands/result', (req, res) => {
  const { machineId } = req.params;
  const commandResult = req.body;
  
  addMessage('POST', `/api/machines/${machineId}/commands/result`, commandResult, req.headers);
  
  res.json({
    success: true,
    timestamp: new Date().toISOString()
  });
});

// API para o frontend de debug

// Listar todas as mensagens
app.get('/debug/messages', (req, res) => {
  const limit = parseInt(req.query.limit) || 100;
  const offset = parseInt(req.query.offset) || 0;
  
  res.json({
    messages: messages.slice(offset, offset + limit),
    total: messages.length
  });
});

// Limpar mensagens
app.delete('/debug/messages', (req, res) => {
  messages.length = 0;
  broadcastToClients({ type: 'messages_cleared' });
  res.json({ success: true, message: 'Mensagens limpas' });
});

// Listar máquinas registradas
app.get('/debug/machines', (req, res) => {
  const machinesArray = Array.from(machines.entries()).map(([id, data]) => ({
    id,
    ...data
  }));
  
  res.json({
    machines: machinesArray,
    count: machines.size
  });
});

// Obter detalhes de uma máquina específica
app.get('/debug/machines/:machineId', (req, res) => {
  const { machineId } = req.params;
  
  if (machines.has(machineId)) {
    res.json(machines.get(machineId));
  } else {
    res.status(404).json({ error: 'Máquina não encontrada' });
  }
});

// Estatísticas
app.get('/debug/stats', (req, res) => {
  const stats = {
    total_messages: messages.length,
    total_machines: machines.size,
    message_types: {},
    endpoints: {},
    last_24h: 0
  };
  
  const now = new Date();
  const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000);
  
  messages.forEach(msg => {
    // Contar por tipo
    stats.message_types[msg.type] = (stats.message_types[msg.type] || 0) + 1;
    
    // Contar por endpoint
    stats.endpoints[msg.endpoint] = (stats.endpoints[msg.endpoint] || 0) + 1;
    
    // Contar últimas 24h
    if (new Date(msg.timestamp) > yesterday) {
      stats.last_24h++;
    }
  });
  
  res.json(stats);
});

// Servidor HTTP
const server = app.listen(PORT, () => {
  console.log(`🚀 Agent Debug Server rodando em http://localhost:${PORT}`);
  console.log(`📊 Interface de debug: http://localhost:${PORT}`);
  console.log(`🔌 WebSocket em ws://localhost:${PORT}`);
});

// WebSocket Server para updates em tempo real
const wss = new WebSocket.Server({ server });

wss.on('connection', (ws) => {
  console.log('📱 Cliente conectado via WebSocket');
  
  // Enviar estatísticas iniciais
  ws.send(JSON.stringify({
    type: 'initial_data',
    messages: messages.slice(0, 50), // Últimas 50 mensagens
    machines: Array.from(machines.entries()).map(([id, data]) => ({ id, ...data }))
  }));
  
  ws.on('message', (data) => {
    try {
      const message = JSON.parse(data);
      console.log('📨 Mensagem do cliente:', message);
      
      // Aqui podemos adicionar comandos do cliente se necessário
      if (message.type === 'get_messages') {
        ws.send(JSON.stringify({
          type: 'messages',
          messages: messages.slice(0, message.limit || 100)
        }));
      }
    } catch (error) {
      console.error('❌ Erro ao processar mensagem WebSocket:', error);
    }
  });
  
  ws.on('close', () => {
    console.log('📱 Cliente desconectado');
  });
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('🛑 Parando servidor...');
  server.close(() => {
    console.log('✅ Servidor parado');
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  console.log('🛑 Parando servidor...');
  server.close(() => {
    console.log('✅ Servidor parado');
    process.exit(0);
  });
}); 