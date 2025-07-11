# Task 01: Criar Interfaces Multiplataforma

## 📋 Objetivo
Definir interfaces claras para separar o código específico de cada plataforma, permitindo implementações distintas para macOS, Windows e Linux.

## 🎯 Entregáveis
- [ ] Interface `PlatformCollector` definida
- [ ] Interface `PlatformInfo` padronizada
- [ ] Estruturas de dados genéricas
- [ ] Documentação das interfaces

## 📊 Contexto
Atualmente o código está misturado com comandos específicos do macOS. Precisamos criar uma abstração que permita implementações específicas por plataforma mantendo uma API unificada.

## 🔧 Implementação

### 1. Criar `internal/collector/interfaces.go`
```go
package collector

import (
    "context"
    "time"
)

// PlatformCollector define a interface para coleta específica por plataforma
type PlatformCollector interface {
    // Informações básicas da plataforma
    GetPlatformInfo(ctx context.Context) (*PlatformInfo, error)
    
    // Machine ID único por plataforma
    GetMachineID(ctx context.Context) (string, error)
    
    // Descoberta de aplicações instaladas
    CollectInstalledApps(ctx context.Context) ([]Application, error)
    
    // Serviços do sistema
    CollectSystemServices(ctx context.Context) ([]Service, error)
    
    // Informações específicas da plataforma
    CollectPlatformSpecific(ctx context.Context) (map[string]interface{}, error)
}

// PlatformInfo contém informações básicas da plataforma
type PlatformInfo struct {
    OS           string                 `json:"os"`
    Architecture string                 `json:"architecture"`
    Version      string                 `json:"version"`
    Hostname     string                 `json:"hostname"`
    Uptime       time.Duration          `json:"uptime"`
    Platform     string                 `json:"platform"`
    Specific     map[string]interface{} `json:"specific,omitempty"`
}

// Application representa uma aplicação instalada
type Application struct {
    Name        string    `json:"name"`
    Version     string    `json:"version,omitempty"`
    Vendor      string    `json:"vendor,omitempty"`
    Path        string    `json:"path,omitempty"`
    InstallDate string    `json:"install_date,omitempty"`
    Size        int64     `json:"size,omitempty"`
    Type        string    `json:"type,omitempty"` // "system", "user", "store"
}

// Service representa um serviço do sistema
type Service struct {
    Name        string `json:"name"`
    DisplayName string `json:"display_name,omitempty"`
    Status      string `json:"status"`
    StartType   string `json:"start_type,omitempty"`
    ProcessID   int    `json:"process_id,omitempty"`
    Path        string `json:"path,omitempty"`
    Description string `json:"description,omitempty"`
}
```

### 2. Atualizar `internal/collector/types.go`
```go
// Adicionar novos campos às estruturas existentes
type SystemInfo struct {
    // ... campos existentes ...
    
    // Novos campos multiplataforma
    Platform     *PlatformInfo          `json:"platform"`
    Applications []Application          `json:"applications,omitempty"`
    Services     []Service             `json:"services,omitempty"`
    Specific     map[string]interface{} `json:"platform_specific,omitempty"`
}
```

### 3. Criar `internal/collector/common.go`
```go
package collector

import (
    "context"
    "runtime"
    "time"
    
    "github.com/shirou/gopsutil/v3/host"
)

// GetBasicPlatformInfo coleta informações básicas usando gopsutil
func GetBasicPlatformInfo(ctx context.Context) (*PlatformInfo, error) {
    info, err := host.InfoWithContext(ctx)
    if err != nil {
        return nil, err
    }
    
    return &PlatformInfo{
        OS:           runtime.GOOS,
        Architecture: runtime.GOARCH,
        Version:      info.KernelVersion,
        Hostname:     info.Hostname,
        Uptime:       time.Duration(info.Uptime) * time.Second,
        Platform:     info.Platform,
    }, nil
}

// SanitizeApplicationName limpa nomes de aplicações
func SanitizeApplicationName(name string) string {
    // Remove caracteres especiais e normaliza
    // Implementar lógica de sanitização
    return name
}

// ValidateService valida dados de serviço
func ValidateService(service *Service) bool {
    return service.Name != "" && service.Status != ""
}
```

## 📋 Checklist de Implementação

### Arquivos a Criar
- [ ] `internal/collector/interfaces.go` - Interfaces principais
- [ ] `internal/collector/common.go` - Funções compartilhadas

### Arquivos a Modificar
- [ ] `internal/collector/types.go` - Adicionar novos campos
- [ ] `internal/collector/collector.go` - Preparar para usar interfaces

### Validações
- [ ] Interfaces compilam sem erros
- [ ] Estruturas de dados são consistentes
- [ ] Funções comuns funcionam em todas as plataformas
- [ ] Documentação das interfaces está clara

## 🎯 Critérios de Sucesso
- [ ] Interfaces bem definidas e documentadas
- [ ] Estruturas de dados padronizadas
- [ ] Código comum separado do específico
- [ ] Base preparada para implementações específicas

## 📚 Referências
- [Go Interfaces](https://tour.golang.org/methods/9) - Documentação oficial
- [gopsutil host](https://pkg.go.dev/github.com/shirou/gopsutil/v3/host) - Informações do sistema
- [Design Patterns](https://refactoring.guru/design-patterns/strategy) - Strategy pattern

## ⏭️ Próxima Task
[02-build-tags-refactor.md](02-build-tags-refactor.md) - Implementar build tags para compilação condicional 