package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"crypto/sha256"
	"encoding/hex"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	"agente-poc/internal/logging"
)

// Collector define a interface para coleta de dados
type Collector interface {
	CollectInventory() (*InventoryData, error)
	CollectBasicInfo() (*SystemInfo, error)
	CollectHardwareInfo() (*HardwareInfo, error)
	CollectSoftwareInfo() (*SoftwareInfo, error)
	CollectNetworkInfo() (*NetworkInfo, error)
	CollectMacOSSpecific() (*MacOSInfo, error)
}

// CollectorConfig contém configurações do collector
type CollectorConfig struct {
	Timeout             time.Duration
	EnableCache         bool
	CacheExpiration     time.Duration
	MaxProcesses        int
	MaxApplications     int
	EnableMacOSSpecific bool
}

// CacheItem representa um item em cache
type CacheItem struct {
	Data      interface{}
	Timestamp time.Time
	TTL       time.Duration
}

// SystemCollector é responsável por coletar dados do sistema
type SystemCollector struct {
	interval time.Duration
	logger   logging.Logger
	config   *CollectorConfig
	cache    map[string]*CacheItem
	cacheMu  sync.RWMutex
}

// New cria uma nova instância do SystemCollector
func New(interval time.Duration, logger logging.Logger) *SystemCollector {
	config := &CollectorConfig{
		Timeout:             30 * time.Second,
		EnableCache:         true,
		CacheExpiration:     5 * time.Minute,
		MaxProcesses:        100,
		MaxApplications:     200,
		EnableMacOSSpecific: runtime.GOOS == "darwin",
	}

	return &SystemCollector{
		interval: interval,
		logger:   logger,
		config:   config,
		cache:    make(map[string]*CacheItem),
	}
}

// CollectInventory coleta informações completas do sistema
func (c *SystemCollector) CollectInventory() (*InventoryData, error) {
	c.logger.Debug("Collecting system inventory...")

	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	// Coletar dados em paralelo
	var wg sync.WaitGroup
	var mu sync.Mutex

	var systemInfo *SystemInfo
	var hardwareInfo *HardwareInfo
	var softwareInfo *SoftwareInfo
	var networkInfo *NetworkInfo
	var macOSInfo *MacOSInfo
	var lastError error

	// Função auxiliar para capturar erros
	setError := func(err error) {
		mu.Lock()
		if lastError == nil {
			lastError = err
		}
		mu.Unlock()
	}

	// Coleta de informações básicas do sistema
	wg.Add(1)
	go func() {
		defer wg.Done()
		if info, err := c.collectSystemInfoInternal(ctx); err != nil {
			setError(fmt.Errorf("failed to collect system info: %w", err))
		} else {
			systemInfo = info
		}
	}()

	// Coleta de informações de hardware
	wg.Add(1)
	go func() {
		defer wg.Done()
		if info, err := c.collectHardwareInfoInternal(ctx); err != nil {
			setError(fmt.Errorf("failed to collect hardware info: %w", err))
		} else {
			hardwareInfo = info
		}
	}()

	// Coleta de informações de software
	wg.Add(1)
	go func() {
		defer wg.Done()
		if info, err := c.collectSoftwareInfoInternal(ctx); err != nil {
			setError(fmt.Errorf("failed to collect software info: %w", err))
		} else {
			softwareInfo = info
		}
	}()

	// Coleta de informações de rede
	wg.Add(1)
	go func() {
		defer wg.Done()
		if info, err := c.collectNetworkInfoInternal(ctx); err != nil {
			setError(fmt.Errorf("failed to collect network info: %w", err))
		} else {
			networkInfo = info
		}
	}()

	// Coleta de informações específicas do macOS
	if c.config.EnableMacOSSpecific {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if info, err := c.collectMacOSSpecificInternal(ctx); err != nil {
				c.logger.WithField("error", err).Warning("Failed to collect macOS specific info")
			} else {
				macOSInfo = info
			}
		}()
	}

	wg.Wait()

	// Retornar erro se alguma coleta crítica falhou
	if lastError != nil {
		return nil, lastError
	}

	// Gerar Machine ID
	machineID, err := c.generateMachineID(ctx)
	if err != nil {
		c.logger.WithField("error", err).Warning("Failed to generate machine ID, using fallback")
		// Usar hostname como fallback
		if hostInfo, err := host.InfoWithContext(ctx); err == nil {
			machineID = fmt.Sprintf("fallback-%s", hostInfo.Hostname)
		} else {
			machineID = "fallback-unknown"
		}
	}

	// Construir dados de inventário
	inventory := &InventoryData{
		MachineID:     machineID,
		Timestamp:     time.Now(),
		CollectedAt:   time.Now().Format(time.RFC3339),
		System:        *systemInfo,
		Hardware:      *hardwareInfo,
		Software:      *softwareInfo,
		Network:       *networkInfo,
		MacOSSpecific: macOSInfo,
	}

	c.logger.Debug("System inventory collected successfully")
	return inventory, nil
}

