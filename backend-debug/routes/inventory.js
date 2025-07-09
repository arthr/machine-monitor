/**
 * Rotas para inventário
 */

const express = require("express");
const router = express.Router();
const storage = require("../storage/memory");
const logger = require("../utils/logger");
const { authenticate } = require("../middleware/auth");

/**
 * Receber inventário de uma máquina
 */
router.post("/", authenticate, (req, res) => {
	const { machine_id } = req.body;

	if (!machine_id) {
		logger.warning("Inventário recebido sem machine_id");
		return res.status(400).json({
			error: "machine_id é obrigatório",
		});
	}

	logger.inventory(`Inventário recebido de ${machine_id}`);
	logger.debug("Dados do inventário:", req.body);

	// Salvar inventário
	storage.addInventory(machine_id, req.body);

	res.json({
		status: "ok",
		timestamp: new Date().toISOString(),
	});
});

module.exports = router;
