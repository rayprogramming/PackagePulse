package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rayprogramming/PackagePulse/internal/providers/depsdev"
	"github.com/rayprogramming/PackagePulse/internal/providers/osv"
	"github.com/rayprogramming/PackagePulse/internal/providers/spdx"
	"github.com/rayprogramming/hypermcp"
	"github.com/rayprogramming/hypermcp/cache"
	"go.uber.org/zap"
)

// ToolRegistry manages all MCP tools
type ToolRegistry struct {
	osvClient     *osv.Client
	depsDevClient *depsdev.Client
	spdxClient    *spdx.Client
	logger        *zap.Logger
	cache         *cache.Cache
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry(logger *zap.Logger, c *cache.Cache) (*ToolRegistry, error) {
	return &ToolRegistry{
		osvClient:     osv.NewClient(logger),
		depsDevClient: depsdev.NewClient(logger),
		spdxClient:    spdx.NewClient(logger),
		logger:        logger,
		cache:         c,
	}, nil
}

// VulnsInput defines input for deps.vulns tool
type VulnsInput struct {
	Ecosystem string `json:"ecosystem"`
	Package   string `json:"package"`
	Version   string `json:"version,omitempty"`
}

// VulnsOutput contains vulnerability results
type VulnsOutput struct {
	Package            string              `json:"package"`
	Ecosystem          string              `json:"ecosystem"`
	Version            string              `json:"version,omitempty"`
	VulnerabilityCount int                 `json:"vulnerability_count"`
	Vulnerabilities    []osv.Vulnerability `json:"vulnerabilities"`
	Summary            VulnSummary         `json:"summary"`
}

// VulnSummary provides aggregated vulnerability statistics
type VulnSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Unknown  int `json:"unknown"`
}

// HandleVulns implements deps.vulns tool
// Example: {"ecosystem": "npm", "package": "lodash", "version": "4.17.19"}
func (tr *ToolRegistry) HandleVulns(ctx context.Context, input VulnsInput) (*VulnsOutput, error) {
	cacheKey := fmt.Sprintf("vulns:%s:%s:%s", input.Ecosystem, input.Package, input.Version)

	// Check cache
	if tr.cache != nil {
		if cached, found := tr.cache.Get(cacheKey); found {
			tr.logger.Debug("cache hit", zap.String("key", cacheKey))
			if output, ok := cached.(*VulnsOutput); ok {
				return output, nil
			}
		}
		tr.logger.Debug("cache miss", zap.String("key", cacheKey))
	}

	// Query OSV
	result, err := tr.osvClient.Query(ctx, input.Ecosystem, input.Package, input.Version)
	if err != nil {
		return nil, fmt.Errorf("query OSV: %w", err)
	}

	// Compute summary
	summary := VulnSummary{}
	for _, vuln := range result.Vulns {
		severity := "unknown"
		if len(vuln.Severity) > 0 {
			severity = vuln.Severity[0].Score
		}

		switch {
		case containsIgnoreCase(severity, "critical"):
			summary.Critical++
		case containsIgnoreCase(severity, "high"):
			summary.High++
		case containsIgnoreCase(severity, "medium"):
			summary.Medium++
		case containsIgnoreCase(severity, "low"):
			summary.Low++
		default:
			summary.Unknown++
		}
	}

	output := &VulnsOutput{
		Package:            input.Package,
		Ecosystem:          input.Ecosystem,
		Version:            input.Version,
		VulnerabilityCount: len(result.Vulns),
		Vulnerabilities:    result.Vulns,
		Summary:            summary,
	}

	// Cache result (5 minutes TTL)
	if tr.cache != nil {
		tr.cache.Set(cacheKey, output, 5*time.Minute)
	}

	return output, nil
}

