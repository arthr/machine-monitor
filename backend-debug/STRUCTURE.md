# Estrutura Modular do Backend

Este documento descreve a estrutura modular do backend após a refatoração.

## 📁 Estrutura de Arquivos

```
backend-debug/
├── server.js                 # Servidor principal (ponto de entrada)
├── package.json              # Dependências e scripts
├── package-lock.json         # Lockfile
├── .gitignore                # Arquivos ignorados pelo Git
├── README.md                 # Documentação geral
├── STRUCTURE.md              # Este arquivo
├── test-api.js               # Script de teste das APIs
├── config/
│   └── config.js             # Configurações centralizadas
├── utils/
│   └── logger.js             # Utilitário de logging
├── middleware/
│   └── auth.js               # Middleware de autenticação
├── storage/
│   └── memory.js             # Storage em memória
├── routes/
│   ├── heartbeat.js          # Rotas de heartbeat
│   ├── inventory.js          # Rotas de inventário
│   ├── machines.js           # Rotas de máquinas
│   ├── commands.js           # Rotas de comandos
│   └── debug.js              # Rotas de debug
├── websocket/
│   └── handler.js            # Handler do WebSocket
└── public/
    └── index.html            # Interface web
```

## 🏗️ Separação de Responsabilidades

### 1. **server.js** - Servidor Principal
- Inicialização do Express
- Configuração de middleware
- Registro de rotas
- Configuração do WebSocket
- Graceful shutdown

### 2. **config/config.js** - Configurações
- Configurações do servidor
- Configurações de autenticação
- Configurações de storage
- Configurações de WebSocket
- Comandos permitidos por plataforma

### 3. **utils/logger.js** - Logging
- Logs com timestamp
- Diferentes tipos de log (info, success, warning, error)
- Logs específicos por funcionalidade (websocket, command, heartbeat)
- Controle de verbosidade

### 4. **middleware/auth.js** - Autenticação
- Middleware de autenticação
- Validação de tokens
- Controle de acesso

### 5. **storage/memory.js** - Storage
- Gerenciamento de dados em memória
- Operações CRUD para máquinas
- Gerenciamento de heartbeats
- Gerenciamento de inventários
- Gerenciamento de comandos
- Controle de conexões WebSocket

### 6. **routes/** - Rotas por Funcionalidade

#### **heartbeat.js**
- `POST /heartbeat` - Receber heartbeat

#### **inventory.js**
- `POST /inventory` - Receber inventário

#### **machines.js**
- `GET /machines` - Listar máquinas
- `GET /machines/:id` - Detalhes de máquina

#### **commands.js**
- `POST /commands` - Enviar comando
- `GET /commands/:id` - Status do comando
- `GET /commands` - Listar comandos

#### **debug.js**
- `GET /debug/stats` - Estatísticas
- `DELETE /debug/clear` - Limpar dados
- `GET /debug/info` - Informações do servidor

### 7. **websocket/handler.js** - WebSocket
- Configuração do WebSocket Server
- Gerenciamento de conexões
- Processamento de mensagens
- Controle de comandos em tempo real

## 🔄 Fluxo de Dados

```
Cliente → Middleware → Rota → Storage → Resposta
                ↓
         WebSocket Handler ← Storage
```

## 📊 Benefícios da Refatoração

### ✅ Manutenibilidade
- Código organizado por responsabilidade
- Fácil localização de funcionalidades
- Redução de acoplamento

### ✅ Escalabilidade
- Fácil adição de novas rotas
- Separação clara de concerns
- Reutilização de componentes

### ✅ Testabilidade
- Módulos independentes
- Fácil mock de dependências
- Testes unitários por módulo

### ✅ Legibilidade
- Arquivos menores e focados
- Estrutura clara e previsível
- Documentação por módulo

## 🛠️ Como Usar

### Executar o servidor:
```bash
npm start
```

### Estrutura de imports:
```javascript
// No server.js
const config = require('./config/config');
const logger = require('./utils/logger');
const { authenticate } = require('./middleware/auth');
const storage = require('./storage/memory');

// Nas rotas
const router = require('express').Router();
const storage = require('../storage/memory');
const logger = require('../utils/logger');
const { authenticate } = require('../middleware/auth');
```

## 🔧 Configurações

### Variáveis de Ambiente
```bash
PORT=8080
HOST=localhost
AUTH_TOKEN=dev-token-123
DEBUG=true
LOG_LEVEL=info
```

### Configuração Custom
Edite `config/config.js` para ajustar:
- Porta do servidor
- Token de autenticação
- Limites de storage
- Comandos permitidos

## 📝 Próximos Passos

1. **Testes unitários** para cada módulo
2. **Validação de dados** nas rotas
3. **Rate limiting** para APIs
4. **Persistência** opcional (Redis/MongoDB)
5. **Metrics** e monitoramento
6. **Documentação** da API (Swagger)

## 🐛 Troubleshooting

### Erro de módulo não encontrado
```bash
npm install
```

### Erro de autenticação
Verifique o token em `config/config.js`

### WebSocket não conecta
Verifique se o servidor está rodando na porta correta

### Dados não aparecem
Verifique os logs no console do servidor

## 📚 Referências

- [Express.js](https://expressjs.com/)
- [WebSocket](https://www.npmjs.com/package/ws)
- [Node.js Modules](https://nodejs.org/api/modules.html)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) 