// CollectBasicInfo coleta informações básicas do sistema
func (c *SystemCollector) CollectBasicInfo() (*SystemInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	return c.collectSystemInfoInternal(ctx)
}

// CollectHardwareInfo coleta informações de hardware
func (c *SystemCollector) CollectHardwareInfo() (*HardwareInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	return c.collectHardwareInfoInternal(ctx)
}

// CollectSoftwareInfo coleta informações de software
func (c *SystemCollector) CollectSoftwareInfo() (*SoftwareInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	return c.collectSoftwareInfoInternal(ctx)
}

// CollectNetworkInfo coleta informações de rede
func (c *SystemCollector) CollectNetworkInfo() (*NetworkInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	return c.collectNetworkInfoInternal(ctx)
}

// CollectMacOSSpecific coleta informações específicas do macOS
func (c *SystemCollector) CollectMacOSSpecific() (*MacOSInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	return c.collectMacOSSpecificInternal(ctx)
}

// collectSystemInfoInternal coleta informações básicas do sistema
func (c *SystemCollector) collectSystemInfoInternal(ctx context.Context) (*SystemInfo, error) {
	// Tentar obter do cache primeiro
	if cachedData := c.getFromCache("system_info"); cachedData != nil {
		if info, ok := cachedData.(*SystemInfo); ok {
			return info, nil
		}
	}

	c.logger.Debug("Collecting system info...")

	// Coletar informações do host
	hostInfo, err := host.InfoWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get host info: %w", err)
	}

	// Coletar informações de usuários
	users, err := host.UsersWithContext(ctx)
	if err != nil {
		c.logger.WithField("error", err).Warning("Failed to get users info")
		users = []host.UserStat{} // Continuar sem informações de usuários
	}

	info := &SystemInfo{
		Hostname:     hostInfo.Hostname,
		Platform:     hostInfo.Platform,
		OSVersion:    hostInfo.PlatformVersion,
		Architecture: hostInfo.KernelArch,
		KernelArch:   hostInfo.KernelArch,
		Uptime:       hostInfo.Uptime,
		BootTime:     hostInfo.BootTime,
		UserCount:    len(users),
	}

	// Cachear o resultado
	c.setInCache("system_info", info, c.config.CacheExpiration)

	return info, nil
}

// collectHardwareInfoInternal coleta informações de hardware
func (c *SystemCollector) collectHardwareInfoInternal(ctx context.Context) (*HardwareInfo, error) {
	c.logger.Debug("Collecting hardware info...")

	var wg sync.WaitGroup
	var mu sync.Mutex
	var lastError error

	// Função auxiliar para capturar erros
	setError := func(err error) {
		mu.Lock()
		if lastError == nil {
			lastError = err
		}
		mu.Unlock()
	}

	// Estrutura para armazenar resultados
	hardwareInfo := &HardwareInfo{}

	// Coleta de informações de CPU
	wg.Add(1)
	go func() {
		defer wg.Done()
		if cpuInfo, err := c.collectCPUInfo(ctx); err != nil {
			setError(fmt.Errorf("failed to collect CPU info: %w", err))
		} else {
			mu.Lock()
			hardwareInfo.CPU = *cpuInfo
			mu.Unlock()
		}
	}()

	// Coleta de informações de memória
	wg.Add(1)
	go func() {
		defer wg.Done()
		if memInfo, err := c.collectMemoryInfo(ctx); err != nil {
			setError(fmt.Errorf("failed to collect memory info: %w", err))
		} else {
			mu.Lock()
			hardwareInfo.Memory = *memInfo
			mu.Unlock()
		}
	}()

	// Coleta de informações de disco
	wg.Add(1)
	go func() {
		defer wg.Done()
		if diskInfo, err := c.collectDiskInfo(ctx); err != nil {
			setError(fmt.Errorf("failed to collect disk info: %w", err))
		} else {
			mu.Lock()
			hardwareInfo.Disk = diskInfo
			mu.Unlock()
		}
	}()

	wg.Wait()

	if lastError != nil {
		return nil, lastError
	}

	return hardwareInfo, nil
}

