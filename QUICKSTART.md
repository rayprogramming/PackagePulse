# PackagePulse Quick Start Guide

This guide will help you quickly set up and run PackagePulse MCP server for vulnerability scanning and package health analysis.

## Prerequisites

- **Go 1.24.3 or higher** - Required for building the project
- Terminal/Command line access

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/rayprogramming/PackagePulse.git
cd PackagePulse
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Build the Project

```bash
go build -o PackagePulse
```

This will create a `PackagePulse` binary in your current directory.

## Running PackagePulse

### Basic Usage

PackagePulse runs as an MCP (Model Context Protocol) server using stdio transport:

```bash
./PackagePulse
```

The server will start and communicate via stdin/stdout, ready to accept MCP protocol messages.

### Using with MCP Inspector

To test the server interactively, use the official MCP Inspector:

```bash
npx @modelcontextprotocol/inspector ./PackagePulse
```

### Using with MCP Clients

Configure your MCP client (e.g., Claude Desktop, VSCode extensions) to connect to PackagePulse:

**Example Claude Desktop configuration** (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "packagepulse": {
      "command": "/path/to/PackagePulse/PackagePulse"
    }
  }
}
```

## Available Features

PackagePulse provides the following MCP tools:

### 1. Vulnerability Scanning (`vulns`)
Check packages for known security vulnerabilities using OSV database.

**Example:**
```json
{
  "ecosystem": "npm",
  "package": "lodash",
  "version": "4.17.19"
}
```

### 2. Package Health Analysis (`health`)
Analyze package maintenance status, update frequency, and overall health.

**Example:**
```json
{
  "ecosystem": "npm",
  "package": "express"
}
```

### 3. License Information (`license`)
Retrieve SPDX license details, compatibility, and categorization.

**Example:**
```json
{
  "license_id": "MIT"
}
```

### 4. Upgrade Planning (`upgrade-plan`)
Generate smart upgrade recommendations based on vulnerabilities and package health.

**Example:**
```json
{
  "ecosystem": "npm",
  "package": "lodash",
  "current_version": "4.17.19"
}
```

## Development

### Running Tests

```bash
go test -v ./...
```

**Note:** Some tests require network access to external APIs (OSV, deps.dev). They may fail in restricted environments but this is expected.

### Linting

```bash
go vet ./...
```

For comprehensive linting with golangci-lint:

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run ./...
```

### Local Development

For active development, you can run the server directly:

```bash
go run main.go
```

## Troubleshooting

### Binary Not Found
If you get "command not found" after building:
- Make sure you're in the correct directory
- Use `./PackagePulse` (with `./` prefix) on Unix/Linux/macOS
- Use `PackagePulse.exe` on Windows

### Build Failures
If `go build` fails:
- Verify your Go version: `go version` (must be 1.24.3+)
- Ensure dependencies are downloaded: `go mod download`
- Try cleaning the build cache: `go clean -cache`

### Connection Issues
If the MCP client can't connect:
- Verify the binary path in your client configuration
- Check that the binary has execute permissions: `chmod +x PackagePulse`
- Review client logs for specific error messages

## Project Structure

```
PackagePulse/
├── main.go                      # Server entry point
├── go.mod                       # Go module definition
├── internal/
│   ├── tools/                   # MCP tool implementations
│   ├── resources/               # MCP resource handlers
│   └── providers/               # External API clients
│       ├── osv/                 # OSV vulnerability database
│       ├── depsdev/             # Google deps.dev API
│       └── spdx/                # SPDX license information
└── QUICKSTART.md               # This file
```

## Next Steps

- Explore the [hypermcp framework documentation](https://github.com/rayprogramming/hypermcp)
- Review the [MCP protocol specification](https://modelcontextprotocol.io)
- Check out example integrations in the `examples/` directory (coming soon)

## Support

For issues, questions, or contributions:
- Open an issue on [GitHub](https://github.com/rayprogramming/PackagePulse/issues)
- Review existing documentation in the repository

---

**Version:** 1.0.0  
**Last Updated:** 2025-10-15
