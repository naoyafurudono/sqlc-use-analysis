# 開発環境・ツールチェーン定義

## 1. 開発環境要件

### 1.1. 基本環境

```yaml
言語: Go 1.21+
対象OS: 
  - macOS (開発)
  - Linux (本番・CI)
  - Windows (互換性確認)
  
最小システム要件:
  - CPU: 2コア以上
  - メモリ: 4GB以上
  - ディスク: 2GB以上の空き容量
```

### 1.2. 必要なソフトウェア

```bash
# 必須
Go 1.21+
Git
Make

# 推奨
Docker
Docker Compose
```

## 2. 開発ツール

### 2.1. IDE・エディタ

#### VS Code（推奨）
```json
{
  "extensions": [
    "golang.Go",
    "ms-vscode.vscode-json",
    "redhat.vscode-yaml",
    "ms-vscode.test-adapter-converter",
    "hbenl.vscode-test-explorer"
  ],
  "settings": {
    "go.lintTool": "golangci-lint",
    "go.formatTool": "goimports",
    "go.testFlags": ["-v", "-race"],
    "go.coverOnSave": true,
    "go.coverageDecorator": "gutter"
  }
}
```

#### GoLand（代替）
```yaml
設定:
  - Go Modules: 有効
  - Live Templates: Go標準
  - Code Inspection: 最大
  - Test Runner: 統合
```

### 2.2. コマンドラインツール

```bash
# Go開発ツール
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/golang/mock/mockgen@latest
go install github.com/goreleaser/goreleaser@latest

# テスト・カバレッジ
go install github.com/boumenot/gocover-cobertura@latest
go install github.com/axw/gocov/gocov@latest
go install github.com/AlekSi/gocov-xml@latest

# プロファイリング
go install github.com/google/pprof@latest
```

## 3. プロジェクト構造

### 3.1. ディレクトリ構成

```
sqlc-use-analysis/
├── cmd/
│   └── analyzer/
│       └── main.go                 # エントリーポイント
├── internal/
│   ├── analyzer/
│   │   ├── sql/                   # SQL解析
│   │   │   ├── analyzer.go
│   │   │   ├── extractor.go
│   │   │   ├── parser.go
│   │   │   └── types.go
│   │   ├── go/                    # Go解析
│   │   │   ├── analyzer.go
│   │   │   ├── collector.go
│   │   │   ├── loader.go
│   │   │   └── types.go
│   │   └── mapper/                # 依存関係マッピング
│   │       ├── dependency.go
│   │       ├── graph.go
│   │       └── resolver.go
│   ├── config/
│   │   ├── config.go
│   │   ├── loader.go
│   │   └── validator.go
│   ├── errors/
│   │   ├── collector.go
│   │   ├── reporter.go
│   │   └── types.go
│   ├── io/
│   │   ├── formatter.go
│   │   ├── input.go
│   │   └── output.go
│   └── orchestrator/
│       ├── orchestrator.go
│       └── pipeline.go
├── pkg/
│   └── types/                     # 公開型
│       ├── analysis.go
│       ├── config.go
│       └── dependency.go
├── test/
│   ├── fixtures/                  # テストデータ
│   │   ├── simple_project/
│   │   ├── complex_project/
│   │   └── edge_cases/
│   ├── integration/               # 統合テスト
│   │   ├── analyzer_test.go
│   │   └── e2e_test.go
│   └── unit/                      # ユニットテスト
│       ├── sql_test.go
│       ├── go_test.go
│       └── mapper_test.go
├── examples/                      # 使用例
│   ├── basic/
│   ├── advanced/
│   └── custom_config/
├── docs/                          # ドキュメント
├── scripts/                       # スクリプト
│   ├── build.sh
│   ├── test.sh
│   └── release.sh
├── .github/
│   └── workflows/
│       ├── ci.yml
│       ├── release.yml
│       └── security.yml
├── deployments/
│   ├── docker/
│   │   ├── Dockerfile
│   │   └── docker-compose.yml
│   └── k8s/
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── LICENSE
└── .gitignore
```

### 3.2. 設定ファイル