// collectCPUInfo coleta informações da CPU
func (c *SystemCollector) collectCPUInfo(ctx context.Context) (*CPUInfo, error) {
	// Informações estáticas da CPU
	cpuInfos, err := cpu.InfoWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}

	if len(cpuInfos) == 0 {
		return nil, fmt.Errorf("no CPU info available")
	}

	cpuInfo := cpuInfos[0]

	// Uso da CPU
	cpuPercent, err := cpu.PercentWithContext(ctx, time.Second, true)
	if err != nil {
		c.logger.WithField("error", err).Warning("Failed to get CPU usage")
		cpuPercent = []float64{0.0} // Valor padrão
	}

	return &CPUInfo{
		Model:     cpuInfo.ModelName,
		Vendor:    cpuInfo.VendorID,
		Family:    cpuInfo.Family,
		Cores:     cpuInfo.Cores,
		Threads:   cpuInfo.Cores, // Assumindo sem hyperthreading
		Frequency: cpuInfo.Mhz,
		CacheSize: cpuInfo.CacheSize,
		Usage:     cpuPercent,
	}, nil
}

// collectMemoryInfo coleta informações de memória
func (c *SystemCollector) collectMemoryInfo(ctx context.Context) (*MemoryInfo, error) {
	// Memória virtual
	vmem, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory info: %w", err)
	}

	// Memória swap
	swap, err := mem.SwapMemoryWithContext(ctx)
	if err != nil {
		c.logger.WithField("error", err).Warning("Failed to get swap memory info")
		swap = &mem.SwapMemoryStat{} // Valor padrão
	}

	return &MemoryInfo{
		Total:       vmem.Total,
		Available:   vmem.Available,
		Used:        vmem.Used,
		UsedPercent: vmem.UsedPercent,
		Free:        vmem.Free,
		Cached:      vmem.Cached,
		Buffers:     vmem.Buffers,
		Swap: SwapInfo{
			Total:       swap.Total,
			Used:        swap.Used,
			Free:        swap.Free,
			UsedPercent: swap.UsedPercent,
		},
	}, nil
}

// collectDiskInfo coleta informações de disco
func (c *SystemCollector) collectDiskInfo(ctx context.Context) ([]DiskInfo, error) {
	// Obter partições
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk partitions: %w", err)
	}

	var diskInfos []DiskInfo

	for _, partition := range partitions {
		// Obter uso da partição
		usage, err := disk.UsageWithContext(ctx, partition.Mountpoint)
		if err != nil {
			c.logger.WithFields(map[string]interface{}{
				"partition": partition.Mountpoint,
				"error":     err,
			}).Warning("Failed to get disk usage")
			continue
		}

		diskInfo := DiskInfo{
			Device:      partition.Device,
			Mountpoint:  partition.Mountpoint,
			Fstype:      partition.Fstype,
			Total:       usage.Total,
			Free:        usage.Free,
			Used:        usage.Used,
			UsedPercent: usage.UsedPercent,
			Inodes:      usage.InodesTotal,
			InodesFree:  usage.InodesFree,
			InodesUsed:  usage.InodesUsed,
		}

		diskInfos = append(diskInfos, diskInfo)
	}

	return diskInfos, nil
}

