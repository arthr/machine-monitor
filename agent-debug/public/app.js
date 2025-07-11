class AgentDebugDashboard {
  constructor() {
    this.messages = [];
    this.machines = [];
    this.stats = {};
    this.autoScroll = true;
    this.filters = {
      endpoint: "",
      type: "",
    };

    this.init();
  }

  init() {
    this.connectWebSocket();
    this.loadInitialData();
    this.setupEventListeners();

    // Atualizar dados a cada 10 segundos
    setInterval(() => this.loadStats(), 10000);
  }

  connectWebSocket() {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}`;

    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      console.log("✅ WebSocket conectado");
      this.updateConnectionStatus(true);
    };

    this.ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      this.handleWebSocketMessage(data);
    };

    this.ws.onclose = () => {
      console.log("❌ WebSocket desconectado");
      this.updateConnectionStatus(false);

      // Tentar reconectar após 3 segundos
      setTimeout(() => this.connectWebSocket(), 3000);
    };

    this.ws.onerror = (error) => {
      console.error("❌ Erro no WebSocket:", error);
      this.updateConnectionStatus(false);
    };
  }

  handleWebSocketMessage(data) {
    switch (data.type) {
      case "new_message":
        this.addMessage(data.message);
        break;
      case "initial_data":
        this.messages = data.messages || [];
        this.machines = data.machines || [];
        this.renderMessages();
        this.renderMachines();
        break;
      case "messages_cleared":
        this.messages = [];
        this.renderMessages();
        break;
      case "ping":
        this.addMessage({
          type: "ping",
          timestamp: new Date().toISOString(),
          data: {},
          headers: {},
        });
        break;
    }
  }

  updateConnectionStatus(connected) {
    const statusElement = document.getElementById("connectionStatus");
    const statusDot = document.querySelector(".status-dot");

    if (connected) {
      statusElement.textContent = "Conectado";
      statusDot.style.backgroundColor = "#238636";
    } else {
      statusElement.textContent = "Desconectado";
      statusDot.style.backgroundColor = "#da3633";
    }
  }

  async loadInitialData() {
    try {
      await Promise.all([
        this.loadMessages(),
        this.loadMachines(),
        this.loadStats(),
      ]);
    } catch (error) {
      console.error("❌ Erro ao carregar dados iniciais:", error);
    }
  }

  async loadMessages() {
    try {
      const response = await fetch("/debug/messages?limit=100");
      const data = await response.json();
      this.messages = data.messages;
      this.renderMessages();
    } catch (error) {
      console.error("❌ Erro ao carregar mensagens:", error);
    }
  }

  async loadMachines() {
    try {
      const response = await fetch("/debug/machines");
      const data = await response.json();
      this.machines = data.machines;
      this.renderMachines();
    } catch (error) {
      console.error("❌ Erro ao carregar máquinas:", error);
    }
  }

  async loadStats() {
    try {
      const response = await fetch("/debug/stats");
      this.stats = await response.json();
      this.renderStats();
    } catch (error) {
      console.error("❌ Erro ao carregar estatísticas:", error);
    }
  }

  addMessage(message) {
    this.messages.unshift(message);

    // Manter apenas as últimas 500 mensagens no frontend
    if (this.messages.length > 500) {
      this.messages = this.messages.slice(0, 500);
    }

    this.renderMessages();

    if (this.autoScroll) {
      setTimeout(() => {
        const messagesList = document.getElementById("messagesList");
        messagesList.scrollTop = 0;
      }, 100);
    }
  }

  renderMessages() {
    const container = document.getElementById("messagesList");
    const filteredMessages = this.getFilteredMessages();

    if (filteredMessages.length === 0) {
      container.innerHTML = `
                <div style="text-align: center; color: #7d8590; margin-top: 2rem;">
                    ${
                      this.messages.length === 0
                        ? "Aguardando mensagens do agente..."
                        : "Nenhuma mensagem corresponde aos filtros"
                    }
                </div>
            `;
      return;
    }

    container.innerHTML = filteredMessages
      .map(
        (message) => `
            <div class="message-item">
                <div class="message-header" onclick="toggleMessageContent('${
                  message.id
                }')">
                    <div class="message-meta">
                        <span class="method ${message.type}">${
          message.type
        }</span>
                        <span class="endpoint">${message.endpoint}</span>
                        <span class="timestamp">${this.formatTimestamp(
                          message.timestamp
                        )}</span>
                    </div>
                    <span class="size">${this.formatBytes(message.size)}</span>
                </div>
                <div class="message-content" id="content-${message.id}">
                    <h4>📋 Dados:</h4>
                    <div class="json-content">${this.formatJSON(
                      message.data
                    )}</div>
                    
                    ${
                      Object.keys(message.headers).length > 0
                        ? `
                        <h4 style="margin-top: 1rem;">📤 Headers:</h4>
                        <div class="json-content">${this.formatJSON(
                          message.headers
                        )}</div>
                    `
                        : ""
                    }
                </div>
            </div>
        `
      )
      .join("");

    // Atualizar contador
    document.getElementById("messageCount").textContent =
      filteredMessages.length;
    document.getElementById("lastUpdate").textContent =
      new Date().toLocaleTimeString();
  }

  renderMachines() {
    const container = document.getElementById("machinesContainer");

    if (this.machines.length === 0) {
      container.innerHTML = `
                <div style="text-align: center; color: #7d8590;">
                    Nenhuma máquina registrada
                </div>
            `;
      return;
    }

    container.innerHTML = this.machines
      .map(
        (machine) => `
            <div class="machine-item">
                <div class="machine-id">${machine.id}</div>
                <div class="machine-info">
                    ${machine.hostname || "N/A"} | ${machine.os || "N/A"}<br>
                    Último contato: ${this.formatTimestamp(machine.last_seen)}
                </div>
            </div>
        `
      )
      .join("");
  }

  renderStats() {
    document.getElementById("totalMessages").textContent =
      this.stats.total_messages || 0;
    document.getElementById("totalMachines").textContent =
      this.stats.total_machines || 0;
    document.getElementById("last24h").textContent = this.stats.last_24h || 0;
  }

  getFilteredMessages() {
    return this.messages.filter((message) => {
      const matchesEndpoint =
        !this.filters.endpoint ||
        message.endpoint
          .toLowerCase()
          .includes(this.filters.endpoint.toLowerCase());
      const matchesType =
        !this.filters.type || message.type === this.filters.type;

      return matchesEndpoint && matchesType;
    });
  }

  setupEventListeners() {
    // Filtro por endpoint
    document.getElementById("filterInput").addEventListener("input", (e) => {
      this.filters.endpoint = e.target.value;
      this.renderMessages();
    });

    // Filtro por tipo
    document.getElementById("typeFilter").addEventListener("change", (e) => {
      this.filters.type = e.target.value;
      this.renderMessages();
    });
  }

  formatTimestamp(timestamp) {
    const date = new Date(timestamp);
    return date.toLocaleString("pt-BR");
  }

  formatBytes(bytes) {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  }

  formatJSON(obj) {
    return JSON.stringify(obj, null, 2);
  }

  async clearMessages() {
    if (confirm("Tem certeza que deseja limpar todas as mensagens?")) {
      try {
        await fetch("/debug/messages", { method: "DELETE" });
        this.messages = [];
        this.renderMessages();
      } catch (error) {
        console.error("❌ Erro ao limpar mensagens:", error);
        alert("Erro ao limpar mensagens");
      }
    }
  }

  toggleAutoScroll() {
    this.autoScroll = !this.autoScroll;
    const btn = document.getElementById("autoScrollBtn");

    if (this.autoScroll) {
      btn.classList.add("active");
      btn.textContent = "📜 Auto Scroll (ON)";
    } else {
      btn.classList.remove("active");
      btn.textContent = "📜 Auto Scroll (OFF)";
    }
  }

  async refreshData() {
    await this.loadInitialData();
  }
}

// Funções globais para os event handlers
let dashboard;

function toggleMessageContent(messageId) {
  const content = document.getElementById(`content-${messageId}`);
  content.classList.toggle("expanded");
}

function clearMessages() {
  dashboard.clearMessages();
}

function toggleAutoScroll() {
  dashboard.toggleAutoScroll();
}

function refreshData() {
  dashboard.refreshData();
}

function filterMessages() {
  // Esta função é chamada pelos event listeners configurados no setupEventListeners
}

// Inicializar dashboard quando a página carregar
document.addEventListener("DOMContentLoaded", () => {
  dashboard = new AgentDebugDashboard();
});
