/**
 * Rotas para máquinas
 */

const express = require("express");
const router = express.Router();
const storage = require("../storage/memory");
const logger = require("../utils/logger");
const { authenticate } = require("../middleware/auth");

/**
 * Listar todas as máquinas
 */
router.get("/", authenticate, (req, res) => {
	const machines = storage.getAllMachines();

	logger.debug(`Listando ${machines.length} máquinas`);

	res.json(machines);
});

/**
 * Obter dados de uma máquina específica
 */
router.get("/:id", authenticate, (req, res) => {
	const machineId = req.params.id;
	const machine = storage.getMachine(machineId);

	if (!machine) {
		logger.warning(`Máquina ${machineId} não encontrada`);
		return res.status(404).json({
			error: "Máquina não encontrada",
		});
	}

	logger.debug(`Detalhes da máquina ${machineId} solicitados`);

	res.json({
		machine,
		heartbeats: storage.getHeartbeats(machineId),
		inventories: storage.getInventories(machineId),
		connected: storage.hasWebSocketConnection(machineId),
	});
});

module.exports = router;
