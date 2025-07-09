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
		logger.websocket("Nova conexão WebSocket estabelecida");

		let machineId = null;

		// Configurar handlers
		ws.on("message", (data) => {
			try {
				const message = JSON.parse(data.toString());

				// Primeiro mensagem deve conter machine_id
				if (!machineId && message.machine_id) {
					machineId = message.machine_id;
					storage.setWebSocketConnection(machineId, ws);
					logger.websocket(
						`WebSocket registrado para máquina: ${machineId}`
					);
					return;
				}

				// Processar resultados de comandos
				if (message.id && (message.output || message.error)) {
					logger.websocket(`Resultado de comando recebido:`, {
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
			} catch (error) {
				logger.error(
					"Erro ao processar mensagem WebSocket:",
					error.message
				);
			}
		});

		ws.on("close", () => {
			if (machineId) {
				storage.removeWebSocketConnection(machineId);
				logger.websocket(
					`WebSocket desconectado para máquina: ${machineId}`
				);
			}
		});

		ws.on("error", (error) => {
			logger.error("Erro WebSocket:", error.message);
		});
	});

	logger.info("WebSocket Server configurado");
	return wss;
}

module.exports = {
	setupWebSocketServer,
};