#### .golangci.yml
```yaml
run:
  timeout: 5m
  modules-download-mode: readonly

linters:
  enable-all: true
  disable:
    - exhaustivestruct
    - exhaustruct
    - gochecknoglobals
    - gochecknoinits
    - gofumpt
    - goimports
    - golint
    - gomnd
    - interfacer
    - maligned
    - scopelint
    - wsl

linters-settings:
  cyclop:
    max-complexity: 15
  dupl:
    threshold: 100
  funlen:
    lines: 80
    statements: 40
  gocognit:
    min-complexity: 15
  goconst:
    min-len: 3
    min-occurrences: 3
  gocyclo:
    min-complexity: 15
  godot:
    capital: true
  godox:
    keywords:
      - TODO
      - FIXME
      - BUG
  goerr113:
    check-type-assertions: true
    check-blank: true
  goheader:
    values:
      const:
        COMPANY: "sqlc-use-analysis"
      regexp:
        YEAR: "20[0-9][0-9]"
  goimports:
    local-prefixes: github.com/naoyafurudono/sqlc-use-analysis
  gomoddirectives:
    replace-local: true
  gomodguard:
    blocked:
      modules:
        - github.com/golang/protobuf:
            recommendations:
              - google.golang.org/protobuf
  goprintffuncname:
    funcs:
      - (github.com/golangci/golangci-lint/pkg/logutils.Log).Infof
      - (github.com/golangci/golangci-lint/pkg/logutils.Log).Warnf
      - (github.com/golangci/golangci-lint/pkg/logutils.Log).Errorf
      - (github.com/golangci/golangci-lint/pkg/logutils.Log).Fatalf
  gosec:
    severity: medium
    confidence: medium
  lll:
    line-length: 120
  misspell:
    locale: US
  nestif:
    min-complexity: 6
  nolintlint:
    require-explanation: true
    require-specific: true
  prealloc:
    simple: true
    range-loops: true
    for-loops: true
  testpackage:
    skip-regexp: (export|internal)_test\.go
  unparam:
    check-exported: true
  unused:
    check-exported: false
  whitespace:
    multi-if: true
    multi-func: true

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - dupl
        - funlen
        - gocognit
        - gocyclo
        - gosec
        - lll
    - path: test/fixtures/
      linters:
        - lll
        - dupl
```

#### Makefile
```makefile
.PHONY: build test lint clean install deps help

# Variables
BINARY_NAME=sqlc-analyzer
VERSION?=dev
LDFLAGS=-ldflags "-X main.version=${VERSION}"

# Default target
all: deps lint test build

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run linter
lint:
	golangci-lint run

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html

# Build the binary
build:
	go build ${LDFLAGS} -o bin/${BINARY_NAME} cmd/analyzer/main.go

# Build for all platforms
build-all:
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-linux-amd64 cmd/analyzer/main.go
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-amd64 cmd/analyzer/main.go
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-windows-amd64.exe cmd/analyzer/main.go

# Install the binary
install:
	go install ${LDFLAGS} cmd/analyzer/main.go

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Run security scan
security:
	govulncheck ./...

# Generate mocks
generate:
	go generate ./...

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Run integration tests
integration-test:
	go test -tags=integration ./test/integration/...

# Docker build
docker-build:
	docker build -t ${BINARY_NAME}:${VERSION} .

# Docker run
docker-run:
	docker run --rm ${BINARY_NAME}:${VERSION}

# Help
help:
	@echo "Available targets:"
	@echo "  deps             - Install dependencies"
	@echo "  lint             - Run linter"
	@echo "  test             - Run tests"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  build            - Build binary"
	@echo "  build-all        - Build for all platforms"
	@echo "  install          - Install binary"
	@echo "  clean            - Clean build artifacts"
	@echo "  security         - Run security scan"
	@echo "  generate         - Generate mocks"
	@echo "  bench            - Run benchmarks"
	@echo "  integration-test - Run integration tests"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-run       - Run Docker container"
	@echo "  help             - Show this help"
```

## 4. CI/CD設定

### 4.1. GitHub Actions

#### .github/workflows/ci.yml
```yaml
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: [ 1.21.x, 1.22.x ]
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ matrix.go-version }}
    
    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: |
          ~/.cache/go-build
          ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ matrix.go-version }}-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-${{ matrix.go-version }}-
    
    - name: Install dependencies
      run: make deps
    
    - name: Run linter
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest
        args: --timeout=5m
    
    - name: Run tests
      run: make test
    
    - name: Run security scan
      run: make security
    
    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
        flags: unittests
        name: codecov-umbrella
    
    - name: Build
      run: make build
    
    - name: Run integration tests
      run: make integration-test

  build-matrix:
    runs-on: ubuntu-latest
    needs: test
    if: github.ref == 'refs/heads/main'
    
    strategy:
      matrix:
        goos: [linux, darwin, windows]
        goarch: [amd64]
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: 1.21.x
    
    - name: Build
      env:
        GOOS: ${{ matrix.goos }}
        GOARCH: ${{ matrix.goarch }}
      run: |
        make build
        if [ "$GOOS" = "windows" ]; then
          mv bin/sqlc-analyzer bin/sqlc-analyzer-${GOOS}-${GOARCH}.exe
        else
          mv bin/sqlc-analyzer bin/sqlc-analyzer-${GOOS}-${GOARCH}
        fi
    
    - name: Upload artifacts
      uses: actions/upload-artifact@v3
      with:
        name: binaries
        path: bin/
```

#### .github/workflows/release.yml
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
      with:
        fetch-depth: 0
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: 1.21.x
    
    - name: Run GoReleaser
      uses: goreleaser/goreleaser-action@v4
      with:
        version: latest
        args: release --clean
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### 4.2. GoReleaser設定

