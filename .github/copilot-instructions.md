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
- `internal/tools/tools.go`: MCP tool implementations (deps.vulns, deps.health, license.info, deps.upgrade_plan)
- `internal/resources/resources.go`: MCP resource implementations (currently minimal)
- `internal/providers/osv/`: OSV.dev API client for vulnerability data
- `internal/providers/depsdev/`: deps.dev API client for package health metrics
- `internal/providers/spdx/`: Embedded SPDX license database

## Development Workflow

### Setup & Run
```bash
go mod tidy          # Install/sync dependencies
go run main.go       # Run the MCP server (stdio mode)
go build -o packagepulse  # Build binary
./packagepulse       # Run binary
```

### Testing
```bash
go test ./...                    # Run all tests
go test -v ./internal/tools      # Run specific package tests with verbose output
go test -run TestName ./...      # Run specific test
go test -cover ./...             # Run tests with coverage
```

**Note**: Some tests require internet access to external APIs (OSV.dev, deps.dev). Tests are structured to be informative even when network calls fail.

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
The server is fully implemented with the following features:
1. **Tools Implemented** (in `internal/tools/tools.go`):
   - `deps.vulns` - Query OSV.dev for vulnerability information
   - `deps.health` - Get package health metrics from deps.dev
   - `license.info` - Query SPDX license database
   - `deps.upgrade_plan` - Generate upgrade recommendations with vulnerability analysis
2. **Resources** - Basic structure in place (`internal/resources/resources.go`)
3. **Provider Clients**:
   - `internal/providers/osv` - OSV.dev API client for vulnerabilities
   - `internal/providers/depsdev` - deps.dev API client for package health
   - `internal/providers/spdx` - Embedded SPDX license database

### Adding New Features
When adding new tools, resources, or prompts:
1. Define tool/resource/prompt interfaces in the appropriate registry file (`internal/tools/tools.go` or `internal/resources/resources.go`)
2. Implement handlers with proper context handling and error returns
3. Register in the appropriate `Register()` method called by `registerFeatures()` in `main.go`
4. Leverage the built-in cache for expensive API calls (vulnerability lookups, package metadata)
5. Add comprehensive tests following the pattern in `*_test.go` files

## Common Gotchas
- **Go Version**: Requires Go 1.24.3+ (check `go.mod`)
- **stdio Transport**: Server blocks on stdin/stdout; can't use regular print statements for debugging (use logger)
- **Cache Configuration**: Ristretto cache is pre-configured; MaxCost is in bytes (currently 100MB)
- **Context Propagation**: Always pass context through the call chain for proper cancellation
- **Test Network Access**: Some tests make real API calls to OSV.dev and deps.dev; they may fail in restricted network environments
- **JSON Marshaling**: All tool inputs/outputs must be JSON-serializable structs with proper tags

## Provider Architecture

### Provider Pattern
Each external data source has a dedicated client in `internal/providers/`:

1. **OSV Client** (`internal/providers/osv/`):
   - Queries OSV.dev API for vulnerability information
   - Methods: `Query()`, `BatchQuery()`
   - Returns structured vulnerability data with severity, CVEs, and remediation info

2. **DepsDevClient** (`internal/providers/depsdev/`):
   - Queries deps.dev API for package metadata and health metrics
   - Methods: `GetPackage()`, `ComputeHealthMetrics()`
   - Provides maintenance scores, update frequency, and version information

3. **SPDX Client** (`internal/providers/spdx/`):
   - Embedded license database (no external API calls)
   - Methods: `GetLicense()`, `SearchLicenses()`, `ListCategories()`, `GetLicensesByCategory()`
   - Contains comprehensive SPDX license information with compatibility ratings

### Creating New Providers
1. Create a new directory under `internal/providers/`
2. Implement a `Client` struct with a `NewClient(logger *zap.Logger)` constructor
3. Add methods for API interactions with proper error handling
4. Include comprehensive tests in `client_test.go`
5. Use structured logging for debugging API calls

## Tool Implementation Patterns

