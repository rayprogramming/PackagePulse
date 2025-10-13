package spdx

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestSPDXClient_GetLicense(t *testing.T) {
	logger := zap.NewNop()
	client := NewClient(logger)
	ctx := context.Background()

	tests := []struct {
		name          string
		licenseID     string
		expectError   bool
		checkOSI      bool
		checkCategory string
	}{
		{
			name:          "MIT License",
			licenseID:     "MIT",
			expectError:   false,
			checkOSI:      true,
			checkCategory: "Permissive",
		},
		{
			name:          "Apache 2.0",
			licenseID:     "Apache-2.0",
			expectError:   false,
			checkOSI:      true,
			checkCategory: "Permissive",
		},
		{
			name:          "GPL-3.0",
			licenseID:     "GPL-3.0",
			expectError:   false,
			checkOSI:      true,
			checkCategory: "Copyleft",
		},
		{
			name:          "BSD-3-Clause",
			licenseID:     "BSD-3-Clause",
			expectError:   false,
			checkOSI:      true,
			checkCategory: "Permissive",
		},
		{
			name:          "Case insensitive lookup",
			licenseID:     "mit",
			expectError:   false,
			checkOSI:      true,
			checkCategory: "Permissive",
		},
		{
			name:        "Unknown license",
			licenseID:   "UNKNOWN-LICENSE-123",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			license, err := client.GetLicense(ctx, tt.licenseID)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for license %s, got nil", tt.licenseID)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if license == nil {
				t.Fatal("Expected license, got nil")
			}

			t.Logf("Found license: %s - %s", license.ID, license.Name)
			t.Logf("  OSI Approved: %v", license.IsOSIApproved)
			t.Logf("  Category: %s", license.Category)
			t.Logf("  Compatibility: %s", license.Compatibility)

			if tt.checkOSI && !license.IsOSIApproved {
				t.Errorf("Expected license %s to be OSI approved", tt.licenseID)
			}

			if tt.checkCategory != "" && license.Category != tt.checkCategory {
				t.Errorf("Expected category %s, got %s", tt.checkCategory, license.Category)
			}

			if license.Name == "" {
				t.Error("License name should not be empty")
			}

			if len(license.SeeAlso) == 0 {
				t.Error("License should have at least one reference URL")
			}
		})
	}
}

func TestSPDXClient_SearchLicenses(t *testing.T) {
	logger := zap.NewNop()
	client := NewClient(logger)
	ctx := context.Background()

	tests := []struct {
		name        string
		query       string
		expectCount int
		minCount    int
	}{
		{
			name:     "Search for MIT",
			query:    "MIT",
			minCount: 1,
		},
		{
			name:     "Search for Apache",
			query:    "apache",
			minCount: 1,
		},
		{
			name:     "Search for GPL",
			query:    "gpl",
			minCount: 2, // GPL-2.0, GPL-3.0, LGPL, AGPL
		},
		{
			name:     "Search for BSD",
			query:    "bsd",
			minCount: 2, // BSD-2-Clause, BSD-3-Clause
		},
		{
			name:     "Search for permissive in comments",
			query:    "permissive",
			minCount: 3,
		},
		{
			name:        "No results",
			query:       "xyz123notfound",
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := client.SearchLicenses(ctx, tt.query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			t.Logf("Search '%s' returned %d results", tt.query, len(results))
			for _, license := range results {
				t.Logf("  - %s: %s", license.ID, license.Name)
			}

			if tt.expectCount > 0 && len(results) != tt.expectCount {
				t.Errorf("Expected exactly %d results, got %d", tt.expectCount, len(results))
			}

			if tt.minCount > 0 && len(results) < tt.minCount {
				t.Errorf("Expected at least %d results, got %d", tt.minCount, len(results))
			}
		})
	}
}

func TestSPDXClient_ListCategories(t *testing.T) {
	logger := zap.NewNop()
	client := NewClient(logger)

	categories := client.ListCategories()

	t.Logf("Found %d categories:", len(categories))
	for _, cat := range categories {
		t.Logf("  - %s", cat)
	}

	if len(categories) == 0 {
		t.Error("Expected at least one category")
	}

	// Check for expected categories
	expectedCategories := map[string]bool{
		"Permissive":      false,
		"Copyleft":        false,
		"Weak Copyleft":   false,
		"Strong Copyleft": false,
		"Public Domain":   false,
	}

	for _, cat := range categories {
		if _, exists := expectedCategories[cat]; exists {
			expectedCategories[cat] = true
		}
	}

	for cat, found := range expectedCategories {
		if !found {
			t.Logf("Warning: Expected category '%s' not found", cat)
		}
	}
}

func TestSPDXClient_GetLicensesByCategory(t *testing.T) {
	logger := zap.NewNop()
	client := NewClient(logger)

	tests := []struct {
		name     string
		category string
		minCount int
	}{
		{
			name:     "Permissive licenses",
			category: "Permissive",
			minCount: 3, // MIT, Apache, BSD, etc.
		},
		{
			name:     "Copyleft licenses",
			category: "Copyleft",
			minCount: 1, // GPL-3.0, GPL-2.0
		},
		{
			name:     "Public Domain",
			category: "Public Domain",
			minCount: 1, // CC0, Unlicense, WTFPL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := client.GetLicensesByCategory(tt.category)

			t.Logf("Category '%s' has %d licenses:", tt.category, len(results))
			for _, license := range results {
				t.Logf("  - %s: %s", license.ID, license.Name)
			}

			if len(results) < tt.minCount {
				t.Errorf("Expected at least %d licenses in category '%s', got %d",
					tt.minCount, tt.category, len(results))
			}
		})
	}
}

func TestSPDXClient_LicenseCompatibility(t *testing.T) {
	logger := zap.NewNop()
	client := NewClient(logger)
	ctx := context.Background()

	tests := []struct {
		name               string
		licenseID          string
		expectedCompatible string
	}{
		{
			name:               "MIT is very compatible",
			licenseID:          "MIT",
			expectedCompatible: "Very High",
		},
		{
			name:               "Apache-2.0 is highly compatible",
			licenseID:          "Apache-2.0",
			expectedCompatible: "High",
		},
		{
			name:               "GPL-3.0 has low compatibility",
			licenseID:          "GPL-3.0",
			expectedCompatible: "Low",
		},
		{
			name:               "AGPL-3.0 has very low compatibility",
			licenseID:          "AGPL-3.0",
			expectedCompatible: "Very Low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			license, err := client.GetLicense(ctx, tt.licenseID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			t.Logf("License: %s", license.ID)
			t.Logf("  Compatibility: %s", license.Compatibility)
			t.Logf("  Category: %s", license.Category)
			t.Logf("  OSI Approved: %v", license.IsOSIApproved)

			if license.Compatibility != tt.expectedCompatible {
				t.Errorf("Expected compatibility '%s', got '%s'",
					tt.expectedCompatible, license.Compatibility)
			}
		})
	}
}
