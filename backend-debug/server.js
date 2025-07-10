/**
 * Servidor principal do backend de debug
 * Refatorado com separação de responsabilidades
 */

const express = require("express");
const cors = require("cors");
const config = require("./config/config");
const logger = require("./utils/logger");
const { setupWebSocketServer } = require("./websocket/handler");

// Importar rotas
const heartbeatRoutes = require("./routes/heartbeat");
const inventoryRoutes = require("./routes/inventory");
const machinesRoutes = require("./routes/machines");
const commandsRoutes = require("./routes/commands");
const debugRoutes = require("./routes/debug");

const app = express();

// ==================== MIDDLEWARE ====================

app.use(cors());
// Aumentar limite do body-parser para aceitar inventários grandes
app.use(express.json({ limit: "50mb" }));
app.use(express.urlencoded({ extended: true, limit: "50mb" }));
app.use(express.static(config.paths.public));

// ==================== ROTAS ====================

// Página principal
app.get("/", (req, res) => {
	res.sendFile(config.paths.index);
});

// Registrar rotas
app.use("/heartbeat", heartbeatRoutes);
app.use("/inventory", inventoryRoutes);
app.use("/machines", machinesRoutes);
app.use("/commands", commandsRoutes);
app.use("/debug", debugRoutes);

// ==================== SERVIDOR ====================

const server = app.listen(config.server.port, () => {
	logger.success(
		`Servidor rodando em http://${config.server.host}:${config.server.port}`
	);
});

// Configurar WebSocket
const wss = setupWebSocketServer(server);

// ==================== GRACEFUL SHUTDOWN ====================

function gracefulShutdown() {
	logger.info("Iniciando shutdown graceful...");

	server.close(() => {
		logger.success("Servidor parado");
		process.exit(0);
	});
}

process.on("SIGTERM", gracefulShutdown);
process.on("SIGINT", gracefulShutdown);