### Tool Structure
```go
// 1. Define input struct with JSON tags
type ToolInput struct {
    Field1 string `json:"field1"`
    Field2 int    `json:"field2"`
}

// 2. Define output struct
type ToolOutput struct {
    Result string `json:"result"`
    Data   []Item `json:"data"`
}

// 3. Implement handler
func (tr *ToolRegistry) HandleTool(ctx context.Context, input ToolInput) (*mcp.CallToolResult, error) {
    // Check cache first
    cacheKey := fmt.Sprintf("tool:%s:%s", input.Field1, input.Field2)
    if cached, ok := tr.cache.Get(cacheKey); ok {
        return cached.(*mcp.CallToolResult), nil
    }
    
    // Perform work...
    output := &ToolOutput{...}
    
    // Marshal to JSON
    jsonData, err := json.MarshalIndent(output, "", "  ")
    if err != nil {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to format output: %v", err)}},
        }, nil
    }
    
    result := &mcp.CallToolResult{
        Content: []mcp.Content{&mcp.TextContent{Text: string(jsonData)}},
    }
    
    // Cache result
    tr.cache.Set(cacheKey, result, 5*time.Minute)
    
    return result, nil
}

// 4. Register in ToolRegistry.Register()
srv.AddTool(
    &mcp.Tool{
        Name:        "tool.name",
        Description: "Clear description of what the tool does",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "field1": map[string]interface{}{
                    "type":        "string",
                    "description": "Description of field1",
                },
            },
            "required": []string{"field1"},
        },
    },
    handlerFunc,
)
srv.IncrementToolCount()
```

### Error Handling Best Practices
- Return `*mcp.CallToolResult` with `IsError: true` for user-facing errors
- Use structured logging for debugging (not return values)
- Validate inputs early and return clear error messages
- Cache both successful and error responses when appropriate

## Integration Points
- External vulnerability databases (OSV, NVD, Snyk APIs)
- Package registries (npm, PyPI, crates.io, Maven Central)
- CI/CD systems (GitHub Actions, GitLab CI, Jenkins)
- MCP clients (Claude Desktop, VSCode extensions, custom integrations)

## Testing Patterns

### Unit Test Structure
Follow the pattern in `internal/tools/tools_test.go` and `internal/providers/*/client_test.go`:

```go
func TestToolHandler(t *testing.T) {
    // Setup
    logger, _ := zap.NewDevelopment()
    srv, err := hypermcp.New(hypermcp.Config{
        Name:         "test",
        Version:      "1.0.0",
        CacheEnabled: true,
        CacheConfig:  cache.Config{...},
    }, logger)
    require.NoError(t, err)
    
    registry, err := NewToolRegistry(logger, srv.Cache())
    require.NoError(t, err)
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Test cases
    tests := []struct {
        name      string
        input     ToolInput
        wantError bool
    }{
        {
            name: "valid input",
            input: ToolInput{Field1: "value"},
            wantError: false,
        },
        {
            name: "invalid input",
            input: ToolInput{Field1: ""},
            wantError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := registry.HandleTool(ctx, tt.input)
            if tt.wantError {
                require.True(t, result.IsError)
            } else {
                require.False(t, result.IsError)
                // Additional assertions...
            }
        })
    }
}
```

### Test Best Practices
- Use table-driven tests for comprehensive coverage
- Include both success and error cases
- Test with realistic data (actual package names, versions)
- Handle network failures gracefully in provider tests
- Use `context.WithTimeout()` for tests that make external calls
- Log results with `t.Logf()` for debugging

## Example Workflows

### Adding a New Tool
1. Define input/output structs in `internal/tools/tools.go`
2. Implement handler method following the pattern above
3. Register in `ToolRegistry.Register()` method
4. Add comprehensive tests in `internal/tools/tools_test.go`
5. Update documentation if the tool adds significant functionality

### Adding a New Provider
1. Create directory `internal/providers/newprovider/`
2. Implement `client.go` with `Client` struct and methods
3. Add `client_test.go` with comprehensive tests
4. Integrate into `ToolRegistry` in `internal/tools/tools.go`
5. Use the provider in tool handlers

### Debugging Issues
1. Run server with development logger: Change `zap.NewProduction()` to `zap.NewDevelopment()` in `main.go`
2. Check logs for structured debug output from providers
3. Test individual tools with MCP Inspector
4. Run specific tests with `-v` flag: `go test -v -run TestName ./...`
5. Use cache stats to verify caching behavior

## Supported Ecosystems & Examples

### Tool Usage Examples

**deps.vulns** - Check for vulnerabilities:
```json
{
  "ecosystem": "npm",
  "package": "lodash",
  "version": "4.17.19"
}
```

**deps.health** - Get package health metrics:
```json
{
  "ecosystem": "pypi",
  "package": "requests"
}
```

**license.info** - Query license information:
```json
{
  "license_id": "MIT"
}
```

**deps.upgrade_plan** - Generate upgrade recommendations:
```json
{
  "ecosystem": "npm",
  "package": "express",
  "current_version": "4.17.1"
}
```

### Supported Package Ecosystems
- `npm` (Node.js/JavaScript)
- `pypi` (Python)
- `Go` (Go modules)
- `Maven` (Java)
- `cargo` (Rust)
- `NuGet` (.NET)

Use exact ecosystem names as shown above in tool inputs.