// Register registers all tools with the server
func (tr *ToolRegistry) Register(srv *hypermcp.Server) error {
	mcpServer := srv.MCP()

	// deps.vulns - Vulnerability scanning tool
	mcpServer.AddTool(
		&mcp.Tool{
			Name:        "deps.vulns",
			Description: "Query OSV.dev for known vulnerabilities in a package. Supports npm, PyPI, Go, Maven, Cargo, and NuGet ecosystems.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ecosystem": map[string]interface{}{
						"type":        "string",
						"description": "Package ecosystem (npm, pypi, Go, maven, cargo, nuget)",
					},
					"package": map[string]interface{}{
						"type":        "string",
						"description": "Package name (e.g., 'lodash' for npm, 'github.com/gin-gonic/gin' for Go)",
					},
					"version": map[string]interface{}{
						"type":        "string",
						"description": "Specific version to check (optional, omit to check all versions)",
					},
				},
				"required": []string{"ecosystem", "package"},
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var params VulnsInput
			if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{
						Text: fmt.Sprintf("Invalid input: %v", err),
					}},
					IsError: true,
				}, nil
			}

			result, err := tr.HandleVulns(ctx, params)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{
						Text: err.Error(),
					}},
					IsError: true,
				}, nil
			}

			data, _ := json.MarshalIndent(result, "", "  ")
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{
					Text: string(data),
				}},
			}, nil
		},
	)
	srv.IncrementToolCount()

	// deps.health - Package health metrics tool
	mcpServer.AddTool(
		&mcp.Tool{
			Name:        "deps.health",
			Description: "Query deps.dev for package health metrics including maintenance score, update frequency, and recommendations. Supports npm, pypi, Go, and other ecosystems.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ecosystem": map[string]interface{}{
						"type":        "string",
						"description": "Package ecosystem (npm, pypi, go, maven, cargo, nuget)",
					},
					"package": map[string]interface{}{
						"type":        "string",
						"description": "Package name (e.g., 'express' for npm, 'requests' for pypi)",
					},
				},
				"required": []string{"ecosystem", "package"},
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tr.HandleHealth(ctx, req)
		},
	)
	srv.IncrementToolCount()

	// license.info - SPDX license information tool
	mcpServer.AddTool(
		&mcp.Tool{
			Name:        "license.info",
			Description: "Query SPDX license database for detailed license information including OSI approval status, compatibility, and category. Supports all standard SPDX license identifiers.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"license_id": map[string]interface{}{
						"type":        "string",
						"description": "SPDX license identifier (e.g., 'MIT', 'Apache-2.0', 'GPL-3.0')",
					},
				},
				"required": []string{"license_id"},
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var params LicenseInput
			if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{
						Text: fmt.Sprintf("Invalid input: %v", err),
					}},
					IsError: true,
				}, nil
			}

			return tr.HandleLicense(ctx, params)
		},
	)
	srv.IncrementToolCount()

	// deps.upgrade_plan - Smart upgrade recommendations tool
	mcpServer.AddTool(
		&mcp.Tool{
			Name:        "deps.upgrade_plan",
			Description: "Generate smart upgrade recommendations by analyzing vulnerabilities, package health, and maintenance status. Provides priority-based upgrade advice and checks for potential breaking changes.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ecosystem": map[string]interface{}{
						"type":        "string",
						"description": "Package ecosystem (npm, pypi, go, maven, cargo, nuget)",
					},
					"package": map[string]interface{}{
						"type":        "string",
						"description": "Package name (e.g., 'lodash' for npm, 'requests' for pypi)",
					},
					"current_version": map[string]interface{}{
						"type":        "string",
						"description": "Current version in use (e.g., '4.17.19')",
					},
				},
				"required": []string{"ecosystem", "package", "current_version"},
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var params UpgradePlanInput
			if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{
						Text: fmt.Sprintf("Invalid input: %v", err),
					}},
					IsError: true,
				}, nil
			}

			return tr.HandleUpgradePlan(ctx, params)
		},
	)
	srv.IncrementToolCount()

	return nil
}

// HandleHealth implements the deps.health tool
func (tr *ToolRegistry) HandleHealth(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input VulnsInput // Reuse same input structure (ecosystem, package, version optional)
	if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Invalid input: %v", err)}},
		}, nil
	}

	// Check cache first
	cacheKey := fmt.Sprintf("health:%s:%s", input.Ecosystem, input.Package)
	if cached, ok := tr.cache.Get(cacheKey); ok {
		tr.logger.Debug("cache hit", zap.String("key", cacheKey))
		if healthMetrics, ok := cached.(*depsdev.HealthMetrics); ok {
			output, _ := json.MarshalIndent(healthMetrics, "", "  ")
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(output)}},
			}, nil
		}
	}

	// Query deps.dev API
	pkgInfo, err := tr.depsDevClient.GetPackage(ctx, input.Ecosystem, input.Package)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to query deps.dev: %v", err)}},
		}, nil
	}

	// Compute health metrics
	healthMetrics := depsdev.ComputeHealthMetrics(pkgInfo)

	// Cache the result
	tr.cache.Set(cacheKey, healthMetrics, 5*time.Minute)

	// Return formatted output
	output, err := json.MarshalIndent(healthMetrics, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to format output: %v", err)}},
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(output)}},
	}, nil
}

// LicenseInput defines input for license.info tool
type LicenseInput struct {
	LicenseID string `json:"license_id"`
}

