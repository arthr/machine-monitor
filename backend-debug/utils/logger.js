/**
 * Utilitário de logging centralizado
 */

const config = require("../config/config");

class Logger {
	constructor() {
		this.level = config.debug.logLevel;
		this.verbose = config.debug.verbose;
	}

	log(message, data = null) {
		const timestamp = new Date().toISOString();
		console.log(`[${timestamp}] ${message}`);

		if (data) {
			if (this.verbose) {
				console.log(JSON.stringify(data, null, 2));
			} else {
				console.log(JSON.stringify(data));
			}
		}
	}

	info(message, data = null) {
		this.log(`ℹ️  ${message}`, data);
	}

	success(message, data = null) {
		this.log(`✅ ${message}`, data);
	}

	warning(message, data = null) {
		this.log(`⚠️  ${message}`, data);
	}

	error(message, data = null) {
		this.log(`❌ ${message}`, data);
	}

	debug(message, data = null) {
		if (this.verbose) {
			this.log(`🔍 ${message}`, data);
		}
	}

	websocket(message, data = null) {
		this.log(`🔌 ${message}`, data);
	}

	command(message, data = null) {
		this.log(`📤 ${message}`, data);
	}

	heartbeat(message, data = null) {
		this.log(`💓 ${message}`, data);
	}

	inventory(message, data = null) {
		this.log(`📦 ${message}`, data);
	}
}

module.exports = new Logger();