#### .goreleaser.yml
```yaml
before:
  hooks:
    - go mod tidy
    - go generate ./...

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - windows
      - darwin
    goarch:
      - amd64
      - arm64
    main: ./cmd/analyzer/main.go
    binary: sqlc-analyzer
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: 'checksums.txt'

snapshot:
  name_template: "{{ incpatch .Version }}-next"

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'
      - 'merge conflict'
      - Merge pull request
      - Merge remote-tracking branch
      - Merge branch
      - go mod tidy
  groups:
    - title: Features
      regexp: "^.*feat[(\\w)]*:+.*$"
      order: 0
    - title: 'Bug fixes'
      regexp: "^.*fix[(\\w)]*:+.*$"
      order: 1
    - title: Others
      order: 999
```

## 5. Docker設定

### 5.1. Dockerfile

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o sqlc-analyzer cmd/analyzer/main.go

# Runtime stage
FROM scratch

# Copy ca certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/sqlc-analyzer /sqlc-analyzer

# Set entrypoint
ENTRYPOINT ["/sqlc-analyzer"]
```

### 5.2. docker-compose.yml

```yaml
version: '3.8'

services:
  sqlc-analyzer:
    build: .
    volumes:
      - ./test/fixtures:/workspace
      - ./output:/output
    environment:
      - SQLC_ANALYZER_ROOT_PATH=/workspace
      - SQLC_ANALYZER_OUTPUT_PATH=/output/analysis.json
    working_dir: /workspace
```

## 6. 開発ワークフロー

### 6.1. 開発プロセス

```bash
# 1. リポジトリのクローン
git clone https://github.com/naoyafurudono/sqlc-use-analysis.git
cd sqlc-use-analysis

# 2. 依存関係のインストール
make deps

# 3. 開発サイクル
make lint    # 静的解析
make test    # テスト実行
make build   # ビルド

# 4. 統合テスト
make integration-test

# 5. セキュリティチェック
make security
```

### 6.2. Git ワークフロー

```bash
# Feature branch workflow
git checkout -b feature/sql-parser
# 開発...
git add .
git commit -m "feat: add SQL parser functionality"
git push origin feature/sql-parser
# Pull Request作成
```

### 6.3. コミットメッセージ規約

```
<type>(<scope>): <description>

feat: 新機能
fix: バグ修正
docs: ドキュメント
style: フォーマット
refactor: リファクタリング
test: テスト
chore: その他

例:
feat(sql): add support for CTE parsing
fix(mapper): resolve circular dependency detection
docs(readme): add installation instructions
```

## 7. デバッグとプロファイリング

### 7.1. デバッグ設定

```bash
# デバッグビルド
go build -gcflags="-N -l" -o bin/sqlc-analyzer-debug cmd/analyzer/main.go

# delveでのデバッグ
dlv exec bin/sqlc-analyzer-debug

# VS Codeでのデバッグ設定
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/analyzer/main.go",
      "args": [],
      "env": {
        "SQLC_ANALYZER_DEBUG": "true"
      }
    }
  ]
}
```

### 7.2. プロファイリング

```bash
# CPUプロファイル
go build -o bin/sqlc-analyzer cmd/analyzer/main.go
./bin/sqlc-analyzer -cpuprofile=cpu.prof

# メモリプロファイル
./bin/sqlc-analyzer -memprofile=mem.prof

# プロファイル解析
go tool pprof cpu.prof
go tool pprof mem.prof
```

## 8. 環境変数

### 8.1. 開発環境

```bash
# 必須
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# 推奨
export GOPROXY=https://proxy.golang.org
export GOSUMDB=sum.golang.org
export GOPRIVATE=github.com/naoyafurudono/*

# デバッグ
export SQLC_ANALYZER_DEBUG=true
export SQLC_ANALYZER_LOG_LEVEL=debug
```

### 8.2. CI/CD環境

```bash
# GitHub Actions secrets
GITHUB_TOKEN         # リリース用
CODECOV_TOKEN        # カバレッジ用
DOCKER_USERNAME      # Docker Hub用
DOCKER_PASSWORD      # Docker Hub用
```

## 9. トラブルシューティング

### 9.1. よくある問題

```bash
# Go modules の問題
go mod tidy
go mod verify

# キャッシュの問題
go clean -modcache
go clean -cache

# 依存関係の問題
go mod download
```

### 9.2. パフォーマンス問題

```bash
# メモリ使用量の確認
go tool pprof -alloc_space mem.prof

# CPU使用量の確認
go tool pprof -top cpu.prof

# ベンチマーク
go test -bench=. -benchmem ./...
```

## 10. 設定完了チェックリスト

- [ ] Go 1.21+ インストール済み
- [ ] 必要なツールのインストール完了
- [ ] VS Code/GoLand の設定完了
- [ ] プロジェクト構造の作成完了
- [ ] Makefile の動作確認
- [ ] GitHub Actions の設定完了
- [ ] Docker 環境の構築完了
- [ ] 開発ワークフローの理解完了

この開発環境設定により、効率的で品質の高い開発が可能になります。