// HandleLicense retrieves information about a specific SPDX license
func (tr *ToolRegistry) HandleLicense(ctx context.Context, input LicenseInput) (*mcp.CallToolResult, error) {
	tr.logger.Info("Handling license query", zap.String("license_id", input.LicenseID))

	// Validate input
	if input.LicenseID == "" {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "license_id is required"}},
		}, nil
	}

	// Check cache first
	cacheKey := fmt.Sprintf("license:%s", input.LicenseID)
	if cached, ok := tr.cache.Get(cacheKey); ok {
		tr.logger.Debug("cache hit", zap.String("key", cacheKey))
		if licenseInfo, ok := cached.(*spdx.LicenseInfo); ok {
			output, _ := json.MarshalIndent(licenseInfo, "", "  ")
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(output)}},
			}, nil
		}
	}

	// Query SPDX database
	licenseInfo, err := tr.spdxClient.GetLicense(ctx, input.LicenseID)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("License not found: %v", err)}},
		}, nil
	}

	// Cache the result (licenses don't change, so longer TTL)
	tr.cache.Set(cacheKey, licenseInfo, 24*time.Hour)

	// Return formatted output
	output, err := json.MarshalIndent(licenseInfo, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to format output: %v", err)}},
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(output)}},
	}, nil
}

// UpgradePlanInput defines input for deps.upgrade_plan tool
type UpgradePlanInput struct {
	Ecosystem      string `json:"ecosystem"`
	Package        string `json:"package"`
	CurrentVersion string `json:"current_version"`
}

// UpgradePlanOutput contains upgrade recommendations
type UpgradePlanOutput struct {
	Package              string       `json:"package"`
	Ecosystem            string       `json:"ecosystem"`
	CurrentVersion       string       `json:"current_version"`
	LatestVersion        string       `json:"latest_version"`
	IsUpToDate           bool         `json:"is_up_to_date"`
	HasVulnerabilities   bool         `json:"has_vulnerabilities"`
	VulnerabilityCount   int          `json:"vulnerability_count"`
	MaintenanceLevel     string       `json:"maintenance_level"`
	MaintenanceScore     float64      `json:"maintenance_score"`
	DaysSinceUpdate      int          `json:"days_since_update"`
	Priority             string       `json:"priority"`
	Recommendation       string       `json:"recommendation"`
	UpgradePath          []string     `json:"upgrade_path"`
	BreakingChanges      bool         `json:"breaking_changes_possible"`
	VulnerabilitySummary *VulnSummary `json:"vulnerability_summary,omitempty"`
}

