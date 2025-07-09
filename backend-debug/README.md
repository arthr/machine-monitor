# Backend de Debug do Agente

Backend simples para desenvolvimento e debug do agente de monitoramento.

## 🚀 Instalação e Uso

### Pré-requisitos
- Node.js 16+ instalado
- npm ou yarn

### Instalação
```bash
cd backend-debug
npm install
```

### Executar
```bash
npm start
# ou para desenvolvimento com auto-reload:
npm run dev
```

O servidor estará disponível em: http://localhost:8080

## 📋 Funcionalidades

### Interface Web
- **Dashboard**: http://localhost:8080
- Visualização em tempo real das máquinas conectadas
- Envio de comandos para máquinas específicas
- Logs em tempo real
- Estatísticas do servidor

### APIs REST

#### Autenticação
Todas as APIs requerem o header:
```
Authorization: Bearer dev-token-123
```

#### Endpoints

**Heartbeat**
```bash
POST /heartbeat
{
    "machine_id": "mac-001",
    "status": "online",
    "hostname": "MacBook-Pro"
}
```

**Inventário**
```bash
POST /inventory
{
    "machine_id": "mac-001",
    "hardware": {...},
    "software": {...}
}
```

**Listar Máquinas**
```bash
GET /machines
```

**Detalhes de Máquina**
```bash
GET /machines/{id}
```

**Enviar Comando**
```bash
POST /commands
{
    "machine_id": "mac-001",
    "command": "ps",
    "args": ["aux"]
}
```

**Status do Comando**
```bash
GET /commands/{id}
```

**Estatísticas**
```bash
GET /debug/stats
```

**Limpar Dados**
```bash
DELETE /debug/clear
```

### WebSocket

**Endpoint**: `ws://localhost:8080`

**Registro da Máquina**:
```json
{
    "machine_id": "mac-001"
}
```

**Resultado de Comando**:
```json
{
    "id": "cmd_1234567890",
    "output": "resultado do comando",
    "error": null
}
```

## 🧪 Testes

### Teste de Heartbeat
```bash
curl -X POST http://localhost:8080/heartbeat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev-token-123" \
  -d '{"machine_id": "test-001", "status": "online", "hostname": "Test Machine"}'
```

### Teste de Comando
```bash
curl -X POST http://localhost:8080/commands \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev-token-123" \
  -d '{"machine_id": "test-001", "command": "uptime"}'
```

### WebSocket Test (JavaScript)
```javascript
const ws = new WebSocket('ws://localhost:8080');

ws.onopen = () => {
    // Registrar máquina
    ws.send(JSON.stringify({
        machine_id: 'test-001'
    }));
};

ws.onmessage = (event) => {
    const command = JSON.parse(event.data);
    console.log('Comando recebido:', command);
    
    // Simular execução
    setTimeout(() => {
        ws.send(JSON.stringify({
            id: command.id,
            output: 'resultado simulado',
            error: null
        }));
    }, 1000);
};
```

## 🔧 Configuração

### Variáveis de Ambiente
```bash
PORT=8080              # Porta do servidor
AUTH_TOKEN=dev-token-123   # Token de autenticação
```

### Comandos Disponíveis na Interface
- `ps aux` - Lista processos
- `system_profiler SPHardwareDataType` - Info de hardware (macOS)
- `launchctl list` - Serviços em execução (macOS)
- `uptime` - Tempo ligado
- `whoami` - Usuário atual
- `uname -a` - Informações do sistema
- `df -h` - Espaço em disco
- `top -l 1` - CPU/RAM atual
- `netstat -an` - Conexões de rede

## 📊 Estrutura dos Dados

### Máquina
```json
{
    "id": "mac-001",
    "status": "online",
    "hostname": "MacBook-Pro",
    "last_seen": "2024-01-10T10:30:00Z"
}
```

### Comando
```json
{
    "id": "cmd_1234567890",
    "machine_id": "mac-001",
    "name": "ps",
    "args": ["aux"],
    "status": "completed",
    "output": "resultado...",
    "created_at": "2024-01-10T10:30:00Z",
    "completed_at": "2024-01-10T10:30:05Z"
}
```

### Heartbeat
```json
{
    "status": "online",
    "timestamp": "2024-01-10T10:30:00Z",
    "received_at": "2024-01-10T10:30:01Z"
}
```

## 🎯 Próximos Passos

1. **Executar o backend**: `npm start`
2. **Abrir interface**: http://localhost:8080
3. **Conectar agente**: Implementar cliente Go que se conecte às APIs
4. **Testar comandos**: Usar a interface para enviar comandos de teste

## 🐛 Debug

### Logs
- Todos os logs aparecem no console do servidor
- Logs também disponíveis na interface web
- Formato: `[ISO_TIMESTAMP] mensagem`

### Problemas Comuns
- **Erro de autenticação**: Verificar token `dev-token-123`
- **WebSocket não conecta**: Verificar se o servidor está rodando
- **Comandos não executam**: Verificar se o agente está conectado via WebSocket

## 📝 Notas

- **Dados em memória**: Todos os dados são perdidos quando o servidor para
- **Sem persistência**: Ideal para desenvolvimento/debug
- **Segurança mínima**: Não usar em produção
- **Auto-reload**: Use `npm run dev` para desenvolvimento

Este backend é especificamente para desenvolvimento e debug. Para produção, implementar persistência, autenticação robusta e outras funcionalidades de segurança. 