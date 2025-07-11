package collector

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"machine-monitor-agent/internal/types"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// Collector responsável por coletar informações do sistema
type Collector struct {
	mu          sync.RWMutex
	cache       map[string]interface{}
	cacheTTL    time.Duration
	cacheExpiry map[string]time.Time
}

// NewCollector cria uma nova instância do coletor
func NewCollector(cacheTTL time.Duration) *Collector {
	return &Collector{
		cache:       make(map[string]interface{}),
		cacheTTL:    cacheTTL,
		cacheExpiry: make(map[string]time.Time),
	}
}

// CollectSystemInfo coleta informações do sistema operacional
func (c *Collector) CollectSystemInfo(ctx context.Context) (*types.SystemInfo, error) {
	// Verifica cache
	if cached := c.getFromCache("system_info"); cached != nil {
		if sysInfo, ok := cached.(*types.SystemInfo); ok {
			return sysInfo, nil
		}
	}

	// Coleta informações do host
	hostInfo, err := host.InfoWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter informações do host: %w", err)
	}

	// Coleta usuários logados
	users, err := host.UsersWithContext(ctx)
	if err != nil {
		// Se não conseguir obter usuários, continua sem erro
		users = []host.UserStat{}
	}

	// Converte usuários para o formato interno
	var userList []types.User
	for _, user := range users {
		userList = append(userList, types.User{
			Username:  user.User,
			Terminal:  user.Terminal,
			Host:      user.Host,
			Started:   int64(user.Started),
			Timestamp: time.Now(),
		})
	}

	// Coleta número de processos
	processes, err := process.PidsWithContext(ctx)
	if err != nil {
		processes = []int32{}
	}

	sysInfo := &types.SystemInfo{
		OS:        hostInfo.OS,
		Platform:  hostInfo.Platform,
		Hostname:  hostInfo.Hostname,
		Uptime:    hostInfo.Uptime,
		BootTime:  hostInfo.BootTime,
		Procs:     uint64(len(processes)),
		Users:     userList,
		Timestamp: time.Now(),
	}

	// Armazena no cache
	c.setCache("system_info", sysInfo)

	return sysInfo, nil
}

// CollectHardwareInfo coleta informações de hardware
func (c *Collector) CollectHardwareInfo(ctx context.Context) (*types.HardwareInfo, error) {
	// Verifica cache
	if cached := c.getFromCache("hardware_info"); cached != nil {
		if hwInfo, ok := cached.(*types.HardwareInfo); ok {
			return hwInfo, nil
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	hwInfo := &types.HardwareInfo{
		Timestamp: time.Now(),
	}

	// Coleta informações da CPU
	wg.Add(1)
	go func() {
		defer wg.Done()
		cpuInfo, err := c.collectCPUInfo(ctx)
		if err == nil {
			mu.Lock()
			hwInfo.CPU = *cpuInfo
			mu.Unlock()
		}
	}()

	// Coleta informações da memória
	wg.Add(1)
	go func() {
		defer wg.Done()
		memInfo, err := c.collectMemoryInfo(ctx)
		if err == nil {
			mu.Lock()
			hwInfo.Memory = *memInfo
			mu.Unlock()
		}
	}()

	// Coleta informações de disco
	wg.Add(1)
	go func() {
		defer wg.Done()
		diskInfo, err := c.collectDiskInfo(ctx)
		if err == nil {
			mu.Lock()
			hwInfo.Disk = diskInfo
			mu.Unlock()
		}
	}()

	// Coleta informações de rede
	wg.Add(1)
	go func() {
		defer wg.Done()
		netInfo, err := c.collectNetworkInfo(ctx)
		if err == nil {
			mu.Lock()
			hwInfo.Network = netInfo
			mu.Unlock()
		}
	}()

	// Aguarda todas as goroutines terminarem
	wg.Wait()

	// Armazena no cache
	c.setCache("hardware_info", hwInfo)

	return hwInfo, nil
}

// collectCPUInfo coleta informações da CPU
func (c *Collector) collectCPUInfo(ctx context.Context) (*types.CPUInfo, error) {
	// Informações básicas da CPU
	cpuInfos, err := cpu.InfoWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter informações da CPU: %w", err)
	}

	if len(cpuInfos) == 0 {
		return nil, fmt.Errorf("nenhuma informação de CPU encontrada")
	}

	cpuInfo := cpuInfos[0]

	// Uso da CPU
	cpuPercent, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		cpuPercent = []float64{0.0}
	}

	var usage float64
	if len(cpuPercent) > 0 {
		usage = cpuPercent[0]
	}

	// Número de cores lógicos
	logicalCores, err := cpu.CountsWithContext(ctx, true)
	if err != nil {
		logicalCores = runtime.NumCPU()
	}

	// Número de cores físicos
	physicalCores, err := cpu.CountsWithContext(ctx, false)
	if err != nil {
		physicalCores = runtime.NumCPU()
	}

	return &types.CPUInfo{
		ModelName:   cpuInfo.ModelName,
		Cores:       int32(physicalCores),
		Threads:     int32(logicalCores),
		Frequency:   cpuInfo.Mhz,
		Usage:       usage,
		Temperature: 0.0, // Temperatura não disponível via gopsutil
		Timestamp:   time.Now(),
	}, nil
}

// collectMemoryInfo coleta informações de memória
func (c *Collector) collectMemoryInfo(ctx context.Context) (*types.MemoryInfo, error) {
	vmStat, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter informações de memória: %w", err)
	}

	return &types.MemoryInfo{
		Total:       vmStat.Total,
		Available:   vmStat.Available,
		Used:        vmStat.Used,
		UsedPercent: vmStat.UsedPercent,
		Free:        vmStat.Free,
		Timestamp:   time.Now(),
	}, nil
}

