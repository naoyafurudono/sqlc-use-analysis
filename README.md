# sqlc-use-analysis

A sqlc plugin that analyzes and visualizes database dependencies in Go projects, with primary support for MySQL databases.

## 🎯 Features

### Core Functionality
- **MySQL-First Support**: Optimized for MySQL syntax including backtick-quoted identifiers
- **Comprehensive SQL Analysis**: Extracts table names from SELECT, INSERT, UPDATE, DELETE operations
- **Go Static Analysis**: Uses AST parsing to identify SQLC method calls
- **Dependency Mapping**: Maps relationships between Go functions and database tables
- **JSON Output**: Structured JSON output for easy integration with other tools

### Key Benefits
- **Impact Analysis**: Quickly identify which functions are affected by table schema changes
- **Architecture Visualization**: Understand data flow and dependencies in your codebase
- **SQLC Integration**: Works seamlessly as an SQLC plugin

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

### SQLC Plugin Configuration

Add to your `sqlc.yaml`:

```yaml
version: "2"
sql:
  - engine: "mysql"
    queries: "query.sql"
    schema: "schema.sql"
    gen:
      go:
        package: "db"
        out: "db"
plugins:
  - name: "dependency-analyzer"
    process:
      cmd: "sqlc-analyzer"
    options:
      root_path: "."
      output_path: "db_dependencies.json"
      exclude:
        - "**/*_test.go"
        - "vendor/"
```

### Generate Dependencies
```bash
sqlc generate
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

## 🎯 Database Support

### Primary Support
- **MySQL**: Full support for MySQL-specific syntax including:
  - Backtick-quoted identifiers (\`table_name\`)
  - MySQL JOIN syntax
  - Common MySQL patterns

### Secondary Support
- **PostgreSQL**: Basic support for standard SQL syntax
  - Double-quoted identifiers ("table_name")
  - Standard SQL operations

## 📄 Output Format

The plugin generates JSON output optimized for programmatic consumption:

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