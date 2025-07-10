package executor

import (
	"fmt"
	"regexp"
	"strings"
)

// CommandWhitelist define os comandos permitidos e suas restrições
type CommandWhitelist struct {
	Commands map[string]CommandSpec `json:"commands"`
}

// CommandSpec define as especificações de um comando permitido
type CommandSpec struct {
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	AllowedArgs    []string          `json:"allowed_args,omitempty"`
	ForbiddenArgs  []string          `json:"forbidden_args,omitempty"`
	ArgPatterns    map[string]string `json:"arg_patterns,omitempty"`
	MaxArgs        int               `json:"max_args,omitempty"`
	RequiresAuth   bool              `json:"requires_auth,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
	ResourceLimits ResourceLimits    `json:"resource_limits,omitempty"`
	Platform       []string          `json:"platform,omitempty"`
	UserGroups     []string          `json:"user_groups,omitempty"`
}

// ResourceLimits define limites de recursos para execução
type ResourceLimits struct {
	MaxMemoryMB    int `json:"max_memory_mb,omitempty"`
	MaxCPUPercent  int `json:"max_cpu_percent,omitempty"`
	MaxOutputBytes int `json:"max_output_bytes,omitempty"`
}

// GetMacOSWhitelist retorna a whitelist padrão para macOS
func GetMacOSWhitelist() *CommandWhitelist {
	return &CommandWhitelist{
		Commands: map[string]CommandSpec{
			"system_profiler": {
				Name:        "system_profiler",
				Description: "Coleta informações de hardware e software do sistema",
				AllowedArgs: []string{
					"SPHardwareDataType",
					"SPSoftwareDataType",
					"SPApplicationsDataType",
					"SPNetworkDataType",
					"SPStorageDataType",
					"SPMemoryDataType",
					"SPDisplaysDataType",
					"SPUSBDataType",
					"-json",
					"-xml",
				},
				ForbiddenArgs:  []string{"-detailLevel", "full"},
				MaxArgs:        3,
				TimeoutSeconds: 30,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    100,
					MaxOutputBytes: 1024 * 1024, // 1MB
				},
				Platform: []string{"darwin"},
			},
			"launchctl": {
				Name:        "launchctl",
				Description: "Gerencia serviços do sistema",
				AllowedArgs: []string{"list", "print", "print-disabled"},
				ForbiddenArgs: []string{
					"load", "unload", "start", "stop", "enable", "disable",
					"bootstrap", "bootout", "kickstart", "kill",
				},
				MaxArgs:        2,
				TimeoutSeconds: 15,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    50,
					MaxOutputBytes: 512 * 1024, // 512KB
				},
				Platform: []string{"darwin"},
			},
			"ps": {
				Name:        "ps",
				Description: "Lista processos em execução",
				AllowedArgs: []string{"aux", "axo", "pid,ppid,user,command", "-o", "-A", "-e"},
				ForbiddenArgs: []string{
					"-k", "--kill", "-KILL", "-TERM", "-STOP",
				},
				MaxArgs:        4,
				TimeoutSeconds: 10,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    30,
					MaxOutputBytes: 256 * 1024, // 256KB
				},
				Platform: []string{"darwin", "linux"},
			},
			"netstat": {
				Name:           "netstat",
				Description:    "Mostra conexões de rede",
				AllowedArgs:    []string{"-an", "-rn", "-in", "-s", "-p", "tcp", "udp"},
				MaxArgs:        3,
				TimeoutSeconds: 10,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    20,
					MaxOutputBytes: 128 * 1024, // 128KB
				},
				Platform: []string{"darwin", "linux"},
			},
			"ifconfig": {
				Name:        "ifconfig",
				Description: "Mostra configuração de interfaces de rede",
				AllowedArgs: []string{"-a", "-l", "-u"},
				ForbiddenArgs: []string{
					"up", "down", "add", "delete", "netmask", "broadcast",
				},
				MaxArgs:        2,
				TimeoutSeconds: 5,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    10,
					MaxOutputBytes: 64 * 1024, // 64KB
				},
				Platform: []string{"darwin", "linux"},
			},
			"sw_vers": {
				Name:           "sw_vers",
				Description:    "Mostra versão do sistema operacional",
				AllowedArgs:    []string{"-productName", "-productVersion", "-buildVersion"},
				MaxArgs:        1,
				TimeoutSeconds: 5,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    5,
					MaxOutputBytes: 1024, // 1KB
				},
				Platform: []string{"darwin"},
			},
			"diskutil": {
				Name:        "diskutil",
				Description: "Utilitário de gerenciamento de disco",
				AllowedArgs: []string{"list", "info", "activity"},
				ForbiddenArgs: []string{
					"erase", "format", "partitionDisk", "mount", "unmount",
					"repair", "verify", "resetFusion", "coreStorage",
				},
				MaxArgs:        2,
				TimeoutSeconds: 15,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    30,
					MaxOutputBytes: 128 * 1024, // 128KB
				},
				Platform: []string{"darwin"},
			},
			"top": {
				Name:           "top",
				Description:    "Mostra processos em execução",
				AllowedArgs:    []string{"-l", "1", "-n", "-s", "-o", "cpu", "mem"},
				MaxArgs:        4,
				TimeoutSeconds: 10,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    20,
					MaxOutputBytes: 64 * 1024, // 64KB
				},
				Platform: []string{"darwin", "linux"},
			},
			"whoami": {
				Name:           "whoami",
				Description:    "Mostra o usuário atual",
				AllowedArgs:    []string{},
				MaxArgs:        0,
				TimeoutSeconds: 2,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    5,
					MaxOutputBytes: 256,
				},
				Platform: []string{"darwin", "linux"},
			},
			"uname": {
				Name:           "uname",
				Description:    "Mostra informações do sistema",
				AllowedArgs:    []string{"-a", "-s", "-r", "-v", "-m", "-p"},
				MaxArgs:        1,
				TimeoutSeconds: 5,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    5,
					MaxOutputBytes: 1024,
				},
				Platform: []string{"darwin", "linux"},
			},
			"df": {
				Name:           "df",
				Description:    "Mostra uso do sistema de arquivos",
				AllowedArgs:    []string{"-h", "-k", "-m", "-g", "-T"},
				MaxArgs:        2,
				TimeoutSeconds: 5,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    10,
					MaxOutputBytes: 32 * 1024, // 32KB
				},
				Platform: []string{"darwin", "linux"},
			},
			"uptime": {
				Name:           "uptime",
				Description:    "Mostra tempo de atividade do sistema",
				AllowedArgs:    []string{},
				MaxArgs:        0,
				TimeoutSeconds: 2,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    5,
					MaxOutputBytes: 256,
				},
				Platform: []string{"darwin", "linux"},
			},
		},
	}
}

// ValidateCommand valida se um comando é permitido e seus argumentos são válidos
func (w *CommandWhitelist) ValidateCommand(command string, args []string) error {
	spec, exists := w.Commands[command]
	if !exists {
		return fmt.Errorf("comando não permitido: %s", command)
	}

	// Validar número de argumentos
	if spec.MaxArgs > 0 && len(args) > spec.MaxArgs {
		return fmt.Errorf("muitos argumentos para comando %s: máximo %d, recebido %d",
			command, spec.MaxArgs, len(args))
	}

	// Validar argumentos proibidos
	for _, arg := range args {
		for _, forbidden := range spec.ForbiddenArgs {
			if strings.Contains(arg, forbidden) {
				return fmt.Errorf("argumento proibido '%s' para comando %s", arg, command)
			}
		}
	}

	// Validar argumentos permitidos (se especificados)
	if len(spec.AllowedArgs) > 0 {
		for _, arg := range args {
			if !w.isArgAllowed(arg, spec.AllowedArgs) {
				return fmt.Errorf("argumento não permitido '%s' para comando %s", arg, command)
			}
		}
	}

	// Validar padrões de argumentos
	if len(spec.ArgPatterns) > 0 {
		for i, arg := range args {
			if pattern, exists := spec.ArgPatterns[fmt.Sprintf("arg%d", i)]; exists {
				if matched, err := regexp.MatchString(pattern, arg); err != nil || !matched {
					return fmt.Errorf("argumento %d '%s' não corresponde ao padrão esperado para comando %s",
						i, arg, command)
				}
			}
		}
	}

	return nil
}

// isArgAllowed verifica se um argumento está na lista de permitidos
func (w *CommandWhitelist) isArgAllowed(arg string, allowedArgs []string) bool {
	for _, allowed := range allowedArgs {
		if arg == allowed {
			return true
		}
	}
	return false
}

// GetCommandSpec retorna as especificações de um comando
func (w *CommandWhitelist) GetCommandSpec(command string) (CommandSpec, bool) {
	spec, exists := w.Commands[command]
	return spec, exists
}

// SanitizeArguments remove caracteres perigosos dos argumentos
func SanitizeArguments(args []string) []string {
	sanitized := make([]string, len(args))

	// Padrão para caracteres perigosos
	dangerousChars := regexp.MustCompile(`[;&|<>$\x60\\]`)

	for i, arg := range args {
		// Remove caracteres perigosos
		sanitized[i] = dangerousChars.ReplaceAllString(arg, "")

		// Remove espaços extras
		sanitized[i] = strings.TrimSpace(sanitized[i])

		// Limita o tamanho do argumento
		if len(sanitized[i]) > 1000 {
			sanitized[i] = sanitized[i][:1000]
		}
	}

	return sanitized
}

// IsCommandSafe verifica se um comando é considerado seguro
func IsCommandSafe(command string, args []string) bool {
	// Lista de comandos definitivamente perigosos
	dangerousCommands := []string{
		"rm", "rmdir", "dd", "mkfs", "fdisk", "parted",
		"sudo", "su", "chmod", "chown", "chgrp",
		"kill", "killall", "pkill", "reboot", "shutdown", "halt",
		"crontab", "at", "batch", "nohup",
		"nc", "netcat", "telnet", "ssh", "scp", "rsync",
		"curl", "wget", "ftp", "sftp",
		"python", "python3", "perl", "ruby", "node", "php",
		"sh", "bash", "zsh", "csh", "tcsh",
		"vim", "emacs", "nano", "vi",
		"mount", "umount", "fsck", "tune2fs",
	}

	// Verifica se o comando está na lista de perigosos
	for _, dangerous := range dangerousCommands {
		if command == dangerous {
			return false
		}
	}

	// Verifica argumentos perigosos
	for _, arg := range args {
		// Verifica redirecionamento e pipes
		if strings.Contains(arg, ">") || strings.Contains(arg, "<") ||
			strings.Contains(arg, "|") || strings.Contains(arg, "&") {
			return false
		}

		// Verifica tentativas de command injection
		if strings.Contains(arg, ";") || strings.Contains(arg, "&&") ||
			strings.Contains(arg, "||") || strings.Contains(arg, "`") {
			return false
		}

		// Verifica caminhos perigosos
		dangerousPaths := []string{"/etc/", "/var/", "/usr/", "/bin/", "/sbin/"}
		for _, path := range dangerousPaths {
			if strings.HasPrefix(arg, path) {
				return false
			}
		}
	}

	return true
}