// collectSoftwareInfoInternal coleta informações de software
func (c *SystemCollector) collectSoftwareInfoInternal(ctx context.Context) (*SoftwareInfo, error) {
	c.logger.Debug("Collecting software info...")

	var wg sync.WaitGroup
	var mu sync.Mutex
	var lastError error

	// Função auxiliar para capturar erros
	setError := func(err error) {
		mu.Lock()
		if lastError == nil {
			lastError = err
		}
		mu.Unlock()
	}

	// Estrutura para armazenar resultados
	softwareInfo := &SoftwareInfo{}

	// Coleta de aplicações instaladas
	wg.Add(1)
	go func() {
		defer wg.Done()
		if apps, err := c.collectInstalledApps(ctx); err != nil {
			setError(fmt.Errorf("failed to collect installed apps: %w", err))
		} else {
			mu.Lock()
			softwareInfo.InstalledApplications = apps
			mu.Unlock()
		}
	}()

	// Coleta de processos em execução
	wg.Add(1)
	go func() {
		defer wg.Done()
		if processes, err := c.collectRunningProcesses(ctx); err != nil {
			setError(fmt.Errorf("failed to collect running processes: %w", err))
		} else {
			mu.Lock()
			softwareInfo.RunningProcesses = processes
			mu.Unlock()
		}
	}()

	// Coleta de serviços em execução
	wg.Add(1)
	go func() {
		defer wg.Done()
		if services, err := c.collectRunningServices(ctx); err != nil {
			c.logger.WithField("error", err).Warning("Failed to collect running services")
			mu.Lock()
			softwareInfo.RunningServices = []Service{} // Valor padrão
			mu.Unlock()
		} else {
			mu.Lock()
			softwareInfo.RunningServices = services
			mu.Unlock()
		}
	}()

	wg.Wait()

	if lastError != nil {
		return nil, lastError
	}

	return softwareInfo, nil
}

// collectInstalledApps coleta aplicações instaladas
func (c *SystemCollector) collectInstalledApps(ctx context.Context) ([]Application, error) {
	// Tentar obter do cache primeiro
	if cachedData := c.getFromCache("installed_apps"); cachedData != nil {
		if apps, ok := cachedData.([]Application); ok {
			return apps, nil
		}
	}

	c.logger.Debug("Collecting installed applications...")

	var apps []Application
	applicationsPath := "/Applications"

	// Listar aplicações em /Applications
	err := filepath.WalkDir(applicationsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continuar mesmo com erros
		}

		// Verificar se é um .app
		if d.IsDir() && strings.HasSuffix(path, ".app") {
			appInfo, err := c.getAppInfo(path)
			if err != nil {
				c.logger.WithFields(map[string]interface{}{
					"path":  path,
					"error": err,
				}).Debug("Failed to get app info")
				return nil // Continuar com outras aplicações
			}

			apps = append(apps, *appInfo)

			// Limitar número de aplicações
			if len(apps) >= c.config.MaxApplications {
				return fmt.Errorf("max applications reached") // Parar a caminhada
			}
		}

		return nil
	})

	if err != nil && !strings.Contains(err.Error(), "max applications reached") {
		return nil, fmt.Errorf("failed to walk applications directory: %w", err)
	}

	// Cachear o resultado
	c.setInCache("installed_apps", apps, c.config.CacheExpiration)

	return apps, nil
}

// getAppInfo obtém informações de uma aplicação
func (c *SystemCollector) getAppInfo(appPath string) (*Application, error) {
	info, err := os.Stat(appPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat app: %w", err)
	}

	app := &Application{
		Name:        strings.TrimSuffix(filepath.Base(appPath), ".app"),
		Path:        appPath,
		Size:        0, // Calcular tamanho pode ser custoso
		InstallDate: info.ModTime().Format(time.RFC3339),
	}

	// Tentar obter informações do Info.plist
	plistPath := filepath.Join(appPath, "Contents", "Info.plist")
	if plistInfo, err := c.parseInfoPlist(plistPath); err == nil {
		if version, ok := plistInfo["CFBundleShortVersionString"].(string); ok {
			app.Version = version
		}
		if vendor, ok := plistInfo["CFBundleIdentifier"].(string); ok {
			app.Vendor = vendor
		}
	}

	return app, nil
}

// parseInfoPlist parse básico do Info.plist (simplificado)
func (c *SystemCollector) parseInfoPlist(path string) (map[string]interface{}, error) {
	// Implementação simplificada - na prática, usaria uma biblioteca plist
	// Por ora, retornar vazio
	return map[string]interface{}{}, nil
}

// collectRunningProcesses coleta processos em execução
func (c *SystemCollector) collectRunningProcesses(ctx context.Context) ([]Process, error) {
	c.logger.Debug("Collecting running processes...")

	// Obter lista de PIDs
	pids, err := process.PidsWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get process PIDs: %w", err)
	}

	var processes []Process
	count := 0

	for _, pid := range pids {
		if count >= c.config.MaxProcesses {
			break
		}

		proc, err := process.NewProcessWithContext(ctx, pid)
		if err != nil {
			continue // Processo pode ter terminado
		}

		processInfo, err := c.getProcessInfo(ctx, proc)
		if err != nil {
			continue // Continuar com outros processos
		}

		processes = append(processes, *processInfo)
		count++
	}

	return processes, nil
}

