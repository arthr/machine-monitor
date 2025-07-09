/**
 * Storage em memória para dados do agente
 */

const config = require("../config/config");
const logger = require("../utils/logger");

class MemoryStorage {
	constructor() {
		this.machines = new Map();
		this.heartbeats = new Map();
		this.inventories = new Map();
		this.commands = new Map();
		this.wsConnections = new Map();

		logger.info("Storage em memória inicializado");
	}

	// ============ MACHINES ============

	setMachine(machineId, machineData) {
		this.machines.set(machineId, {
			id: machineId,
			...machineData,
			last_updated: new Date().toISOString(),
		});

		logger.debug(`Máquina ${machineId} atualizada`);
	}

	getMachine(machineId) {
		return this.machines.get(machineId);
	}

	getAllMachines() {
		return Array.from(this.machines.values());
	}

	// ============ HEARTBEATS ============

	addHeartbeat(machineId, heartbeatData) {
		if (!this.heartbeats.has(machineId)) {
			this.heartbeats.set(machineId, []);
		}

		const heartbeats = this.heartbeats.get(machineId);
		heartbeats.push({
			...heartbeatData,
			received_at: new Date().toISOString(),
		});

		// Manter apenas os últimos N heartbeats
		if (heartbeats.length > config.storage.maxHeartbeats) {
			heartbeats.shift();
		}

		logger.debug(`Heartbeat adicionado para ${machineId}`);
	}

	getHeartbeats(machineId) {
		return this.heartbeats.get(machineId) || [];
	}

	// ============ INVENTORIES ============

	addInventory(machineId, inventoryData) {
		if (!this.inventories.has(machineId)) {
			this.inventories.set(machineId, []);
		}

		const inventories = this.inventories.get(machineId);
		inventories.push({
			...inventoryData,
			received_at: new Date().toISOString(),
		});

		// Manter apenas os últimos N inventários
		if (inventories.length > config.storage.maxInventories) {
			inventories.shift();
		}

		logger.debug(`Inventário adicionado para ${machineId}`);
	}

	getInventories(machineId) {
		return this.inventories.get(machineId) || [];
	}

	// ============ COMMANDS ============

	addCommand(commandId, commandData) {
		this.commands.set(commandId, {
			...commandData,
			created_at: new Date().toISOString(),
		});

		logger.debug(`Comando ${commandId} adicionado`);
	}

	updateCommand(commandId, updates) {
		const command = this.commands.get(commandId);
		if (command) {
			this.commands.set(commandId, {
				...command,
				...updates,
				updated_at: new Date().toISOString(),
			});

			logger.debug(`Comando ${commandId} atualizado`);
		}
	}

	getCommand(commandId) {
		return this.commands.get(commandId);
	}

	getAllCommands() {
		return Array.from(this.commands.values());
	}

	// ============ WEBSOCKET CONNECTIONS ============

	setWebSocketConnection(machineId, wsConnection) {
		this.wsConnections.set(machineId, wsConnection);
		logger.websocket(`Conexão WebSocket registrada para ${machineId}`);
	}

	getWebSocketConnection(machineId) {
		return this.wsConnections.get(machineId);
	}

	removeWebSocketConnection(machineId) {
		this.wsConnections.delete(machineId);
		logger.websocket(`Conexão WebSocket removida para ${machineId}`);
	}

	hasWebSocketConnection(machineId) {
		return this.wsConnections.has(machineId);
	}

	// ============ STATISTICS ============

	getStats() {
		return {
			machines: this.machines.size,
			websocket_connections: this.wsConnections.size,
			total_commands: this.commands.size,
			total_heartbeats: Array.from(this.heartbeats.values()).reduce(
				(sum, arr) => sum + arr.length,
				0
			),
			total_inventories: Array.from(this.inventories.values()).reduce(
				(sum, arr) => sum + arr.length,
				0
			),
			uptime: process.uptime(),
		};
	}

	// ============ CLEAR DATA ============

	clear() {
		this.machines.clear();
		this.heartbeats.clear();
		this.inventories.clear();
		this.commands.clear();
		// Não limpar conexões WebSocket para não quebrar conexões ativas

		logger.info("Todos os dados foram limpos do storage");
	}
}

module.exports = new MemoryStorage();
