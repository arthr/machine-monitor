/**
 * Rotas para debug
 */

const express = require("express");
const router = express.Router();
const storage = require("../storage/memory");
const logger = require("../utils/logger");

/**
 * Obter estatísticas do servidor
 */
router.get("/stats", (req, res) => {
	const stats = storage.getStats();

	logger.debug("Estatísticas solicitadas", stats);

	res.json(stats);
});

/**
 * Limpar todos os dados do storage
 */
router.delete("/clear", (req, res) => {
	storage.clear();

	logger.info("Todos os dados foram limpos via API");

	res.json({
		message: "Dados limpos",
		timestamp: new Date().toISOString(),
	});
});

/**
 * Informações sobre o servidor
 */
router.get("/info", (req, res) => {
	const info = {
		version: process.version,
		platform: process.platform,
		arch: process.arch,
		memory: process.memoryUsage(),
		uptime: process.uptime(),
		timestamp: new Date().toISOString(),
	};

	logger.debug("Informações do servidor solicitadas");

	res.json(info);
});

module.exports = router;
