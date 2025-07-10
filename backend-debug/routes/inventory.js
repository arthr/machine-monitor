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
	const { machine_id, type, timestamp, data, checksum } = req.body;

	if (!machine_id) {
		logger.warning("Inventário recebido sem machine_id");
		return res.status(400).json({
			error: "machine_id é obrigatório",
		});
	}

	// Extrair informações principais do inventário
	const inventoryInfo = {
		machine_id,
		type: type || "inventory",
		timestamp: timestamp || new Date().toISOString(),
		checksum,
		received_at: new Date().toISOString(),
	};

	// Se há dados de inventário, extrair informações principais
	if (data) {
		inventoryInfo.system = {
			hostname: data.system?.hostname,
			platform: data.system?.platform,
			architecture: data.system?.architecture,
			os_version: data.system?.os_version,
			uptime: data.system?.uptime,
		};

		inventoryInfo.hardware = {
			cpu: data.hardware?.cpu
				? {
						model: data.hardware.cpu.model,
						cores: data.hardware.cpu.cores,
						usage: data.hardware.cpu.usage,
				  }
				: null,
			memory: data.hardware?.memory
				? {
						total: data.hardware.memory.total,
						used: data.hardware.memory.used,
						used_percent: data.hardware.memory.used_percent,
				  }
				: null,
			disk: data.hardware?.disk
				? data.hardware.disk.map((d) => ({
						device: d.device,
						mountpoint: d.mountpoint,
						size: d.size,
						used: d.used,
						used_percent: d.used_percent,
				  }))
				: [],
		};

		inventoryInfo.software = {
			applications_count:
				data.software?.installed_applications?.length || 0,
			processes_count: data.software?.running_processes?.length || 0,
			services_count: data.software?.running_services?.length || 0,
		};

		inventoryInfo.network = {
			interfaces_count: data.network?.interfaces?.length || 0,
		};

		// Salvar dados completos para detalhes
		inventoryInfo.full_data = data;
	}

	logger.inventory(`📦 Inventário recebido de ${machine_id}`, {
		hostname: inventoryInfo.system?.hostname,
		platform: inventoryInfo.system?.platform,
		applications: inventoryInfo.software?.applications_count,
		processes: inventoryInfo.software?.processes_count,
		services: inventoryInfo.software?.services_count,
		checksum,
	});

	// Salvar inventário
	storage.addInventory(machine_id, inventoryInfo);

	// Atualizar informações da máquina
	if (inventoryInfo.system) {
		storage.setMachine(machine_id, {
			hostname: inventoryInfo.system.hostname,
			platform: inventoryInfo.system.platform,
			architecture: inventoryInfo.system.architecture,
			os_version: inventoryInfo.system.os_version,
			last_inventory: new Date().toISOString(),
			applications_count: inventoryInfo.software?.applications_count,
			processes_count: inventoryInfo.software?.processes_count,
			services_count: inventoryInfo.software?.services_count,
		});
	}

	res.json({
		status: "ok",
		timestamp: new Date().toISOString(),
		message: "Inventário recebido com sucesso",
		checksum_received: checksum,
	});
});

module.exports = router;
