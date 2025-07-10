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
	const { machine_id, command, args = [], timeout = 30 } = req.body;

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
		command: command,
		args,
		status: "pending",
		timeout,
		created_at: new Date().toISOString(),
	};

	storage.addCommand(commandId, commandData);

	// Enviar via WebSocket se conectado usando o formato do agente
	const wsConn = storage.getWebSocketConnection(machine_id);
	if (wsConn && wsConn.readyState === WebSocket.OPEN) {
		const websocketMessage = {
			type: "command",
			id: commandId,
			timestamp: new Date().toISOString(),
			data: {
				type: "system",
				command: command,
				args: args,
				timeout: timeout,
				options: {},
			},
		};

		wsConn.send(JSON.stringify(websocketMessage));

		logger.command(`📤 Comando enviado via WebSocket para ${machine_id}`, {
			id: commandId,
			command,
			args,
			timeout,
		});
	} else {
		logger.warning(`⚠️  Máquina ${machine_id} não conectada via WebSocket`);
	}

	res.json({
		...commandData,
		message: "Comando enviado com sucesso",
	});
});

/**
 * Receber resultado de comando
 */
router.post("/result", authenticate, (req, res) => {
	const {
		id,
		command_id,
		status,
		output,
		error,
		exit_code,
		execution_time_ms,
	} = req.body;

	if (!id && !command_id) {
		logger.warning("Resultado recebido sem id ou command_id");
		return res.status(400).json({
			error: "id ou command_id é obrigatório",
		});
	}

	const targetId = command_id || id;
	const command = storage.getCommand(targetId);

	if (!command) {
		logger.warning(`Comando ${targetId} não encontrado para resultado`);
		return res.status(404).json({
			error: "Comando não encontrado",
		});
	}

	// Atualizar comando com resultado
	const updatedCommand = {
		...command,
		status: status || (error ? "failed" : "completed"),
		output: output || "",
		error: error || "",
		exit_code: exit_code || 0,
		execution_time_ms: execution_time_ms || 0,
		completed_at: new Date().toISOString(),
	};

	storage.updateCommand(targetId, updatedCommand);

	logger.command(`📥 Resultado recebido para comando ${targetId}`, {
		status: updatedCommand.status,
		hasOutput: !!output,
		hasError: !!error,
		exitCode: exit_code,
	});

	res.json({
		status: "ok",
		message: "Resultado recebido com sucesso",
		timestamp: new Date().toISOString(),
	});
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

	logger.debug(`📊 Status do comando ${commandId} solicitado`);

	res.json(command);
});

/**
 * Listar todos os comandos
 */
router.get("/", authenticate, (req, res) => {
	const commands = storage.getAllCommands();

	logger.debug(`📋 Listando ${commands.length} comandos`);

	res.json(commands);
});

module.exports = router;
