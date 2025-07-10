package collector

import (
	"time"

	"agente-poc/internal/logging"
)

// SystemCollector é responsável por coletar dados do sistema
type SystemCollector struct {
	interval time.Duration
	logger   logging.Logger
}

// New cria uma nova instância do SystemCollector
func New(interval time.Duration, logger logging.Logger) *SystemCollector {
	return &SystemCollector{
		interval: interval,
		logger:   logger,
	}
}

// CollectInventory coleta informações completas do sistema
func (c *SystemCollector) CollectInventory() (*InventoryData, error) {
	c.logger.Debug("Collecting system inventory...")

	// TODO: Implementar coleta real de dados
	// Por enquanto, retornar dados mock
	return &InventoryData{
		MachineID:   "mock-machine-id",
		Timestamp:   time.Now(),
		CollectedAt: time.Now().Format(time.RFC3339),
		System: SystemInfo{
			Hostname:     "localhost",
			Platform:     "darwin",
			OSVersion:    "macOS 14.5",
			Architecture: "arm64",
			KernelArch:   "arm64",
			UserCount:    1,
		},
		Hardware: HardwareInfo{
			CPU: CPUInfo{
				Model:     "Apple M3",
				Cores:     8,
				Threads:   8,
				Frequency: 3200.0,
				Usage:     []float64{10.5, 8.2, 12.1, 6.3, 9.8, 7.4, 11.2, 5.9},
				Vendor:    "Apple",
				Family:    "M3",
			},
			Memory: MemoryInfo{
				Total:       16 * 1024 * 1024 * 1024, // 16GB
				Available:   8 * 1024 * 1024 * 1024,  // 8GB
				Used:        8 * 1024 * 1024 * 1024,  // 8GB
				UsedPercent: 50.0,
				Free:        8 * 1024 * 1024 * 1024, // 8GB
				Swap: SwapInfo{
					Total:       2 * 1024 * 1024 * 1024, // 2GB
					Used:        0,
					Free:        2 * 1024 * 1024 * 1024, // 2GB
					UsedPercent: 0.0,
				},
			},
		},
		Software: SoftwareInfo{
			InstalledApplications: []Application{},
			RunningServices:       []Service{},
			RunningProcesses:      []Process{},
		},
		Network: NetworkInfo{
			Interfaces: []NetworkInterface{},
			Statistics: NetworkStatistics{},
		},
	}, nil
}

// CollectBasicInfo coleta informações básicas do sistema
func (c *SystemCollector) CollectBasicInfo() (*SystemInfo, error) {
	c.logger.Debug("Collecting basic system info...")

	// TODO: Implementar coleta real de dados
	return &SystemInfo{
		Hostname:     "localhost",
		Platform:     "darwin",
		OSVersion:    "macOS 14.5",
		Architecture: "arm64",
		KernelArch:   "arm64",
		UserCount:    1,
	}, nil
}
