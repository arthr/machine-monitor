package collector

import "time"

// SystemInfo contém informações básicas do sistema
type SystemInfo struct {
	Hostname     string `json:"hostname"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	Uptime       uint64 `json:"uptime"`
	BootTime     uint64 `json:"boot_time"`
	OSVersion    string `json:"os_version"`
	KernelArch   string `json:"kernel_arch"`
	UserCount    int    `json:"user_count"`
}

// HardwareInfo contém informações de hardware
type HardwareInfo struct {
	CPU    CPUInfo    `json:"cpu"`
	Memory MemoryInfo `json:"memory"`
	Disk   []DiskInfo `json:"disk"`
	System struct {
		Manufacturer string `json:"manufacturer"`
		Model        string `json:"model"`
		SerialNumber string `json:"serial_number"`
		UUID         string `json:"uuid"`
	} `json:"system"`
}

// CPUInfo contém informações da CPU
type CPUInfo struct {
	Model       string    `json:"model"`
	Cores       int32     `json:"cores"`
	Threads     int32     `json:"threads"`
	Frequency   float64   `json:"frequency_mhz"`
	Usage       []float64 `json:"usage_percent"`
	Temperature float64   `json:"temperature_celsius,omitempty"`
	CacheSize   int32     `json:"cache_size_kb,omitempty"`
	Vendor      string    `json:"vendor"`
	Family      string    `json:"family"`
}

// MemoryInfo contém informações de memória
type MemoryInfo struct {
	Total       uint64   `json:"total_bytes"`
	Available   uint64   `json:"available_bytes"`
	Used        uint64   `json:"used_bytes"`
	UsedPercent float64  `json:"used_percent"`
	Free        uint64   `json:"free_bytes"`
	Cached      uint64   `json:"cached_bytes,omitempty"`
	Buffers     uint64   `json:"buffers_bytes,omitempty"`
	Swap        SwapInfo `json:"swap"`
}

// SwapInfo contém informações de swap
type SwapInfo struct {
	Total       uint64  `json:"total_bytes"`
	Used        uint64  `json:"used_bytes"`
	Free        uint64  `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

// DiskInfo contém informações de disco
type DiskInfo struct {
	Device      string  `json:"device"`
	Mountpoint  string  `json:"mountpoint"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total_bytes"`
	Free        uint64  `json:"free_bytes"`
	Used        uint64  `json:"used_bytes"`
	UsedPercent float64 `json:"used_percent"`
	Inodes      uint64  `json:"inodes,omitempty"`
	InodesFree  uint64  `json:"inodes_free,omitempty"`
	InodesUsed  uint64  `json:"inodes_used,omitempty"`
}

// SoftwareInfo contém informações de software
type SoftwareInfo struct {
	InstalledApplications []Application `json:"installed_applications"`
	RunningServices       []Service     `json:"running_services"`
	RunningProcesses      []Process     `json:"running_processes"`
	SystemUpdates         []Update      `json:"system_updates,omitempty"`
}

// Application representa uma aplicação instalada
type Application struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Path        string `json:"path"`
	Size        int64  `json:"size_bytes,omitempty"`
	InstallDate string `json:"install_date,omitempty"`
	Vendor      string `json:"vendor,omitempty"`
}

// Service representa um serviço em execução
type Service struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	PID         int32  `json:"pid,omitempty"`
	StartType   string `json:"start_type,omitempty"`
	Description string `json:"description,omitempty"`
}

// Process representa um processo em execução
type Process struct {
	PID         int32   `json:"pid"`
	Name        string  `json:"name"`
	Command     string  `json:"command"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryUsage uint64  `json:"memory_bytes"`
	Status      string  `json:"status"`
	User        string  `json:"user"`
	StartTime   string  `json:"start_time"`
}

// Update representa uma atualização do sistema
type Update struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Size        int64  `json:"size_bytes,omitempty"`
	Type        string `json:"type"`
}

// NetworkInfo contém informações de rede
type NetworkInfo struct {
	Interfaces   []NetworkInterface  `json:"interfaces"`
	Connections  []NetworkConnection `json:"connections,omitempty"`
	Statistics   NetworkStatistics   `json:"statistics"`
	DefaultRoute string              `json:"default_route,omitempty"`
	DNSServers   []string            `json:"dns_servers,omitempty"`
}

// NetworkInterface representa uma interface de rede
type NetworkInterface struct {
	Name         string   `json:"name"`
	HardwareAddr string   `json:"hardware_addr"`
	IPAddresses  []string `json:"ip_addresses"`
	Status       string   `json:"status"`
	MTU          int      `json:"mtu"`
	Speed        uint64   `json:"speed_mbps,omitempty"`
	Type         string   `json:"type"`
	BytesSent    uint64   `json:"bytes_sent"`
	BytesRecv    uint64   `json:"bytes_recv"`
	PacketsSent  uint64   `json:"packets_sent"`
	PacketsRecv  uint64   `json:"packets_recv"`
	Errors       uint64   `json:"errors"`
	Drops        uint64   `json:"drops"`
}

// NetworkConnection representa uma conexão de rede
type NetworkConnection struct {
	LocalAddr  string `json:"local_addr"`
	RemoteAddr string `json:"remote_addr"`
	Status     string `json:"status"`
	PID        int32  `json:"pid"`
	Type       string `json:"type"`
}

// NetworkStatistics contém estatísticas globais de rede
type NetworkStatistics struct {
	TotalBytesSent   uint64 `json:"total_bytes_sent"`
	TotalBytesRecv   uint64 `json:"total_bytes_recv"`
	TotalPacketsSent uint64 `json:"total_packets_sent"`
	TotalPacketsRecv uint64 `json:"total_packets_recv"`
	TotalErrors      uint64 `json:"total_errors"`
	TotalDrops       uint64 `json:"total_drops"`
}

// InventoryData contém todos os dados coletados do sistema
type InventoryData struct {
	MachineID     string       `json:"machine_id"`
	Timestamp     time.Time    `json:"timestamp"`
	CollectedAt   string       `json:"collected_at"`
	System        SystemInfo   `json:"system"`
	Hardware      HardwareInfo `json:"hardware"`
	Software      SoftwareInfo `json:"software"`
	Network       NetworkInfo  `json:"network"`
	MacOSSpecific *MacOSInfo   `json:"macos_specific,omitempty"`
}

// MacOSInfo contém informações específicas do macOS
type MacOSInfo struct {
	SystemProfiler  map[string]interface{} `json:"system_profiler,omitempty"`
	LaunchdServices []LaunchdService       `json:"launchd_services,omitempty"`
	Homebrew        *HomebrewInfo          `json:"homebrew,omitempty"`
	XcodeVersion    string                 `json:"xcode_version,omitempty"`
}

// LaunchdService representa um serviço do launchd
type LaunchdService struct {
	Label  string `json:"label"`
	PID    string `json:"pid"`
	Status string `json:"status"`
}

// HomebrewInfo contém informações do Homebrew
type HomebrewInfo struct {
	Version           string   `json:"version,omitempty"`
	InstalledPackages []string `json:"installed_packages,omitempty"`
	Casks             []string `json:"casks,omitempty"`
}