// getProcessInfo obtém informações de um processo
func (c *SystemCollector) getProcessInfo(ctx context.Context, proc *process.Process) (*Process, error) {
	name, err := proc.NameWithContext(ctx)
	if err != nil {
		name = "unknown"
	}

	cmdline, err := proc.CmdlineWithContext(ctx)
	if err != nil {
		cmdline = ""
	}

	cpuPercent, err := proc.CPUPercentWithContext(ctx)
	if err != nil {
		cpuPercent = 0.0
	}

	memInfo, err := proc.MemoryInfoWithContext(ctx)
	var memoryUsage uint64
	if err == nil {
		memoryUsage = memInfo.RSS
	}

	statusList, err := proc.StatusWithContext(ctx)
	var status string
	if err != nil || len(statusList) == 0 {
		status = "unknown"
	} else {
		status = statusList[0] // Usar o primeiro status da lista
	}

	username, err := proc.UsernameWithContext(ctx)
	if err != nil {
		username = "unknown"
	}

	createTime, err := proc.CreateTimeWithContext(ctx)
	var startTime string
	if err == nil {
		startTime = time.Unix(createTime/1000, 0).Format(time.RFC3339)
	}

	return &Process{
		PID:         proc.Pid,
		Name:        name,
		Command:     cmdline,
		CPUPercent:  cpuPercent,
		MemoryUsage: memoryUsage,
		Status:      status,
		User:        username,
		StartTime:   startTime,
	}, nil
}

// collectRunningServices coleta serviços em execução (específico do macOS)
func (c *SystemCollector) collectRunningServices(ctx context.Context) ([]Service, error) {
	c.logger.Debug("Collecting running services...")

	// Executar launchctl list
	cmd := exec.CommandContext(ctx, "launchctl", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute launchctl: %w", err)
	}

	var services []Service
	lines := strings.Split(string(output), "\n")

	for _, line := range lines[1:] { // Pular cabeçalho
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		pid := fields[0]
		status := fields[1]
		name := fields[2]

		// Converter PID
		pidInt, err := strconv.Atoi(pid)
		var pidInt32 int32
		if err == nil {
			pidInt32 = int32(pidInt)
		}

		service := Service{
			Name:   name,
			Status: status,
			PID:    pidInt32,
		}

		services = append(services, service)
	}

	return services, nil
}

// collectNetworkInfoInternal coleta informações de rede
func (c *SystemCollector) collectNetworkInfoInternal(ctx context.Context) (*NetworkInfo, error) {
	c.logger.Debug("Collecting network info...")

	// Obter interfaces de rede
	interfaces, err := net.InterfacesWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var networkInterfaces []NetworkInterface
	var totalBytesSent, totalBytesRecv uint64

	for _, iface := range interfaces {
		// Obter estatísticas da interface
		stats, err := net.IOCountersWithContext(ctx, false)
		if err != nil {
			c.logger.WithField("error", err).Warning("Failed to get network IO counters")
			continue
		}

		var ifaceStats *net.IOCountersStat
		for _, stat := range stats {
			if stat.Name == iface.Name {
				ifaceStats = &stat
				break
			}
		}

		if ifaceStats == nil {
			continue
		}

		networkInterface := NetworkInterface{
			Name:         iface.Name,
			HardwareAddr: iface.HardwareAddr,
			MTU:          iface.MTU,
			Status:       "up", // Simplificado
			BytesSent:    ifaceStats.BytesSent,
			BytesRecv:    ifaceStats.BytesRecv,
			PacketsSent:  ifaceStats.PacketsSent,
			PacketsRecv:  ifaceStats.PacketsRecv,
			Errors:       ifaceStats.Errin + ifaceStats.Errout,
			Drops:        ifaceStats.Dropin + ifaceStats.Dropout,
		}

		// Adicionar endereços IP
		for _, addr := range iface.Addrs {
			networkInterface.IPAddresses = append(networkInterface.IPAddresses, addr.Addr)
		}

		networkInterfaces = append(networkInterfaces, networkInterface)

		// Somar para estatísticas globais
		totalBytesSent += ifaceStats.BytesSent
		totalBytesRecv += ifaceStats.BytesRecv
	}

	return &NetworkInfo{
		Interfaces: networkInterfaces,
		Statistics: NetworkStatistics{
			TotalBytesSent: totalBytesSent,
			TotalBytesRecv: totalBytesRecv,
		},
	}, nil
}

