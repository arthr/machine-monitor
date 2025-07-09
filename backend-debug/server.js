const express = require("express");
const WebSocket = require("ws");
const cors = require("cors");
const path = require("path");

const app = express();
const PORT = 8080;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static("public"));

// Storage em memória (super simples)
const storage = {
	machines: new Map(),
	heartbeats: new Map(),
	inventories: new Map(),
	commands: new Map(),
	wsConnections: new Map(),
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
	if (token !== "Bearer dev-token-123") {
		return res.status(401).json({ error: "Token inválido" });
	}
	next();
}

// ==================== ROTAS HTTP ====================

// Página principal
app.get("/", (req, res) => {
	res.sendFile(path.join(__dirname, "public", "index.html"));
});

// Receber heartbeat
app.post("/heartbeat", auth, (req, res) => {
	const { machine_id, status = "online", timestamp } = req.body;

	log(`Heartbeat recebido de ${machine_id}`, req.body);

	// Salvar heartbeat
	if (!storage.heartbeats.has(machine_id)) {
		storage.heartbeats.set(machine_id, []);
	}
	storage.heartbeats.get(machine_id).push({
		status,
		timestamp: timestamp || new Date().toISOString(),
		received_at: new Date().toISOString(),
	});

	// Atualizar ou criar máquina
	storage.machines.set(machine_id, {
		id: machine_id,
		status,
		last_seen: new Date().toISOString(),
		hostname: req.body.hostname || "unknown",
	});

	res.json({ status: "ok", timestamp: new Date().toISOString() });
});

// Receber inventário
app.post("/inventory", auth, (req, res) => {
	const { machine_id } = req.body;

	log(`Inventário recebido de ${machine_id}`);
	log("Dados do inventário:", req.body);

	// Salvar inventário
	if (!storage.inventories.has(machine_id)) {
		storage.inventories.set(machine_id, []);
	}
	storage.inventories.get(machine_id).push({
		...req.body,
		received_at: new Date().toISOString(),
	});

	// Manter apenas os últimos 10 inventários
	const inventories = storage.inventories.get(machine_id);
	if (inventories.length > 10) {
		inventories.shift();
	}

	res.json({ status: "ok", timestamp: new Date().toISOString() });
});

// Listar máquinas
app.get("/machines", auth, (req, res) => {
	const machines = Array.from(storage.machines.values());
	res.json(machines);
});

// Dados de uma máquina específica
app.get("/machines/:id", auth, (req, res) => {
	const machineId = req.params.id;
	const machine = storage.machines.get(machineId);

	if (!machine) {
		return res.status(404).json({ error: "Máquina não encontrada" });
	}

	res.json({
		machine,
		heartbeats: storage.heartbeats.get(machineId) || [],
		inventories: storage.inventories.get(machineId) || [],
		connected: storage.wsConnections.has(machineId),
	});
});

// Enviar comando
app.post("/commands", auth, (req, res) => {
	const { machine_id, command, args = [] } = req.body;

	const commandId = `cmd_${Date.now()}`;
	const commandData = {
		id: commandId,
		machine_id,
		name: command,
		args,
		status: "pending",
		created_at: new Date().toISOString(),
	};

	storage.commands.set(commandId, commandData);

	// Enviar via WebSocket se conectado
	const wsConn = storage.wsConnections.get(machine_id);
	if (wsConn && wsConn.readyState === WebSocket.OPEN) {
		wsConn.send(
			JSON.stringify({
				id: commandId,
				name: command,
				args,
				timestamp: Date.now(),
			})
		);
		log(`Comando enviado via WebSocket para ${machine_id}:`, commandData);
	} else {
		log(`Máquina ${machine_id} não conectada via WebSocket`);
	}

	res.json(commandData);
});

// Status do comando
app.get("/commands/:id", auth, (req, res) => {
	const command = storage.commands.get(req.params.id);
	if (!command) {
		return res.status(404).json({ error: "Comando não encontrado" });
	}
	res.json(command);
});

// Debug - estatísticas
app.get("/debug/stats", (req, res) => {
	res.json({
		machines: storage.machines.size,
		websocket_connections: storage.wsConnections.size,
		total_commands: storage.commands.size,
		uptime: process.uptime(),
	});
});

// Debug - limpar dados
app.delete("/debug/clear", (req, res) => {
	storage.machines.clear();
	storage.heartbeats.clear();
	storage.inventories.clear();
	storage.commands.clear();
	log("Todos os dados foram limpos");
	res.json({ message: "Dados limpos" });
});

// ==================== WEBSOCKET ====================

const server = app.listen(PORT, () => {
	log(`Servidor rodando em http://localhost:${PORT}`);
});

const wss = new WebSocket.Server({ server });

wss.on("connection", (ws, req) => {
	log("Nova conexão WebSocket estabelecida");

	let machineId = null;

	ws.on("message", (data) => {
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
					command.status = message.error ? "failed" : "completed";
					command.completed_at = new Date().toISOString();
					storage.commands.set(message.id, command);
				}
			}
		} catch (error) {
			log("Erro ao processar mensagem WebSocket:", error.message);
		}
	});

	ws.on("close", () => {
		if (machineId) {
			storage.wsConnections.delete(machineId);
			log(`WebSocket desconectado para máquina: ${machineId}`);
		}
	});

	ws.on("error", (error) => {
		log("Erro WebSocket:", error.message);
	});
});

// Graceful shutdown
process.on("SIGTERM", () => {
	log("Parando servidor...");
	server.close(() => {
		log("Servidor parado");
		process.exit(0);
	});
});

process.on("SIGINT", () => {
	log("Parando servidor...");
	server.close(() => {
		log("Servidor parado");
		process.exit(0);
	});
});
