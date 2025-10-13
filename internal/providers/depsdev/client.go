package depsdev

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

const (
	depsDevBaseURL = "https://api.deps.dev/v3alpha"
	apiTimeout     = 30 * time.Second
)

// Client handles deps.dev API interactions
type Client struct {
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a new deps.dev API client
func NewClient(logger *zap.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: apiTimeout,
		},
		logger: logger,
	}
}

// PackageInfo contains metadata about a package
type PackageInfo struct {
	PackageKey PackageKey    `json:"packageKey"`
	Versions   []VersionInfo `json:"versions,omitempty"`
	Links      []Link        `json:"links,omitempty"`
}

// PackageKey identifies a package in an ecosystem
type PackageKey struct {
	System string `json:"system"`
	Name   string `json:"name"`
}

// VersionInfo contains metadata about a specific version
type VersionInfo struct {
	VersionKey      VersionKey    `json:"versionKey"`
	PublishedAt     time.Time     `json:"publishedAt"`
	IsDefault       bool          `json:"isDefault"`
	Licenses        []string      `json:"licenses,omitempty"`
	Links           []Link        `json:"links,omitempty"`
	SlsaProvenances []interface{} `json:"slsaProvenances,omitempty"`
	RelationCount   int           `json:"relationCount,omitempty"`
}

// VersionKey identifies a specific package version
type VersionKey struct {
	System  string `json:"system"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Link represents external links
type Link struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// HealthMetrics computed from package metadata
type HealthMetrics struct {
	PackageName      string    `json:"package_name"`
	Ecosystem        string    `json:"ecosystem"`
	LatestVersion    string    `json:"latest_version"`
	VersionCount     int       `json:"version_count"`
	LastPublished    time.Time `json:"last_published"`
	DaysSinceUpdate  int       `json:"days_since_update"`
	HasRepository    bool      `json:"has_repository"`
	HasDocumentation bool      `json:"has_documentation"`
	LicenseCount     int       `json:"license_count"`
	MaintenanceScore float64   `json:"maintenance_score"`
	MaintenanceLevel string    `json:"maintenance_level"`
	Recommendation   string    `json:"recommendation"`
}

// GetPackage retrieves package information from deps.dev
// Example: client.GetPackage(ctx, "npm", "express")
func (c *Client) GetPackage(ctx context.Context, ecosystem, name string) (*PackageInfo, error) {
	c.logger.Debug("querying deps.dev", zap.String("ecosystem", ecosystem), zap.String("package", name))

	escapedName := url.PathEscape(name)
	endpoint := fmt.Sprintf("%s/systems/%s/packages/%s", depsDevBaseURL, ecosystem, escapedName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.logger.Debug("querying deps.dev",
		zap.String("ecosystem", ecosystem),
		zap.String("package", name))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package not found: %s/%s", ecosystem, name)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("deps.dev API error: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var result PackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	c.logger.Debug("deps.dev query complete",
		zap.Int("versions", len(result.Versions)))

	return &result, nil
}

// ComputeHealthMetrics calculates health metrics from package info
func ComputeHealthMetrics(pkg *PackageInfo) *HealthMetrics {
	metrics := &HealthMetrics{
		PackageName:  pkg.PackageKey.Name,
		Ecosystem:    pkg.PackageKey.System,
		VersionCount: len(pkg.Versions),
	}

	// Find latest version and publication date
	var latestPub time.Time
	for _, v := range pkg.Versions {
		if v.IsDefault {
			metrics.LatestVersion = v.VersionKey.Version
			metrics.LicenseCount = len(v.Licenses)
		}
		if v.PublishedAt.After(latestPub) {
			latestPub = v.PublishedAt
		}
	}
	metrics.LastPublished = latestPub

	if !latestPub.IsZero() {
		metrics.DaysSinceUpdate = int(time.Since(latestPub).Hours() / 24)
	}

	// Check for repository and documentation
	for _, link := range pkg.Links {
		if link.Label == "SOURCE_REPO" || link.Label == "REPOSITORY" {
			metrics.HasRepository = true
		}
		if link.Label == "DOCUMENTATION" {
			metrics.HasDocumentation = true
		}
	}

	// Compute maintenance score (0-100)
	score := 0.0

	// Recent updates (40 points max)
	if metrics.DaysSinceUpdate <= 30 {
		score += 40.0
	} else if metrics.DaysSinceUpdate <= 90 {
		score += 30.0
	} else if metrics.DaysSinceUpdate <= 180 {
		score += 20.0
	} else if metrics.DaysSinceUpdate <= 365 {
		score += 10.0
	}

	// Version count (20 points max)
	if metrics.VersionCount >= 50 {
		score += 20.0
	} else if metrics.VersionCount >= 20 {
		score += 15.0
	} else if metrics.VersionCount >= 10 {
		score += 10.0
	} else if metrics.VersionCount >= 5 {
		score += 5.0
	}

	// Repository (20 points)
	if metrics.HasRepository {
		score += 20.0
	}

	// Documentation (10 points)
	if metrics.HasDocumentation {
		score += 10.0
	}

	// License (10 points)
	if metrics.LicenseCount > 0 {
		score += 10.0
	}

	metrics.MaintenanceScore = score

	// Assign maintenance level and recommendation
	if score >= 80 {
		metrics.MaintenanceLevel = "excellent"
		metrics.Recommendation = "This package is actively maintained with good development practices."
	} else if score >= 60 {
		metrics.MaintenanceLevel = "good"
		metrics.Recommendation = "Package shows regular maintenance and good health indicators."
	} else if score >= 40 {
		metrics.MaintenanceLevel = "fair"
		metrics.Recommendation = "Package is maintained but may have slower update cycles. Review before use."
	} else if score >= 20 {
		metrics.MaintenanceLevel = "poor"
		metrics.Recommendation = "WARNING: Package shows signs of poor maintenance. Consider alternatives."
	} else {
		metrics.MaintenanceLevel = "critical"
		metrics.Recommendation = "CRITICAL: Package appears abandoned or unmaintained. Strongly consider alternatives."
	}

	return metrics
}