// collectMacOSSpecificInternal coleta informações específicas do macOS
func (c *SystemCollector) collectMacOSSpecificInternal(ctx context.Context) (*MacOSInfo, error) {
	c.logger.Debug("Collecting macOS specific info...")

	macOSInfo := &MacOSInfo{}

	// Obter informações do system_profiler
	if systemProfiler, err := c.getSystemProfiler(ctx); err == nil {
		macOSInfo.SystemProfiler = systemProfiler
	}

	// Obter serviços do launchd
	if launchdServices, err := c.getLaunchdServices(ctx); err == nil {
		macOSInfo.LaunchdServices = launchdServices
	}

	// Obter informações do Homebrew
	if homebrewInfo, err := c.getHomebrewInfo(ctx); err == nil {
		macOSInfo.Homebrew = homebrewInfo
	}

	// Obter versão do Xcode
	if xcodeVersion, err := c.getXcodeVersion(ctx); err == nil {
		macOSInfo.XcodeVersion = xcodeVersion
	}

	return macOSInfo, nil
}

// getSystemProfiler obtém informações do system_profiler
func (c *SystemCollector) getSystemProfiler(ctx context.Context) (map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, "system_profiler", "SPHardwareDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute system_profiler: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse system_profiler output: %w", err)
	}

	return result, nil
}

// getLaunchdServices obtém serviços do launchd
func (c *SystemCollector) getLaunchdServices(ctx context.Context) ([]LaunchdService, error) {
	cmd := exec.CommandContext(ctx, "launchctl", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute launchctl: %w", err)
	}

	var services []LaunchdService
	lines := strings.Split(string(output), "\n")

	for _, line := range lines[1:] { // Pular cabeçalho
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		service := LaunchdService{
			PID:    fields[0],
			Status: fields[1],
			Label:  fields[2],
		}

		services = append(services, service)
	}

	return services, nil
}

// getHomebrewInfo obtém informações do Homebrew
func (c *SystemCollector) getHomebrewInfo(ctx context.Context) (*HomebrewInfo, error) {
	// Verificar se o Homebrew está instalado
	cmd := exec.CommandContext(ctx, "brew", "--version")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("homebrew not installed: %w", err)
	}

	version := strings.TrimSpace(string(output))

	// Listar pacotes instalados
	cmd = exec.CommandContext(ctx, "brew", "list")
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list brew packages: %w", err)
	}

	packages := strings.Fields(string(output))

	return &HomebrewInfo{
		Version:           version,
		InstalledPackages: packages,
	}, nil
}

// getXcodeVersion obtém versão do Xcode
func (c *SystemCollector) getXcodeVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "xcodebuild", "-version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Xcode version: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}

	return "", fmt.Errorf("no Xcode version found")
}

// getFromCache obtém dados do cache
func (c *SystemCollector) getFromCache(key string) interface{} {
	if !c.config.EnableCache {
		return nil
	}

	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	item, exists := c.cache[key]
	if !exists {
		return nil
	}

	// Verificar se expirou
	if time.Since(item.Timestamp) > item.TTL {
		// Remover item expirado
		delete(c.cache, key)
		return nil
	}

	return item.Data
}

