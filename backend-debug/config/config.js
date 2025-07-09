/**
 * Configuração do backend de debug
 */

const path = require("path");

module.exports = {
	server: {
		port: process.env.PORT || 8080,
		host: process.env.HOST || "localhost",
	},

	auth: {
		token: process.env.AUTH_TOKEN || "dev-token-123",
	},

	storage: {
		maxHeartbeats: 100,
		maxInventories: 10,
		maxCommands: 1000,
	},

	websocket: {
		pingInterval: 30000,
		pongTimeout: 5000,
	},

	debug: {
		verbose: process.env.DEBUG === "true",
		logLevel: process.env.LOG_LEVEL || "info",
	},

	paths: {
		public: path.join(__dirname, "../public"),
		index: path.join(__dirname, "../public/index.html"),
	},

	// Comandos permitidos por plataforma
	allowedCommands: {
		darwin: [
			"ps",
			"system_profiler",
			"launchctl",
			"uptime",
			"whoami",
			"uname",
			"df",
			"top",
			"netstat",
			"sw_vers",
			"diskutil",
			"pmset",
			"scutil",
		],
		linux: [
			"ps",
			"systemctl",
			"uptime",
			"whoami",
			"uname",
			"df",
			"top",
			"netstat",
			"lsb_release",
			"free",
			"lscpu",
			"lsblk",
		],
		windows: ["tasklist", "systeminfo", "wmic", "sc", "net", "powershell"],
	},
};
