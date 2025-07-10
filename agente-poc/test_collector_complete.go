package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"agente-poc/internal/collector"
	"agente-poc/internal/logging"
)

func main() {
	fmt.Println("🚀 TESTE COMPLETO DO COLLECTOR - TASK 05 + MACHINE ID")
	fmt.Println(strings.Repeat("=", 60))

	// Configurar logger
	logger, err := logging.NewLogger(nil)
	if err != nil {
		log.Fatalf("Erro ao criar logger: %v", err)
	}

	// Criar collector
	c := collector.New(30*time.Second, logger)

	// Marcar início do teste
	startTime := time.Now()
	fmt.Printf("⏱️  Iniciando coleta completa em: %s\n", startTime.Format("15:04:05"))

	// Executar coleta completa
	inventory, err := c.CollectInventory()
	if err != nil {
		log.Fatalf("❌ Erro na coleta de inventário: %v", err)
	}

	// Calcular tempo de execução
	duration := time.Since(startTime)
	fmt.Printf("✅ Coleta concluída em: %s\n", duration.String())

	// Calcular tamanho dos dados JSON
	jsonData, err := json.Marshal(inventory)
	if err != nil {
		log.Fatalf("❌ Erro ao serializar dados: %v", err)
	}
	jsonSize := float64(len(jsonData)) / 1024.0 // KB

	fmt.Println("\n🔍 RESULTADOS DA COLETA:")
	fmt.Println(strings.Repeat("=", 50))

	// 1. MACHINE ID (NOVO!)
	fmt.Printf("🆔 Machine ID: %s\n", inventory.MachineID)
	if inventory.MachineID == "" {
		fmt.Println("❌ ERRO: Machine ID está vazio!")
	} else {
		fmt.Println("✅ Machine ID gerado com sucesso!")
	}

	// 2. INFORMAÇÕES DO SISTEMA
	fmt.Println("\n💻 SISTEMA:")
	fmt.Printf("   📛 Hostname: %s\n", inventory.System.Hostname)
	fmt.Printf("   🖥️  Platform: %s\n", inventory.System.Platform)
	fmt.Printf("   📊 OS Version: %s\n", inventory.System.OSVersion)
	fmt.Printf("   🏗️  Architecture: %s\n", inventory.System.Architecture)
	fmt.Printf("   ⏰ Uptime: %d segundos (%.2f horas)\n", inventory.System.Uptime, float64(inventory.System.Uptime)/3600)
	fmt.Printf("   👥 Usuários: %d\n", inventory.System.UserCount)

	// 3. HARDWARE
	fmt.Println("\n🔧 HARDWARE:")
	fmt.Printf("   💾 CPU: %s\n", inventory.Hardware.CPU.Model)
	fmt.Printf("   🔢 Cores: %d\n", inventory.Hardware.CPU.Cores)
	fmt.Printf("   ⚡ Frequência: %.2f MHz\n", inventory.Hardware.CPU.Frequency)

	// Uso da CPU
	if len(inventory.Hardware.CPU.Usage) > 0 {
		var avgCPU float64
		for _, usage := range inventory.Hardware.CPU.Usage {
			avgCPU += usage
		}
		avgCPU /= float64(len(inventory.Hardware.CPU.Usage))
		fmt.Printf("   📈 Uso médio CPU: %.2f%%\n", avgCPU)
	}

	// Memória
	totalMemGB := float64(inventory.Hardware.Memory.Total) / (1024 * 1024 * 1024)
	usedMemGB := float64(inventory.Hardware.Memory.Used) / (1024 * 1024 * 1024)
	fmt.Printf("   🧠 Memória Total: %.2f GB\n", totalMemGB)
	fmt.Printf("   📊 Memória Usada: %.2f GB (%.2f%%)\n", usedMemGB, inventory.Hardware.Memory.UsedPercent)

	// Discos
	fmt.Printf("   💽 Partições: %d\n", len(inventory.Hardware.Disk))
	for i, disk := range inventory.Hardware.Disk {
		if i >= 3 { // Mostrar apenas primeiras 3 partições
			fmt.Printf("   ... e mais %d partições\n", len(inventory.Hardware.Disk)-3)
			break
		}
		totalGB := float64(disk.Total) / (1024 * 1024 * 1024)
		fmt.Printf("   💿 %s: %.2f GB (%.1f%% usado)\n", disk.Device, totalGB, disk.UsedPercent)
	}

	// 4. SOFTWARE
	fmt.Println("\n📦 SOFTWARE:")
	fmt.Printf("   🎯 Aplicações: %d instaladas\n", len(inventory.Software.InstalledApplications))
	fmt.Printf("   🔄 Processos: %d em execução\n", len(inventory.Software.RunningProcesses))
	fmt.Printf("   ⚙️  Serviços: %d detectados\n", len(inventory.Software.RunningServices))

	// Mostrar algumas aplicações
	fmt.Println("   📱 Aplicações principais:")
	for i, app := range inventory.Software.InstalledApplications {
		if i >= 5 { // Mostrar apenas primeiras 5
			fmt.Printf("   ... e mais %d aplicações\n", len(inventory.Software.InstalledApplications)-5)
			break
		}
		fmt.Printf("      - %s %s\n", app.Name, app.Version)
	}

	// Processos top CPU
	fmt.Println("   🔥 Processos com maior uso de CPU:")
	processCount := 0
	for _, proc := range inventory.Software.RunningProcesses {
		if proc.CPUPercent > 0.1 && processCount < 5 {
			fmt.Printf("      - %s (PID: %d, CPU: %.2f%%)\n", proc.Name, proc.PID, proc.CPUPercent)
			processCount++
		}
	}

	// 5. REDE
	fmt.Println("\n🌐 REDE:")
	fmt.Printf("   🔗 Interfaces: %d\n", len(inventory.Network.Interfaces))

	activeInterfaces := 0
	for _, iface := range inventory.Network.Interfaces {
		if len(iface.IPAddresses) > 0 {
			activeInterfaces++
			if activeInterfaces <= 3 { // Mostrar apenas primeiras 3 interfaces ativas
				fmt.Printf("   📡 %s: %s\n", iface.Name, iface.IPAddresses[0])
			}
		}
	}

	totalBytesSent := float64(inventory.Network.Statistics.TotalBytesSent) / (1024 * 1024)
	totalBytesRecv := float64(inventory.Network.Statistics.TotalBytesRecv) / (1024 * 1024)
	fmt.Printf("   📤 Dados enviados: %.2f MB\n", totalBytesSent)
	fmt.Printf("   📥 Dados recebidos: %.2f MB\n", totalBytesRecv)

	// 6. INFORMAÇÕES ESPECÍFICAS DO MACOS
	if inventory.MacOSSpecific != nil {
		fmt.Println("\n🍎 MACOS ESPECÍFICO:")

		if inventory.MacOSSpecific.Homebrew != nil {
			fmt.Printf("   🍺 Homebrew: %s\n", inventory.MacOSSpecific.Homebrew.Version)
			fmt.Printf("   📦 Pacotes Homebrew: %d\n", len(inventory.MacOSSpecific.Homebrew.InstalledPackages))
		}

		if inventory.MacOSSpecific.XcodeVersion != "" {
			fmt.Printf("   👨‍💻 Xcode: %s\n", inventory.MacOSSpecific.XcodeVersion)
		}

		if len(inventory.MacOSSpecific.LaunchdServices) > 0 {
			fmt.Printf("   🚀 Serviços launchd: %d\n", len(inventory.MacOSSpecific.LaunchdServices))
		}
	}

	// 7. ESTATÍSTICAS DO CACHE
	fmt.Println("\n📊 ESTATÍSTICAS:")
	cacheStats := c.GetCacheStats()
	fmt.Printf("   💾 Cache habilitado: %v\n", cacheStats["enabled"])
	fmt.Printf("   📄 Itens em cache: %v\n", cacheStats["items"])
	fmt.Printf("   ⏳ Itens expirados: %v\n", cacheStats["expired"])
	fmt.Printf("   📏 Tamanho JSON: %.2f KB\n", jsonSize)
	fmt.Printf("   ⏱️  Tempo de coleta: %s\n", duration.String())

	// 8. RESUMO FINAL
	fmt.Println("\n🎯 RESUMO FINAL:")
	fmt.Printf("   ✅ Machine ID: %s\n", inventory.MachineID)
	fmt.Printf("   ✅ %d aplicações detectadas\n", len(inventory.Software.InstalledApplications))
	fmt.Printf("   ✅ %d processos monitorados\n", len(inventory.Software.RunningProcesses))
	fmt.Printf("   ✅ %d serviços descobertos\n", len(inventory.Software.RunningServices))
	fmt.Printf("   ✅ %d partições analisadas\n", len(inventory.Hardware.Disk))
	fmt.Printf("   ✅ CPU %s com %d cores\n", inventory.Hardware.CPU.Model, inventory.Hardware.CPU.Cores)
	fmt.Printf("   ✅ %.2f GB memória total, %.2f%% uso\n", totalMemGB, inventory.Hardware.Memory.UsedPercent)
	fmt.Printf("   ✅ %.2f KB dados JSON coletados\n", jsonSize)
	fmt.Printf("   ✅ %s tempo de coleta\n", duration.String())
	fmt.Printf("   ✅ Cache: %v itens armazenados\n", cacheStats["items"])

	if inventory.MacOSSpecific != nil {
		if inventory.MacOSSpecific.Homebrew != nil {
			fmt.Printf("   ✅ Homebrew instalado, %d pacotes\n", len(inventory.MacOSSpecific.Homebrew.InstalledPackages))
		}
		if inventory.MacOSSpecific.XcodeVersion != "" {
			fmt.Printf("   ✅ %s detectado\n", inventory.MacOSSpecific.XcodeVersion)
		}
	}

	fmt.Println("\n🎉 TESTE COMPLETO CONCLUÍDO COM SUCESSO!")
	fmt.Println(strings.Repeat("=", 60))
}