// setInCache armazena dados no cache
func (c *SystemCollector) setInCache(key string, data interface{}, ttl time.Duration) {
	if !c.config.EnableCache {
		return
	}

	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	c.cache[key] = &CacheItem{
		Data:      data,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
}

// ClearCache limpa o cache
func (c *SystemCollector) ClearCache() {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	c.cache = make(map[string]*CacheItem)
	c.logger.Debug("Cache cleared")
}

// GetCacheStats retorna estatísticas do cache
func (c *SystemCollector) GetCacheStats() map[string]interface{} {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	stats := map[string]interface{}{
		"enabled": c.config.EnableCache,
		"items":   len(c.cache),
	}

	var expired int
	for _, item := range c.cache {
		if time.Since(item.Timestamp) > item.TTL {
			expired++
		}
	}

	stats["expired"] = expired

	return stats
}

// generateMachineID gera um identificador único para a máquina
func (c *SystemCollector) generateMachineID(ctx context.Context) (string, error) {
	// Tentar obter do cache primeiro (cache persistente)
	if cachedData := c.getFromCache("machine_id"); cachedData != nil {
		if machineID, ok := cachedData.(string); ok && machineID != "" {
			return machineID, nil
		}
	}

	c.logger.Debug("Generating machine ID...")

	// Método 1: Hardware UUID via system_profiler
	if machineID, err := c.getMachineIDFromSystemProfiler(ctx); err == nil && machineID != "" {
		// Cachear por 24 horas (não deve mudar)
		c.setInCache("machine_id", machineID, 24*time.Hour)
		return machineID, nil
	}

	// Método 2: Hardware UUID via ioreg
	if machineID, err := c.getMachineIDFromIOReg(ctx); err == nil && machineID != "" {
		c.setInCache("machine_id", machineID, 24*time.Hour)
		return machineID, nil
	}

	// Método 3: Fallback - combinação de características únicas
	if machineID, err := c.generateFallbackMachineID(ctx); err == nil && machineID != "" {
		c.setInCache("machine_id", machineID, 24*time.Hour)
		return machineID, nil
	}

	return "", fmt.Errorf("failed to generate machine ID using all methods")
}

// getMachineIDFromSystemProfiler obtém UUID do hardware via system_profiler
func (c *SystemCollector) getMachineIDFromSystemProfiler(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "system_profiler", "SPHardwareDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute system_profiler: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse system_profiler output: %w", err)
	}

	// Navegar na estrutura JSON para encontrar o UUID
	if spHardwareData, ok := result["SPHardwareDataType"].([]interface{}); ok && len(spHardwareData) > 0 {
		if hardwareData, ok := spHardwareData[0].(map[string]interface{}); ok {
			if platformUUID, ok := hardwareData["platform_UUID"].(string); ok {
				return platformUUID, nil
			}
		}
	}

	return "", fmt.Errorf("UUID not found in system_profiler output")
}

// getMachineIDFromIOReg obtém UUID do hardware via ioreg
func (c *SystemCollector) getMachineIDFromIOReg(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute ioreg: %w", err)
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		if strings.Contains(line, "IOPlatformUUID") {
			// Extrair UUID da linha: "IOPlatformUUID" = "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				uuid := strings.TrimSpace(parts[1])
				uuid = strings.Trim(uuid, "\"")
				if uuid != "" {
					return uuid, nil
				}
			}
		}
	}

	return "", fmt.Errorf("UUID not found in ioreg output")
}

// generateFallbackMachineID gera ID baseado em características únicas do sistema
func (c *SystemCollector) generateFallbackMachineID(ctx context.Context) (string, error) {
	var components []string

	// Adicionar hostname
	if hostInfo, err := host.InfoWithContext(ctx); err == nil {
		components = append(components, hostInfo.Hostname)
	}

	// Adicionar MAC addresses das interfaces de rede
	if interfaces, err := net.InterfacesWithContext(ctx); err == nil {
		for _, iface := range interfaces {
			if iface.HardwareAddr != "" && iface.HardwareAddr != "00:00:00:00:00:00" {
				components = append(components, iface.HardwareAddr)
			}
		}
	}

	// Adicionar informações da CPU
	if cpuInfos, err := cpu.InfoWithContext(ctx); err == nil && len(cpuInfos) > 0 {
		cpuInfo := cpuInfos[0]
		if cpuInfo.ModelName != "" {
			components = append(components, cpuInfo.ModelName)
		}
	}

	// Adicionar informações de memória total
	if memInfo, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		components = append(components, fmt.Sprintf("mem_%d", memInfo.Total))
	}

	if len(components) == 0 {
		return "", fmt.Errorf("no unique components found for fallback machine ID")
	}

	// Combinar todos os componentes e gerar hash
	combined := strings.Join(components, "|")
	hasher := sha256.New()
	hasher.Write([]byte(combined))
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Retornar um UUID-like formato baseado no hash
	if len(hash) >= 32 {
		return fmt.Sprintf("%s-%s-%s-%s-%s",
			hash[0:8],
			hash[8:12],
			hash[12:16],
			hash[16:20],
			hash[20:32],
		), nil
	}

	return hash, nil
}