// collectDiskInfo coleta informações de disco
func (c *Collector) collectDiskInfo(ctx context.Context) ([]types.DiskInfo, error) {
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter partições: %w", err)
	}

	var diskInfos []types.DiskInfo
	for _, partition := range partitions {
		usage, err := disk.UsageWithContext(ctx, partition.Mountpoint)
		if err != nil {
			// Se não conseguir obter uso, pula esta partição
			continue
		}

		diskInfo := types.DiskInfo{
			Device:      partition.Device,
			Mountpoint:  partition.Mountpoint,
			Fstype:      partition.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
			Timestamp:   time.Now(),
		}

		diskInfos = append(diskInfos, diskInfo)
	}

	return diskInfos, nil
}

// collectNetworkInfo coleta informações de rede
func (c *Collector) collectNetworkInfo(ctx context.Context) ([]types.NetworkInfo, error) {
	interfaces, err := net.InterfacesWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter interfaces de rede: %w", err)
	}

	// Obter estatísticas de rede
	ioCounters, err := net.IOCountersWithContext(ctx, true)
	if err != nil {
		ioCounters = []net.IOCountersStat{}
	}

	// Mapear estatísticas por nome da interface
	statsMap := make(map[string]net.IOCountersStat)
	for _, stat := range ioCounters {
		statsMap[stat.Name] = stat
	}

	var networkInfos []types.NetworkInfo
	for _, iface := range interfaces {
		// Converter endereços da interface
		var addrStrings []string
		for _, addr := range iface.Addrs {
			addrStrings = append(addrStrings, addr.Addr)
		}

		// Obter estatísticas se disponíveis
		var bytesSent, bytesRecv, packetsSent, packetsRecv uint64
		if stats, exists := statsMap[iface.Name]; exists {
			bytesSent = stats.BytesSent
			bytesRecv = stats.BytesRecv
			packetsSent = stats.PacketsSent
			packetsRecv = stats.PacketsRecv
		}

		networkInfo := types.NetworkInfo{
			Name:         iface.Name,
			HardwareAddr: iface.HardwareAddr,
			Flags:        iface.Flags,
			Addrs:        addrStrings,
			BytesSent:    bytesSent,
			BytesRecv:    bytesRecv,
			PacketsSent:  packetsSent,
			PacketsRecv:  packetsRecv,
			Timestamp:    time.Now(),
		}

		networkInfos = append(networkInfos, networkInfo)
	}

	return networkInfos, nil
}

// CollectInventory coleta inventário completo
func (c *Collector) CollectInventory(ctx context.Context, machineID string) (*types.Inventory, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var systemInfo *types.SystemInfo
	var hardwareInfo *types.HardwareInfo
	var collectErrors []error

	// Coleta informações do sistema
	wg.Add(1)
	go func() {
		defer wg.Done()
		sysInfo, err := c.CollectSystemInfo(ctx)
		mu.Lock()
		if err != nil {
			collectErrors = append(collectErrors, fmt.Errorf("erro ao coletar informações do sistema: %w", err))
		} else {
			systemInfo = sysInfo
		}
		mu.Unlock()
	}()

	// Coleta informações de hardware
	wg.Add(1)
	go func() {
		defer wg.Done()
		hwInfo, err := c.CollectHardwareInfo(ctx)
		mu.Lock()
		if err != nil {
			collectErrors = append(collectErrors, fmt.Errorf("erro ao coletar informações de hardware: %w", err))
		} else {
			hardwareInfo = hwInfo
		}
		mu.Unlock()
	}()

	// Aguarda todas as coletas terminarem
	wg.Wait()

	// Se houve erros críticos, retorna erro
	if len(collectErrors) > 0 && (systemInfo == nil || hardwareInfo == nil) {
		return nil, fmt.Errorf("erros críticos na coleta: %v", collectErrors)
	}

	// Cria valores padrão se necessário
	if systemInfo == nil {
		systemInfo = &types.SystemInfo{
			OS:        runtime.GOOS,
			Platform:  runtime.GOARCH,
			Hostname:  "unknown",
			Timestamp: time.Now(),
		}
	}

	if hardwareInfo == nil {
		hardwareInfo = &types.HardwareInfo{
			CPU:       types.CPUInfo{Timestamp: time.Now()},
			Memory:    types.MemoryInfo{Timestamp: time.Now()},
			Disk:      []types.DiskInfo{},
			Network:   []types.NetworkInfo{},
			Timestamp: time.Now(),
		}
	}

	inventory := &types.Inventory{
		MachineID: machineID,
		System:    *systemInfo,
		Hardware:  *hardwareInfo,
		Timestamp: time.Now(),
	}

	return inventory, nil
}

// getFromCache obtém um item do cache se ainda válido
func (c *Collector) getFromCache(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if expiry, exists := c.cacheExpiry[key]; exists {
		if time.Now().Before(expiry) {
			return c.cache[key]
		}
		// Cache expirado, remove
		delete(c.cache, key)
		delete(c.cacheExpiry, key)
	}

	return nil
}

// setCache armazena um item no cache
func (c *Collector) setCache(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = value
	c.cacheExpiry[key] = time.Now().Add(c.cacheTTL)
}

// ClearCache limpa o cache
func (c *Collector) ClearCache() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]interface{})
	c.cacheExpiry = make(map[string]time.Time)
}
