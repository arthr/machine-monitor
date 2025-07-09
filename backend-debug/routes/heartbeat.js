/**
 * Rotas para heartbeat
 */

const express = require("express");
const router = express.Router();
const storage = require("../storage/memory");
const logger = require("../utils/logger");
const { authenticate } = require("../middleware/auth");

/**
 * Receber heartbeat de uma máquina
 */
router.post("/", authenticate, (req, res) => {
	const { machine_id, status = "online", timestamp, hostname } = req.body;

	if (!machine_id) {
		logger.warning("Heartbeat recebido sem machine_id");
		return res.status(400).json({
			error: "machine_id é obrigatório",
		});
	}

	logger.heartbeat(`Heartbeat recebido de ${machine_id}`, {
		status,
		hostname,
		timestamp,
	});

	// Salvar heartbeat
	storage.addHeartbeat(machine_id, {
		status,
		timestamp: timestamp || new Date().toISOString(),
	});

	// Atualizar ou criar máquina
	storage.setMachine(machine_id, {
		status,
		hostname: hostname || "unknown",
		last_seen: new Date().toISOString(),
	});

	res.json({
		status: "ok",
		timestamp: new Date().toISOString(),
	});
});

module.exports = router;
