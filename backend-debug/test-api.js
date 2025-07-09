#!/usr/bin/env node

/**
 * Script de teste para validar o backend de debug
 * Execute: node test-api.js
 */

const WebSocket = require("ws");

const BASE_URL = "http://localhost:8080";
const WS_URL = "ws://localhost:8080";
const AUTH_TOKEN = "Bearer dev-token-123";

console.log("🧪 Iniciando testes do backend...\n");

// Teste HTTP
async function testHTTP() {
	console.log("📡 Testando APIs HTTP...");

	try {
		// Teste de heartbeat
		const heartbeatResponse = await fetch(`${BASE_URL}/heartbeat`, {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
				Authorization: AUTH_TOKEN,
			},
			body: JSON.stringify({
				machine_id: "test-machine",
				status: "online",
				hostname: "Test Machine",
			}),
		});

		const heartbeatResult = await heartbeatResponse.json();
		console.log("✅ Heartbeat:", heartbeatResult);

		// Teste de listar máquinas
		const machinesResponse = await fetch(`${BASE_URL}/machines`, {
			headers: { Authorization: AUTH_TOKEN },
		});

		const machines = await machinesResponse.json();
		console.log("✅ Máquinas:", machines);

		// Teste de estatísticas
		const statsResponse = await fetch(`${BASE_URL}/debug/stats`);
		const stats = await statsResponse.json();
		console.log("✅ Stats:", stats);

		// Teste de envio de comando
		const commandResponse = await fetch(`${BASE_URL}/commands`, {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
				Authorization: AUTH_TOKEN,
			},
			body: JSON.stringify({
				machine_id: "test-machine",
				command: "uptime",
				args: [],
			}),
		});

		const commandResult = await commandResponse.json();
		console.log("✅ Comando enviado:", commandResult);

		console.log("\n📡 Testes HTTP completos!\n");
	} catch (error) {
		console.error("❌ Erro nos testes HTTP:", error.message);
	}
}

// Teste WebSocket
function testWebSocket() {
	return new Promise((resolve, reject) => {
		console.log("🔌 Testando WebSocket...");

		const ws = new WebSocket(WS_URL);

		ws.on("open", () => {
			console.log("✅ WebSocket conectado");

			// Registrar máquina
			ws.send(
				JSON.stringify({
					machine_id: "test-machine",
				})
			);

			console.log("✅ Máquina registrada via WebSocket");
		});

		ws.on("message", (data) => {
			const message = JSON.parse(data.toString());
			console.log("✅ Comando recebido via WebSocket:", message);

			// Simular resposta do comando
			setTimeout(() => {
				ws.send(
					JSON.stringify({
						id: message.id,
						output: "teste de comando executado com sucesso",
						error: null,
					})
				);

				console.log("✅ Resposta enviada via WebSocket");

				// Fechar conexão
				ws.close();

				setTimeout(() => {
					console.log("\n🔌 Teste WebSocket completo!\n");
					resolve();
				}, 1000);
			}, 1000);
		});

		ws.on("error", (error) => {
			console.error("❌ Erro WebSocket:", error.message);
			reject(error);
		});

		ws.on("close", () => {
			console.log("🔌 WebSocket desconectado");
		});
	});
}

// Executar testes
async function runTests() {
	try {
		await testHTTP();
		await testWebSocket();

		console.log("🎉 Todos os testes passaram!");
		console.log("\n📋 Próximos passos:");
		console.log("1. npm start - Iniciar servidor");
		console.log("2. Abrir http://localhost:8080");
		console.log("3. Conectar agente Go");
	} catch (error) {
		console.error("❌ Erro geral:", error.message);
		process.exit(1);
	}
}

// Verificar se o servidor está rodando
fetch(`${BASE_URL}/debug/stats`)
	.then(() => {
		console.log("✅ Servidor detectado em http://localhost:8080\n");
		runTests();
	})
	.catch(() => {
		console.log("❌ Servidor não está rodando!");
		console.log("💡 Execute: npm start\n");
		process.exit(1);
	});
