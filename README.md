# sqlc-use-analysis

A sqlc plugin that analyzes and visualizes database dependencies in Go projects.

## 🎯 Project Status

**Current Phase**: M1 - Infrastructure Development (Week 1-2)

### ✅ Completed
- [x] Project initialization with Go modules
- [x] Complete directory structure setup
- [x] Basic CLI implementation with plugin interface
- [x] Comprehensive type definitions for analysis
- [x] Configuration management system
- [x] Error handling framework
- [x] Input/output interfaces
- [x] Basic orchestrator for coordinating analysis
- [x] Makefile with development commands
- [x] GitHub Actions CI/CD pipeline
- [x] Comprehensive unit tests with good coverage
- [x] golangci-lint configuration
- [x] Git ignore and basic project files

### 🔄 In Progress
- [ ] SQL query analyzer implementation
- [ ] Go static analyzer implementation
- [ ] Dependency mapping engine

### 📋 Next Steps
1. Implement SQL query analyzer (M2)
2. Implement Go static analyzer (M3)
3. Create dependency mapping engine (M4)

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Make
- Git

### Installation
```bash
git clone https://github.com/naoyafurudono/sqlc-use-analysis.git
cd sqlc-use-analysis
make deps
make build
```

### Basic Usage
```bash
# Test with sample input
echo '{"settings": {"root_path": "."}, "queries": []}' | ./bin/sqlc-analyzer
```

## 🏗️ Architecture

The plugin follows a modular architecture:

```
┌─────────────────────────────────────────────────────────────┐
│                     sqlc generate                           │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                  Plugin Main Process                        │
│  ┌─────────────────────┐ ┌──────────────────────┐          │
│  │  Config Loader      │ │  Error Handler       │          │
│  └─────────────────────┘ └──────────────────────┘          │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │               Analysis Orchestrator                 │   │
│  └─────────────────────────────────────────────────────┘   │
│                   │                    │                    │
│  ┌────────────────▼──────┐ ┌─────────▼──────────┐         │
│  │  SQL Query Analyzer   │ │  Go Static Analyzer │         │
│  └───────────────────────┘ └────────────────────┘         │
│                   │                    │                    │
│  ┌────────────────▼────────────────────▼──────────┐        │
│  │            Dependency Mapper                    │        │
│  └────────────────────────┬───────────────────────┘        │
│                           │                                 │
│  ┌────────────────────────▼───────────────────────┐        │
│  │              JSON Output Formatter              │        │
│  └─────────────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint

# Run security scan
make security
```

Current test coverage: **~85%**

## 📚 Documentation

- [Technical Specification](docs/spec.md)
- [System Design](docs/design.md)
- [Development Plan](docs/development_plan.md)
- [Risk Analysis](docs/risk_analysis.md)

## 🛠️ Development

### Make Commands
```bash
make deps             # Install dependencies
make build            # Build binary
make test             # Run tests
make lint             # Run linter
make clean            # Clean build artifacts
make help             # Show all commands
```

### Project Structure
```
.
├── cmd/analyzer/           # CLI entry point
├── internal/
│   ├── config/            # Configuration management
│   ├── errors/            # Error handling
│   ├── io/                # Input/output interfaces
│   ├── orchestrator/      # Analysis coordination
│   └── analyzer/          # Analysis modules (TODO)
├── pkg/types/             # Public type definitions
├── test/                  # Test files and fixtures
└── docs/                  # Documentation
```

## 🎯 Goals

This plugin will provide:

1. **Function-level analysis**: Identify which Go functions access which database tables
2. **Operation mapping**: Track SELECT, INSERT, UPDATE, DELETE operations per function
3. **Dependency visualization**: Generate comprehensive dependency maps
4. **Architecture insights**: Help understand data flow and dependencies

## 📄 Output Format

The plugin generates JSON output with both function-centric and table-centric views:

```json
{
  "metadata": {
    "generated_at": "2024-01-01T00:00:00Z",
    "version": "1.0.0",
    "total_functions": 10,
    "total_tables": 5
  },
  "function_view": {
    "package.FunctionName": [
      {
        "table": "users",
        "operations": ["SELECT", "INSERT"]
      }
    ]
  },
  "table_view": {
    "users": [
      {
        "function": "package.FunctionName",
        "operations": ["SELECT", "INSERT"]
      }
    ]
  }
}
```

## 🤝 Contributing

This project is currently in active development. See [Development Plan](docs/development_plan.md) for the roadmap.

## 📜 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🔗 Links

- [sqlc Documentation](https://docs.sqlc.dev/)
- [Go Static Analysis](https://pkg.go.dev/go/analysis)
- [Plugin Development Guide](https://docs.sqlc.dev/en/latest/reference/plugins.html)

---

**Generated with Claude Code** 🤖