/**
 * Handler para WebSocket
 */

const WebSocket = require("ws");
const storage = require("../storage/memory");
const logger = require("../utils/logger");

/**
 * Configurar WebSocket Server
 */
function setupWebSocketServer(server) {
	const wss = new WebSocket.Server({ server });

	wss.on("connection", (ws, req) => {
		logger.websocket("🔌 Nova conexão WebSocket estabelecida");

		let machineId = null;

		// Configurar handlers
		ws.on("message", (data) => {
			try {
				const message = JSON.parse(data.toString());

				// Primeiro mensagem deve conter machine_id (registro)
				if (!machineId && message.machine_id) {
					machineId = message.machine_id;
					storage.setWebSocketConnection(machineId, ws);
					logger.websocket(
						`📝 WebSocket registrado para máquina: ${machineId}`
					);

					// Responder com confirmação
					ws.send(
						JSON.stringify({
							type: "registration_ack",
							id: `ack_${Date.now()}`,
							timestamp: new Date().toISOString(),
							data: {
								message: "Registro bem-sucedido",
								machine_id: machineId,
							},
						})
					);
					return;
				}

				// Processar mensagens baseadas no tipo
				switch (message.type) {
					case "command_result":
						handleCommandResult(message, machineId);
						break;
					case "ping":
						handlePing(ws, message);
						break;
					case "pong":
						handlePong(message, machineId);
						break;
					case "status_update":
						handleStatusUpdate(message, machineId);
						break;
					default:
						// Formato legado - processar resultados de comandos
						if (message.id && (message.output || message.error)) {
							handleLegacyCommandResult(message, machineId);
						} else {
							logger.websocket(
								`⚠️  Tipo de mensagem desconhecido: ${message.type}`
							);
						}
				}
			} catch (error) {
				logger.error(
					"❌ Erro ao processar mensagem WebSocket:",
					error.message
				);
			}
		});

		ws.on("close", () => {
			if (machineId) {
				storage.removeWebSocketConnection(machineId);
				logger.websocket(
					`🔌 WebSocket desconectado para máquina: ${machineId}`
				);
			}
		});

		ws.on("error", (error) => {
			logger.error("❌ Erro WebSocket:", error.message);
		});
	});

	logger.info("ℹ️  WebSocket Server configurado");
	return wss;
}

/**
 * Processar resultado de comando (formato do agente)
 */
function handleCommandResult(message, machineId) {
	const { id, data } = message;

	if (!data || !data.command_id) {
		logger.warning("⚠️  Resultado de comando sem command_id");
		return;
	}

	logger.websocket(`📥 Resultado de comando recebido:`, {
		id: data.id,
		command_id: data.command_id,
		machineId,
		status: data.status,
		hasOutput: !!data.output,
		hasError: !!data.error,
	});

	// Atualizar comando
	storage.updateCommand(data.command_id, {
		status: data.status,
		output: data.output,
		error: data.error,
		exit_code: data.exit_code,
		execution_time_ms: data.execution_time,
		completed_at: new Date().toISOString(),
	});
}

/**
 * Processar resultado de comando (formato legado)
 */
function handleLegacyCommandResult(message, machineId) {
	logger.websocket(`📥 Resultado de comando recebido (legado):`, {
		id: message.id,
		machineId,
		hasOutput: !!message.output,
		hasError: !!message.error,
	});

	storage.updateCommand(message.id, {
		output: message.output,
		error: message.error,
		status: message.error ? "failed" : "completed",
		completed_at: new Date().toISOString(),
	});
}

/**
 * Processar ping
 */
function handlePing(ws, message) {
	logger.websocket("🏓 Ping recebido, enviando pong");

	ws.send(
		JSON.stringify({
			type: "pong",
			id: message.id,
			timestamp: new Date().toISOString(),
		})
	);
}

/**
 * Processar pong
 */
function handlePong(message, machineId) {
	logger.websocket(`🏓 Pong recebido de ${machineId}`);
}

/**
 * Processar atualização de status
 */
function handleStatusUpdate(message, machineId) {
	logger.websocket(`📊 Status update recebido de ${machineId}`, message.data);

	if (message.data) {
		storage.setMachine(machineId, {
			status: message.data.status,
			last_status_update: new Date().toISOString(),
		});
	}
}

module.exports = {
	setupWebSocketServer,
};
