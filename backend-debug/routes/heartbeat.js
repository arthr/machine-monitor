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
	const {
		machine_id,
		status = "online",
		timestamp,
		agent_version,
		uptime_seconds,
		system_health,
		pending_commands,
		active_tasks,
	} = req.body;

	if (!machine_id) {
		logger.warning("Heartbeat recebido sem machine_id");
		return res.status(400).json({
			error: "machine_id é obrigatório",
		});
	}

	// Extrair hostname do system_health ou usar machine_id
	const hostname = req.body.hostname || machine_id.split("-")[0] || "unknown";

	logger.heartbeat(`💓 Heartbeat recebido de ${machine_id}`, {
		status,
		hostname,
		agent_version,
		uptime: uptime_seconds,
		system_health: system_health
			? {
					cpu: system_health.cpu_usage_percent,
					memory: system_health.memory_usage_percent,
					disk: system_health.disk_usage_percent,
					status: system_health.status,
			  }
			: null,
	});

	// Salvar heartbeat com dados enriquecidos
	const heartbeatData = {
		status,
		timestamp: timestamp || new Date().toISOString(),
		agent_version,
		uptime_seconds,
		system_health,
		pending_commands,
		active_tasks,
		received_at: new Date().toISOString(),
	};

	storage.addHeartbeat(machine_id, heartbeatData);

	// Atualizar ou criar máquina com dados enriquecidos
	storage.setMachine(machine_id, {
		status,
		hostname,
		agent_version,
		uptime_seconds,
		system_health,
		last_seen: new Date().toISOString(),
		last_heartbeat: new Date().toISOString(),
	});

	res.json({
		status: "ok",
		timestamp: new Date().toISOString(),
		message: "Heartbeat recebido com sucesso",
	});
});

module.exports = router;
