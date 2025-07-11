# Machine Monitor Agent - Makefile

# Variáveis
BINARY_NAME=machine-monitor-agent
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME).exe
BINARY_DARWIN=$(BINARY_NAME)_darwin

VERSION=1.0.0
BUILD_TIME=$(shell date +%Y%m%d_%H%M%S)
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# Comandos padrão
.PHONY: all build clean test deps run install uninstall

all: clean deps test build

# Compilação
build:
	go build ${LDFLAGS} -o ${BINARY_NAME} cmd/main.go

# Compilação para diferentes plataformas
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags "linux" -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)" -o $(BINARY_NAME)_linux cmd/main.go

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)" -o $(BINARY_NAME)_windows.exe cmd/main.go

build-darwin:
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_DARWIN} cmd/main.go

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_DARWIN}_arm64 cmd/main.go

# Compilação para todas as plataformas
build-all: build-linux build-windows build-darwin build-darwin-arm64

# Dependências
deps:
	go mod download
	go mod tidy

# Testes
test:
	go test -v ./...

# Executar em modo console
run:
	go run cmd/main.go -console -config config.json

# Executar em modo console com debug
run-debug:
	go run cmd/main.go -console -config config.json

# Instalar como serviço
install:
	sudo ./$(BINARY_NAME) -install -config $(shell pwd)/config.json

# Desinstalar serviço
uninstall:
	sudo ./$(BINARY_NAME) -uninstall

# Iniciar serviço
start:
	sudo ./$(BINARY_NAME) -start

# Parar serviço
stop:
	sudo ./$(BINARY_NAME) -stop

# Reiniciar serviço
restart:
	sudo ./$(BINARY_NAME) -restart

# Limpeza
clean:
	go clean
	rm -f ${BINARY_NAME}
	rm -f ${BINARY_UNIX}
	rm -f ${BINARY_WINDOWS}
	rm -f ${BINARY_DARWIN}
	rm -f ${BINARY_DARWIN}_arm64

# Formatação do código
fmt:
	go fmt ./...

# Linter
lint:
	golangci-lint run

# Verificar vulnerabilidades
security:
	gosec ./...

# Gerar documentação
docs:
	godoc -http=:6060

# Criar release
release: clean deps test build-all
	mkdir -p release
	cp ${BINARY_UNIX} release/
	cp ${BINARY_WINDOWS} release/
	cp ${BINARY_DARWIN} release/
	cp ${BINARY_DARWIN}_arm64 release/
	cp config.json release/
	cp README.md release/ 2>/dev/null || true
	cd release && tar -czf machine-monitor-agent-${VERSION}.tar.gz *

# Criar pacote para distribuição
package: release
	@echo "Pacote criado: release/machine-monitor-agent-${VERSION}.tar.gz"

# Ajuda
help:
	@echo "Comandos disponíveis:"
	@echo "  build          - Compila o binário para a plataforma atual"
	@echo "  build-linux    - Compila para Linux"
	@echo "  build-windows  - Compila para Windows"
	@echo "  build-darwin   - Compila para macOS (Intel)"
	@echo "  build-darwin-arm64 - Compila para macOS (Apple Silicon)"
	@echo "  build-all      - Compila para todas as plataformas"
	@echo "  deps           - Baixa dependências"
	@echo "  test           - Executa testes"
	@echo "  run            - Executa em modo console"
	@echo "  run-debug      - Executa em modo console com debug"
	@echo "  install        - Instala como serviço"
	@echo "  uninstall      - Remove serviço"
	@echo "  start          - Inicia serviço"
	@echo "  stop           - Para serviço"
	@echo "  restart        - Reinicia serviço"
	@echo "  clean          - Limpa arquivos gerados"
	@echo "  fmt            - Formata código"
	@echo "  lint           - Executa linter"
	@echo "  security       - Verifica vulnerabilidades"
	@echo "  docs           - Gera documentação"
	@echo "  release        - Cria release completo"
	@echo "  package        - Cria pacote para distribuição"
	@echo "  help           - Mostra esta ajuda" 