// HandleUpgradePlan generates smart upgrade recommendations
func (tr *ToolRegistry) HandleUpgradePlan(ctx context.Context, input UpgradePlanInput) (*mcp.CallToolResult, error) {
	tr.logger.Info("Handling upgrade plan request",
		zap.String("ecosystem", input.Ecosystem),
		zap.String("package", input.Package),
		zap.String("current_version", input.CurrentVersion))

	// Validate input
	if input.Ecosystem == "" || input.Package == "" || input.CurrentVersion == "" {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "ecosystem, package, and current_version are required"}},
		}, nil
	}

	// Check cache first
	cacheKey := fmt.Sprintf("upgrade:%s:%s:%s", input.Ecosystem, input.Package, input.CurrentVersion)
	if cached, ok := tr.cache.Get(cacheKey); ok {
		tr.logger.Debug("cache hit", zap.String("key", cacheKey))
		if plan, ok := cached.(*UpgradePlanOutput); ok {
			output, _ := json.MarshalIndent(plan, "", "  ")
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(output)}},
			}, nil
		}
	}

	// Step 1: Check for vulnerabilities in current version
	tr.logger.Debug("Checking vulnerabilities", zap.String("version", input.CurrentVersion))
	vulnResp, err := tr.osvClient.Query(ctx, input.Ecosystem, input.Package, input.CurrentVersion)
	if err != nil {
		tr.logger.Warn("Failed to query vulnerabilities", zap.Error(err))
	}

	hasVulns := vulnResp != nil && len(vulnResp.Vulns) > 0
	vulnCount := 0
	var vulnSummary *VulnSummary
	if hasVulns {
		vulnCount = len(vulnResp.Vulns)
		summary := computeVulnSummary(vulnResp.Vulns)
		vulnSummary = &summary
	}

	// Step 2: Get package health and latest version
	tr.logger.Debug("Fetching package health")
	pkgInfo, err := tr.depsDevClient.GetPackage(ctx, input.Ecosystem, input.Package)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to query package info: %v", err)}},
		}, nil
	}

	healthMetrics := depsdev.ComputeHealthMetrics(pkgInfo)

	// Step 3: Analyze and generate recommendations
	plan := &UpgradePlanOutput{
		Package:              input.Package,
		Ecosystem:            input.Ecosystem,
		CurrentVersion:       input.CurrentVersion,
		LatestVersion:        healthMetrics.LatestVersion,
		IsUpToDate:           input.CurrentVersion == healthMetrics.LatestVersion,
		HasVulnerabilities:   hasVulns,
		VulnerabilityCount:   vulnCount,
		MaintenanceLevel:     healthMetrics.MaintenanceLevel,
		MaintenanceScore:     healthMetrics.MaintenanceScore,
		DaysSinceUpdate:      healthMetrics.DaysSinceUpdate,
		VulnerabilitySummary: vulnSummary,
		UpgradePath:          []string{input.CurrentVersion, healthMetrics.LatestVersion},
	}

	// Check for potential breaking changes (simplified semver check)
	plan.BreakingChanges = checkBreakingChanges(input.CurrentVersion, healthMetrics.LatestVersion)

	// Determine priority and recommendation
	if hasVulns {
		// URGENT: Security vulnerabilities present
		plan.Priority = "URGENT"
		criticalCount := 0
		highCount := 0
		if vulnSummary != nil {
			criticalCount = vulnSummary.Critical
			highCount = vulnSummary.High
		}

		if criticalCount > 0 {
			plan.Recommendation = fmt.Sprintf("CRITICAL: Upgrade immediately! Found %d critical vulnerabilities in current version.", criticalCount)
		} else if highCount > 0 {
			plan.Recommendation = fmt.Sprintf("URGENT: Upgrade to %s to address %d high-severity vulnerabilities.",
				healthMetrics.LatestVersion, highCount)
		} else {
			plan.Recommendation = fmt.Sprintf("URGENT: Upgrade to %s to address %d known vulnerabilities.",
				healthMetrics.LatestVersion, vulnCount)
		}
	} else if plan.IsUpToDate {
		// Already on latest version
		plan.Priority = "OK"
		if healthMetrics.MaintenanceLevel == "poor" || healthMetrics.MaintenanceLevel == "critical" {
			plan.Recommendation = fmt.Sprintf("On latest version, but package shows %s maintenance. Consider alternatives.",
				healthMetrics.MaintenanceLevel)
		} else {
			plan.Recommendation = "Already on latest version. No action needed."
		}
	} else {
		// Not on latest, no vulnerabilities
		if healthMetrics.MaintenanceLevel == "poor" || healthMetrics.MaintenanceLevel == "critical" {
			plan.Priority = "WARNING"
			plan.Recommendation = fmt.Sprintf("WARNING: Package shows %s maintenance (score: %.1f). Upgrade to %s available, but consider package alternatives.",
				healthMetrics.MaintenanceLevel, healthMetrics.MaintenanceScore, healthMetrics.LatestVersion)
		} else if healthMetrics.DaysSinceUpdate > 180 {
			plan.Priority = "LOW"
			plan.Recommendation = fmt.Sprintf("Upgrade available (%s), but no urgent issues. Current version is %d days old.",
				healthMetrics.LatestVersion, healthMetrics.DaysSinceUpdate)
		} else if plan.BreakingChanges {
			plan.Priority = "MEDIUM"
			plan.Recommendation = fmt.Sprintf("Upgrade to %s recommended, but may contain breaking changes. Review changelog before upgrading.",
				healthMetrics.LatestVersion)
		} else {
			plan.Priority = "RECOMMENDED"
			plan.Recommendation = fmt.Sprintf("Upgrade to %s recommended for latest features and improvements.",
				healthMetrics.LatestVersion)
		}
	}

	// Cache the result
	tr.cache.Set(cacheKey, plan, 5*time.Minute)

	// Return formatted output
	output, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to format output: %v", err)}},
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(output)}},
	}, nil
}

// checkBreakingChanges performs a simplified semver check
func checkBreakingChanges(current, latest string) bool {
	// Simple heuristic: if major version changes, assume breaking changes
	// This is a simplified check - real semver parsing would be more robust
	if len(current) == 0 || len(latest) == 0 {
		return false
	}

	// Extract first character (major version for simple cases like "1.2.3" vs "2.0.0")
	if current[0] != latest[0] {
		return true
	}

	return false
}

// computeVulnSummary analyzes vulnerabilities and returns a severity summary
func computeVulnSummary(vulns []osv.Vulnerability) VulnSummary {
	summary := VulnSummary{}
	for _, vuln := range vulns {
		severity := "unknown"
		if len(vuln.Severity) > 0 {
			severity = vuln.Severity[0].Score
		}

		switch {
		case containsIgnoreCase(severity, "critical"):
			summary.Critical++
		case containsIgnoreCase(severity, "high"):
			summary.High++
		case containsIgnoreCase(severity, "medium"):
			summary.Medium++
		case containsIgnoreCase(severity, "low"):
			summary.Low++
		default:
			summary.Unknown++
		}
	}
	return summary
}

// Helper function for case-insensitive substring matching
func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return findInString(s, substr)
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func findInString(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
