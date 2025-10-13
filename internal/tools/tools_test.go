package tools

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rayprogramming/hypermcp"
	"github.com/rayprogramming/hypermcp/cache"
	"go.uber.org/zap"
)

func TestVulnsHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Create a test server to get the cache
	srv, err := hypermcp.New(hypermcp.Config{
		Name:         "test",
		Version:      "1.0.0",
		CacheEnabled: true,
		CacheConfig: cache.Config{
			MaxCost:     100 * 1024 * 1024,
			NumCounters: 10000,
			BufferItems: 64,
		},
	}, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	registry, err := NewToolRegistry(logger, srv.Cache())
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name      string
		input     VulnsInput
		wantError bool
	}{
		{
			name: "npm package with known vulns",
			input: VulnsInput{
				Ecosystem: "npm",
				Package:   "lodash",
				Version:   "4.17.19",
			},
			wantError: false,
		},
		{
			name: "go package",
			input: VulnsInput{
				Ecosystem: "Go",
				Package:   "github.com/gin-gonic/gin",
				Version:   "1.7.0",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.HandleVulns(ctx, tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("HandleVulns() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if err == nil {
				t.Logf("Found %d vulnerabilities for %s", result.VulnerabilityCount, tt.input.Package)
				t.Logf("Summary: Critical=%d High=%d Medium=%d Low=%d Unknown=%d",
					result.Summary.Critical, result.Summary.High,
					result.Summary.Medium, result.Summary.Low, result.Summary.Unknown)

				if result.VulnerabilityCount > 0 {
					t.Logf("First vulnerability: %s - %s",
						result.Vulnerabilities[0].ID,
						result.Vulnerabilities[0].Summary)
				}
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Create a test server to get the cache
	srv, err := hypermcp.New(hypermcp.Config{
		Name:         "test",
		Version:      "1.0.0",
		CacheEnabled: true,
		CacheConfig: cache.Config{
			MaxCost:     100 * 1024 * 1024,
			NumCounters: 10000,
			BufferItems: 64,
		},
	}, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	registry, err := NewToolRegistry(logger, srv.Cache())
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name      string
		input     VulnsInput
		wantError bool
	}{
		{
			name: "npm express package",
			input: VulnsInput{
				Ecosystem: "npm",
				Package:   "express",
			},
			wantError: false,
		},
		{
			name: "pypi requests package",
			input: VulnsInput{
				Ecosystem: "pypi",
				Package:   "requests",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate MCP CallToolRequest
			args, _ := json.Marshal(tt.input)
			req := &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{
					Name:      "deps.health",
					Arguments: args,
				},
			}

			result, err := registry.HandleHealth(ctx, req)
			if (err != nil) != tt.wantError {
				t.Errorf("HandleHealth() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err == nil && !result.IsError {
				// Parse the health metrics from response
				if len(result.Content) > 0 {
					if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
						t.Logf("Health metrics for %s:\n%s", tt.input.Package, textContent.Text)
					}
				}
			} else if result.IsError {
				t.Errorf("HandleHealth() returned error result")
			}
		})
	}
}

func TestLicenseHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Create a test server to get the cache
	srv, err := hypermcp.New(hypermcp.Config{
		Name:         "test",
		Version:      "1.0.0",
		CacheEnabled: true,
		CacheConfig: cache.Config{
			MaxCost:     100 * 1024 * 1024,
			NumCounters: 10000,
			BufferItems: 64,
		},
	}, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	registry, err := NewToolRegistry(logger, srv.Cache())
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tests := []struct {
		name      string
		input     LicenseInput
		wantError bool
		checkOSI  bool
	}{
		{
			name: "MIT License",
			input: LicenseInput{
				LicenseID: "MIT",
			},
			wantError: false,
			checkOSI:  true,
		},
		{
			name: "Apache-2.0 License",
			input: LicenseInput{
				LicenseID: "Apache-2.0",
			},
			wantError: false,
			checkOSI:  true,
		},
		{
			name: "GPL-3.0 License",
			input: LicenseInput{
				LicenseID: "GPL-3.0",
			},
			wantError: false,
			checkOSI:  true,
		},
		{
			name: "Case insensitive lookup",
			input: LicenseInput{
				LicenseID: "mit",
			},
			wantError: false,
			checkOSI:  true,
		},
		{
			name: "Unknown license",
			input: LicenseInput{
				LicenseID: "UNKNOWN-LICENSE-XYZ",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.HandleLicense(ctx, tt.input)
			if err != nil {
				t.Errorf("HandleLicense() unexpected error: %v", err)
				return
			}

			if tt.wantError {
				// Expect an error result
				if !result.IsError {
					t.Errorf("Expected error result for unknown license, got success")
				} else {
					// Log the error message
					if len(result.Content) > 0 {
						if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
							t.Logf("Expected error message: %s", textContent.Text)
						}
					}
				}
				return
			}

			// For successful cases
			if result.IsError {
				if len(result.Content) > 0 {
					if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
						t.Errorf("Unexpected error: %s", textContent.Text)
					}
				}
				return
			}

			// Parse the license info from response
			if len(result.Content) > 0 {
				if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
					t.Logf("License info for %s:\n%s", tt.input.LicenseID, textContent.Text)

					// Verify it's valid JSON
					var licenseInfo map[string]interface{}
					if err := json.Unmarshal([]byte(textContent.Text), &licenseInfo); err != nil {
						t.Errorf("Failed to parse license info JSON: %v", err)
					}
				}
			}
		})
	}
}

func TestUpgradePlanHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Create a test server to get the cache
	srv, err := hypermcp.New(hypermcp.Config{
		Name:         "test",
		Version:      "1.0.0",
		CacheEnabled: true,
		CacheConfig: cache.Config{
			MaxCost:     100 * 1024 * 1024,
			NumCounters: 10000,
			BufferItems: 64,
		},
	}, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	registry, err := NewToolRegistry(logger, srv.Cache())
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name           string
		input          UpgradePlanInput
		wantError      bool
		expectUrgent   bool
		expectWarning  bool
		expectUpToDate bool
	}{
		{
			name: "npm package with known vulnerabilities",
			input: UpgradePlanInput{
				Ecosystem:      "npm",
				Package:        "lodash",
				CurrentVersion: "4.17.19",
			},
			wantError:    false,
			expectUrgent: true, // lodash 4.17.19 has known vulnerabilities
		},
		{
			name: "pypi package - requests (check if up to date or needs upgrade)",
			input: UpgradePlanInput{
				Ecosystem:      "pypi",
				Package:        "requests",
				CurrentVersion: "2.25.0",
			},
			wantError: false,
			// Will check if upgrade is recommended
		},
		{
			name: "npm express - current stable version",
			input: UpgradePlanInput{
				Ecosystem:      "npm",
				Package:        "express",
				CurrentVersion: "4.18.2",
			},
			wantError: false,
			// May or may not be up to date depending on latest version
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.HandleUpgradePlan(ctx, tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("HandleUpgradePlan() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err == nil && !result.IsError {
				// Parse the upgrade plan from response
				if len(result.Content) > 0 {
					if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
						t.Logf("Upgrade plan for %s@%s:\n%s",
							tt.input.Package, tt.input.CurrentVersion, textContent.Text)

						// Parse JSON to verify structure
						var plan UpgradePlanOutput
						if err := json.Unmarshal([]byte(textContent.Text), &plan); err != nil {
							t.Errorf("Failed to parse upgrade plan JSON: %v", err)
							return
						}

						// Validate output structure
						if plan.Package != tt.input.Package {
							t.Errorf("Expected package %s, got %s", tt.input.Package, plan.Package)
						}
						if plan.CurrentVersion != tt.input.CurrentVersion {
							t.Errorf("Expected current version %s, got %s",
								tt.input.CurrentVersion, plan.CurrentVersion)
						}
						if plan.LatestVersion == "" {
							t.Error("Latest version should not be empty")
						}
						if plan.Priority == "" {
							t.Error("Priority should not be empty")
						}
						if plan.Recommendation == "" {
							t.Error("Recommendation should not be empty")
						}

						// Check expectations
						if tt.expectUrgent && plan.Priority != "URGENT" {
							t.Logf("Note: Expected URGENT priority for %s, got %s",
								tt.input.Package, plan.Priority)
						}
						if tt.expectUpToDate && !plan.IsUpToDate {
							t.Errorf("Expected package to be up to date")
						}

						// Log key findings
						t.Logf("  Priority: %s", plan.Priority)
						t.Logf("  Up to date: %v", plan.IsUpToDate)
						t.Logf("  Has vulnerabilities: %v (count: %d)",
							plan.HasVulnerabilities, plan.VulnerabilityCount)
						t.Logf("  Maintenance: %s (score: %.1f)",
							plan.MaintenanceLevel, plan.MaintenanceScore)
						t.Logf("  Breaking changes possible: %v", plan.BreakingChanges)
					}
				}
			} else if result.IsError {
				t.Errorf("HandleUpgradePlan() returned error result")
			}
		})
	}
}