// GetWindowsWhitelist retorna a whitelist padrão para Windows
func GetWindowsWhitelist() *CommandWhitelist {
	return &CommandWhitelist{
		Commands: map[string]CommandSpec{
			"systeminfo": {
				Name:           "systeminfo",
				Description:    "Coleta informações do sistema Windows",
				AllowedArgs:    []string{"/fo", "csv", "/fo", "table"},
				MaxArgs:        2,
				TimeoutSeconds: 30,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    100,
					MaxOutputBytes: 1024 * 1024, // 1MB
				},
				Platform: []string{"windows"},
			},
			"tasklist": {
				Name:           "tasklist",
				Description:    "Lista processos em execução",
				AllowedArgs:    []string{"/fo", "csv", "table", "/v", "/svc"},
				ForbiddenArgs:  []string{"/f", "/fi", "/pid", "/im"},
				MaxArgs:        4,
				TimeoutSeconds: 15,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    50,
					MaxOutputBytes: 512 * 1024, // 512KB
				},
				Platform: []string{"windows"},
			},
			"netstat": {
				Name:           "netstat",
				Description:    "Mostra conexões de rede",
				AllowedArgs:    []string{"-an", "-rn", "-s", "-p", "tcp", "udp"},
				MaxArgs:        3,
				TimeoutSeconds: 10,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    20,
					MaxOutputBytes: 128 * 1024, // 128KB
				},
				Platform: []string{"windows", "darwin", "linux"},
			},
			"ipconfig": {
				Name:           "ipconfig",
				Description:    "Mostra configuração de rede",
				AllowedArgs:    []string{"/all", "/displaydns", "/flushdns"},
				ForbiddenArgs:  []string{"/release", "/renew", "/setclassid"},
				MaxArgs:        1,
				TimeoutSeconds: 10,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    10,
					MaxOutputBytes: 64 * 1024, // 64KB
				},
				Platform: []string{"windows"},
			},
			"ver": {
				Name:           "ver",
				Description:    "Mostra versão do Windows",
				AllowedArgs:    []string{},
				MaxArgs:        0,
				TimeoutSeconds: 5,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    5,
					MaxOutputBytes: 1024,
				},
				Platform: []string{"windows"},
			},
			"whoami": {
				Name:           "whoami",
				Description:    "Mostra o usuário atual",
				AllowedArgs:    []string{"/user", "/groups", "/priv"},
				MaxArgs:        1,
				TimeoutSeconds: 5,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    5,
					MaxOutputBytes: 1024,
				},
				Platform: []string{"windows", "darwin", "linux"},
			},
			"wmic": {
				Name:        "wmic",
				Description: "Windows Management Instrumentation",
				AllowedArgs: []string{
					"os", "get", "caption,version,buildnumber",
					"cpu", "get", "name,manufacturer,maxclockspeed",
					"computersystem", "get", "model,manufacturer,totalpysicalmemory",
					"logicaldisk", "get", "size,freespace,caption",
				},
				ForbiddenArgs: []string{
					"process", "call", "create", "terminate",
					"service", "start", "stop", "delete",
					"startup", "add", "remove",
				},
				MaxArgs:        6,
				TimeoutSeconds: 20,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    30,
					MaxOutputBytes: 256 * 1024, // 256KB
				},
				Platform: []string{"windows"},
			},
			"powershell": {
				Name:        "powershell",
				Description: "PowerShell para comandos específicos",
				AllowedArgs: []string{
					"-Command", "Get-ComputerInfo",
					"-Command", "Get-Process",
					"-Command", "Get-Service",
					"-Command", "Get-WmiObject Win32_OperatingSystem",
					"-Command", "Get-WmiObject Win32_ComputerSystem",
				},
				ForbiddenArgs: []string{
					"Remove-", "Delete-", "Stop-", "Restart-",
					"Set-", "New-", "Install-", "Uninstall-",
					"Invoke-", "Start-Process", "Stop-Process",
				},
				MaxArgs:        3,
				TimeoutSeconds: 30,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:    50,
					MaxOutputBytes: 512 * 1024, // 512KB
				},
				Platform: []string{"windows"},
			},
		},
	}
}
