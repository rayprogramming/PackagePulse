# PackagePulse - AI Coding Agent Instructions

## Project Overview
PackagePulse is an MCP (Model Context Protocol) server for vulnerability scanning and package health analysis. It's built using the `hypermcp` framework (custom wrapper around the official `modelcontextprotocol/go-sdk`).

## Architecture

### Core Components
- **MCP Server**: Uses `hypermcp.Server` as the foundation, which wraps the official Go MCP SDK
- **Transport**: Runs on stdio transport (`hypermcp.TransportStdio`) for communication with MCP clients
- **Caching**: Built-in Ristretto cache (100MB max, 10k counters, 64 buffer items)
- **Logging**: Structured logging via `zap.Logger` in production mode

### Key Files
- `main.go`: Server initialization, configuration, and feature registration entry point
- `go.mod`: Dependency management (Go 1.24.3+, hypermcp v1.0.0)

## Development Workflow

### Setup & Run
```bash
go mod tidy          # Install/sync dependencies
go run main.go       # Run the MCP server (stdio mode)
```

### Testing with MCP Inspector
The server communicates via stdio, so test with the official MCP Inspector or an MCP-compatible client.

## Critical Patterns

### Feature Registration Pattern
All tools, resources, and prompts MUST be registered in `registerFeatures()` function:

```go
func registerFeatures(srv *hypermcp.Server) {
    // Register MCP tools here using srv.RegisterTool()
    // Register MCP resources using srv.RegisterResource()
    // Register MCP prompts using srv.RegisterPrompt()
}
```

### hypermcp Framework Conventions
- **Server Initialization**: Always create server with `hypermcp.New(cfg, logger)`
- **Configuration**: Use `hypermcp.Config` struct with Name, Version, CacheEnabled, and CacheConfig
- **Transport**: Launch with `hypermcp.RunWithTransport(ctx, srv, transport, logger)`
- **Registration Stats**: Call `srv.LogRegistrationStats()` before running to verify registrations

### MCP Protocol Specifics
- **Tools**: Interactive functions that clients can invoke (e.g., vulnerability scans)
- **Resources**: Read-only data sources (e.g., package metadata, health reports)
- **Prompts**: Template-based interactions for common workflows

### Error Handling
- Use `logger.Fatal()` for startup failures (server creation, transport errors)
- Return proper error responses in tool/resource handlers (the hypermcp framework handles error serialization)

## Project-Specific Context

### Purpose & Use Cases
- **Code Reviews**: Scan dependencies for vulnerabilities during PR reviews
- **CI/CD Integration**: Automated security checks in build pipelines
- **Security Posture**: Real-time package health monitoring in AI assistant conversations

### Current State
The server skeleton is in place but `registerFeatures()` is a placeholder. When implementing:
1. Tools should check vulnerabilities (OSV, Snyk, etc.) and analyze package health
2. Resources might expose vulnerability databases or package metadata
3. Prompts could guide users through security assessment workflows

### Next Steps for Development
When adding functionality, follow this pattern:
1. Define tool/resource/prompt interfaces in separate files (e.g., `tools.go`, `resources.go`)
2. Implement handlers with proper context handling and error returns
3. Register in `registerFeatures()` before `LogRegistrationStats()` call
4. Leverage the built-in cache for expensive API calls (vulnerability lookups, package metadata)

## Common Gotchas
- **Go Version**: Requires Go 1.24.3+ (check `go.mod`)
- **stdio Transport**: Server blocks on stdin/stdout; can't use regular print statements for debugging (use logger)
- **Cache Configuration**: Ristretto cache is pre-configured; MaxCost is in bytes (currently 100MB)
- **Context Propagation**: Always pass context through the call chain for proper cancellation

## Integration Points
- External vulnerability databases (OSV, NVD, Snyk APIs)
- Package registries (npm, PyPI, crates.io, Maven Central)
- CI/CD systems (GitHub Actions, GitLab CI, Jenkins)
- MCP clients (Claude Desktop, VSCode extensions, custom integrations)
