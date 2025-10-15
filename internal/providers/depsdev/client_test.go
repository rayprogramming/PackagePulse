package depsdev

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestDepsDevClientGetPackage(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewClient(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name      string
		ecosystem string
		pkg       string
		wantError bool
	}{
		{
			name:      "npm express - popular package",
			ecosystem: "npm",
			pkg:       "express",
			wantError: false,
		},
		{
			name:      "pypi requests - popular package",
			ecosystem: "pypi",
			pkg:       "requests",
			wantError: false,
		},
		{
			name:      "npm lodash",
			ecosystem: "npm",
			pkg:       "lodash",
			wantError: false,
		},
		{
			name:      "invalid package",
			ecosystem: "npm",
			pkg:       "this-package-absolutely-does-not-exist-12345",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.GetPackage(ctx, tt.ecosystem, tt.pkg)
			if (err != nil) != tt.wantError {
				t.Errorf("GetPackage() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if err == nil {
				t.Logf("Package: %s", result.PackageKey.Name)
				t.Logf("Versions: %d", len(result.Versions))
				t.Logf("Links: %d", len(result.Links))

				// Test health metrics computation
				metrics := ComputeHealthMetrics(result)
				t.Logf("\nHealth Metrics:")
				t.Logf("  Latest Version: %s", metrics.LatestVersion)
				t.Logf("  Days Since Update: %d", metrics.DaysSinceUpdate)
				t.Logf("  Maintenance Score: %.1f/100", metrics.MaintenanceScore)
				t.Logf("  Maintenance Level: %s", metrics.MaintenanceLevel)
				t.Logf("  Has Repository: %v", metrics.HasRepository)
				t.Logf("  Has Documentation: %v", metrics.HasDocumentation)
				t.Logf("  Recommendation: %s", metrics.Recommendation)

				// Validate metrics make sense
				if metrics.VersionCount != len(result.Versions) {
					t.Errorf("Version count mismatch: got %d, want %d", metrics.VersionCount, len(result.Versions))
				}
				if metrics.MaintenanceScore < 0 || metrics.MaintenanceScore > 100 {
					t.Errorf("Invalid maintenance score: %.1f (should be 0-100)", metrics.MaintenanceScore)
				}
			}
		})
	}
}

func TestComputeHealthMetrics(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		pkg          *PackageInfo
		wantLevel    string
		wantMinScore float64
		wantMaxScore float64
	}{
		{
			name: "excellent package - recent, many versions, docs",
			pkg: &PackageInfo{
				PackageKey: PackageKey{Name: "test-excellent", System: "npm"},
				Versions:   make([]VersionInfo, 60),
				Links: []Link{
					{Label: "SOURCE_REPO", URL: "https://github.com/test/test"},
					{Label: "DOCUMENTATION", URL: "https://test.dev"},
				},
			},
			wantLevel:    "excellent",
			wantMinScore: 80.0,
			wantMaxScore: 100.0,
		},
		{
			name: "good package - regular updates",
			pkg: &PackageInfo{
				PackageKey: PackageKey{Name: "test-good", System: "npm"},
				Versions:   make([]VersionInfo, 25),
				Links: []Link{
					{Label: "SOURCE_REPO", URL: "https://github.com/test/good"},
				},
			},
			wantLevel:    "good",
			wantMinScore: 60.0,
			wantMaxScore: 80.0,
		},
		{
			name: "poor maintenance - old package",
			pkg: &PackageInfo{
				PackageKey: PackageKey{Name: "test-old", System: "npm"},
				Versions: []VersionInfo{
					{
						VersionKey:  VersionKey{Version: "1.0.0"},
						PublishedAt: now.Add(-200 * 24 * time.Hour), // ~200 days old
						IsDefault:   true,
					},
				},
			},
			wantLevel:    "critical", // 200 days + 1 version = critical
			wantMinScore: 0.0,
			wantMaxScore: 20.0,
		},
		{
			name: "critical - abandoned package",
			pkg: &PackageInfo{
				PackageKey: PackageKey{Name: "abandoned", System: "npm"},
				Versions: []VersionInfo{
					{
						VersionKey:  VersionKey{Version: "0.1.0"},
						PublishedAt: now.Add(-800 * 24 * time.Hour), // ~800 days old
						IsDefault:   true,
					},
				},
			},
			wantLevel:    "critical",
			wantMinScore: 0.0,
			wantMaxScore: 20.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up versions with publication dates
			switch tt.name {
			case "excellent package - recent, many versions, docs":
				for i := range tt.pkg.Versions {
					tt.pkg.Versions[i] = VersionInfo{
						VersionKey:  VersionKey{Version: fmt.Sprintf("1.%d.0", i)},
						PublishedAt: now.Add(-10 * 24 * time.Hour),
						IsDefault:   i == len(tt.pkg.Versions)-1,
						Licenses:    []string{"MIT"},
					}
				}
			case "good package - regular updates":
				for i := range tt.pkg.Versions {
					tt.pkg.Versions[i] = VersionInfo{
						VersionKey:  VersionKey{Version: fmt.Sprintf("2.%d.0", i)},
						PublishedAt: now.Add(-60 * 24 * time.Hour),
						IsDefault:   i == len(tt.pkg.Versions)-1,
						Licenses:    []string{"Apache-2.0"},
					}
				}
			}

			metrics := ComputeHealthMetrics(tt.pkg)

			if metrics.MaintenanceLevel != tt.wantLevel {
				t.Errorf("MaintenanceLevel = %s, want %s", metrics.MaintenanceLevel, tt.wantLevel)
			}

			if metrics.MaintenanceScore < tt.wantMinScore || metrics.MaintenanceScore > tt.wantMaxScore {
				t.Errorf("MaintenanceScore = %.1f, want between %.1f and %.1f",
					metrics.MaintenanceScore, tt.wantMinScore, tt.wantMaxScore)
			}

			if metrics.Recommendation == "" {
				t.Error("Recommendation should not be empty")
			}

			t.Logf("Package: %s", metrics.PackageName)
			t.Logf("Score: %.1f (%s)", metrics.MaintenanceScore, metrics.MaintenanceLevel)
			t.Logf("Recommendation: %s", metrics.Recommendation)
		})
	}
}

func TestHealthMetricsScoring(t *testing.T) {
	// Test scoring components individually
	now := time.Now()

	t.Run("recent update scoring", func(t *testing.T) {
		testCases := []struct {
			daysOld     int
			minExpected float64
		}{
			{10, 40.0},  // Within 30 days -> 40 points
			{60, 30.0},  // 31-90 days -> 30 points
			{120, 20.0}, // 91-180 days -> 20 points
			{200, 10.0}, // 181-365 days -> 10 points
			{400, 0.0},  // > 365 days -> 0 points
		}

		for _, tc := range testCases {
			pkg := &PackageInfo{
				PackageKey: PackageKey{Name: "test", System: "npm"},
				Versions: []VersionInfo{
					{
						PublishedAt: now.Add(-time.Duration(tc.daysOld) * 24 * time.Hour),
						IsDefault:   true,
					},
				},
			}
			metrics := ComputeHealthMetrics(pkg)
			if metrics.MaintenanceScore < tc.minExpected {
				t.Errorf("For %d days old: score=%.1f, expected at least %.1f",
					tc.daysOld, metrics.MaintenanceScore, tc.minExpected)
			}
		}
	})
}
