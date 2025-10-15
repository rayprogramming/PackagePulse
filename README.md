# PackagePulse

PackagePulse is an MCP server designed to check Open Vulnerability & Package Health. It is instantly useful in code reviews, CI assistants, and security posture chats.

## Features
- Open Vulnerability checks
- Package Health analysis
- Integration with CI/CD pipelines

## Getting Started

### Prerequisites
- Go 1.24+

### Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/rayprogramming/PackagePulse.git
   ```
2. Navigate to the project directory:
   ```bash
   cd PackagePulse
   ```
3. Install dependencies:
   ```bash
   go mod tidy
   ```

### Running the Server
# PackagePulse

PackagePulse is a production-ready Model Context Protocol (MCP) server for open-source package security and health analysis. It provides real-time vulnerability scanning, dependency maintenance metrics, and license information through MCP tools and resources.

## Features

### Tools
- **deps.vulns** - Query OSV.dev for known vulnerabilities ✅ IMPLEMENTED
- **deps.health** - Get package health metrics from deps.dev ✅ IMPLEMENTED
- **license.info** - Look up SPDX license information ✅ IMPLEMENTED
- **deps.upgrade_plan** - Generate safe upgrade recommendations ✅ IMPLEMENTED

### Resources
- **res://osv/vulns** - OSV vulnerability database access
- **res://deps/graph** - Package dependency graph from deps.dev
- **res://license/spdx** - SPDX license database queries

## Installation

### Prerequisites
- Go 1.24.3+

### Quick Start

1. Clone the repository:
   ```bash
   git clone https://github.com/rayprogramming/PackagePulse.git
   cd PackagePulse
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Run the server:
   ```bash
   go run main.go
   ```

4. Or build and run:
   ```bash
   go build -o packagepulse
   ./packagepulse
   ```

## Usage Examples

### Tool: deps.vulns
Query for vulnerabilities in a package:

```json
{
  "ecosystem": "npm",
  "package": "lodash",
  "version": "4.17.19"
}
```

Response includes vulnerability count, detailed CVE information, and severity summary.

### Tool: deps.health
Get package health metrics:

```json
{
  "ecosystem": "npm",
  "package": "express"
}
```

Returns:
- Latest version
- Version count
- Days since last update
- Maintenance score (0-100)
- Maintenance level (excellent/good/fair/poor/critical)

### Tool: license.info
Look up license details:

```json
{
  "license_id": "MIT"
}
```

Returns SPDX license metadata including OSI approval status.

### Tool: deps.upgrade_plan
Generate upgrade recommendations:

```json
{
  "ecosystem": "npm",
  "package": "lodash",
  "current_version": "4.17.19"
}
```

Returns safe upgrade path with vulnerability analysis and maintenance assessment.

### Resource: res://osv/vulns
```
res://osv/vulns?ecosystem=npm&package=lodash&version=4.17.19
```

### Resource: res://deps/graph
```
res://deps/graph?ecosystem=npm&package=express
```

### Resource: res://license/spdx
```
res://license/spdx?id=MIT
res://license/spdx?query=apache
res://license/spdx?osi_only=true
```

## Architecture

PackagePulse follows a clean, modular architecture:

```
PackagePulse/
├── main.go                          # Server initialization
├── internal/
│   ├── providers/                   # External API clients
│   │   ├── osv/                     # OSV.dev client
│   │   ├── depsdev/                 # deps.dev client
│   │   └── spdx/                    # SPDX license provider
│   ├── tools/                       # MCP tool implementations
│   └── resources/                   # MCP resource implementations
```

### Key Design Decisions

- **Caching**: Ristretto cache with 5-minute TTL for API responses
- **Context Handling**: Full context propagation for cancellation
- **Error Handling**: Typed errors with context information
- **Logging**: Structured logging via zap
- **Testing**: Comprehensive unit and integration tests

## Performance

- Cache hit latency: <100 µs (target)
- OSV API queries: ~300-500ms (uncached)
- deps.dev queries: ~400-700ms (uncached)
- SPDX lookups: <1ms (embedded data)

Run benchmarks:
```bash
go test -bench=. ./internal/tools/
```

## Testing

Run all tests:
```bash
go test ./...
```

Run with coverage:
```bash
go test -cover ./...
```

Run specific provider tests:
```bash
go test ./internal/providers/osv/
go test ./internal/providers/depsdev/
go test ./internal/providers/spdx/
```

## Development

### Adding New Tools

1. Implement handler in `internal/tools/tools.go`
2. Define input/output structs
3. Register in `Register()` method
4. Add tests in `tools_test.go`

### Adding New Resources

1. Implement handler in `internal/resources/resources.go`
2. Parse URI parameters
3. Register in `Register()` method
4. Document URI format

## API Data Sources

- **OSV.dev**: https://api.osv.dev/v1/query (free, no auth)
- **deps.dev**: https://deps.dev/_/s/{ecosystem}/p/{name} (free, no auth)
- **SPDX**: Embedded JSON dataset (v3.24.0)

## Configuration

Environment variables:
- None required (uses public APIs)

Cache configuration (in main.go):
- MaxCost: 100MB
- NumCounters: 10,000
- BufferItems: 64

## Use Cases

- **Code Reviews**: Scan dependencies during PR reviews
- **CI/CD**: Automated security checks in pipelines
- **Security Audits**: Real-time vulnerability assessments
- **License Compliance**: SPDX license verification
- **Dependency Management**: Health monitoring and upgrade planning

## Supported Ecosystems

- npm (Node.js)
- PyPI (Python)
- Go
- Maven (Java)
- Cargo (Rust)
- NuGet (.NET)

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Credits

Built with:
- [hypermcp](https://github.com/rayprogramming/hypermcp) - MCP framework
- [OSV.dev](https://osv.dev) - Vulnerability database
- [deps.dev](https://deps.dev) - Package metadata
- [SPDX](https://spdx.org) - License data

### License
MIT License