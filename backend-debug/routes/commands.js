/**
 * Rotas para comandos
 */

const express = require("express");
const router = express.Router();
const WebSocket = require("ws");
const storage = require("../storage/memory");
const logger = require("../utils/logger");
const { authenticate } = require("../middleware/auth");

/**
 * Enviar comando para uma máquina
 */
router.post("/", authenticate, (req, res) => {
	const { machine_id, command, args = [] } = req.body;

	if (!machine_id || !command) {
		logger.warning("Comando recebido sem machine_id ou command");
		return res.status(400).json({
			error: "machine_id e command são obrigatórios",
		});
	}

	const commandId = `cmd_${Date.now()}`;
	const commandData = {
		id: commandId,
		machine_id,
		name: command,
		args,
		status: "pending",
	};

	storage.addCommand(commandId, commandData);

	// Enviar via WebSocket se conectado
	const wsConn = storage.getWebSocketConnection(machine_id);
	if (wsConn && wsConn.readyState === WebSocket.OPEN) {
		wsConn.send(
			JSON.stringify({
				id: commandId,
				name: command,
				args,
				timestamp: Date.now(),
			})
		);

		logger.command(`Comando enviado via WebSocket para ${machine_id}`, {
			id: commandId,
			command,
			args,
		});
	} else {
		logger.warning(`Máquina ${machine_id} não conectada via WebSocket`);
	}

	res.json(commandData);
});

/**
 * Obter status de um comando
 */
router.get("/:id", authenticate, (req, res) => {
	const commandId = req.params.id;
	const command = storage.getCommand(commandId);

	if (!command) {
		logger.warning(`Comando ${commandId} não encontrado`);
		return res.status(404).json({
			error: "Comando não encontrado",
		});
	}

	logger.debug(`Status do comando ${commandId} solicitado`);

	res.json(command);
});

/**
 * Listar todos os comandos
 */
router.get("/", authenticate, (req, res) => {
	const commands = storage.getAllCommands();

	logger.debug(`Listando ${commands.length} comandos`);

	res.json(commands);
});

module.exports = router;
