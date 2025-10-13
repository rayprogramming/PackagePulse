package spdx

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// Client provides access to SPDX license information
type Client struct {
	logger   *zap.Logger
	licenses map[string]*LicenseInfo
}

// LicenseInfo represents structured license data
type LicenseInfo struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Text          string   `json:"text,omitempty"`
	IsOSIApproved bool     `json:"is_osi_approved"`
	IsFSFLibre    bool     `json:"is_fsf_libre"`
	IsDeprecated  bool     `json:"is_deprecated"`
	SeeAlso       []string `json:"see_also,omitempty"`
	Comments      string   `json:"comments,omitempty"`
	Category      string   `json:"category"`
	Compatibility string   `json:"compatibility"`
}

// NewClient creates a new SPDX license client
func NewClient(logger *zap.Logger) *Client {
	client := &Client{
		logger:   logger,
		licenses: make(map[string]*LicenseInfo),
	}

	// Initialize with common license data
	client.initializeLicenses()

	return client
}

// GetLicense retrieves information about a specific license by SPDX ID
func (c *Client) GetLicense(ctx context.Context, licenseID string) (*LicenseInfo, error) {
	c.logger.Debug("Looking up license", zap.String("id", licenseID))

	// Normalize the license ID (case-insensitive lookup)
	normalizedID := strings.ToUpper(strings.TrimSpace(licenseID))

	// Check exact match first
	if license, ok := c.licenses[licenseID]; ok {
		return license, nil
	}

	// Try normalized lookup
	for id, license := range c.licenses {
		if strings.ToUpper(id) == normalizedID {
			return license, nil
		}
	}

	return nil, fmt.Errorf("license not found: %s", licenseID)
}

// SearchLicenses searches for licenses matching the query
func (c *Client) SearchLicenses(ctx context.Context, query string) ([]*LicenseInfo, error) {
	c.logger.Debug("Searching licenses", zap.String("query", query))

	query = strings.ToLower(strings.TrimSpace(query))
	var results []*LicenseInfo

	for _, license := range c.licenses {
		// Search in ID, name, and comments
		if strings.Contains(strings.ToLower(license.ID), query) ||
			strings.Contains(strings.ToLower(license.Name), query) ||
			strings.Contains(strings.ToLower(license.Comments), query) {
			results = append(results, license)
		}
	}

	c.logger.Debug("Search complete", zap.Int("results", len(results)))
	return results, nil
}

// ListCategories returns all available license categories
func (c *Client) ListCategories() []string {
	categories := make(map[string]bool)
	for _, license := range c.licenses {
		if license.Category != "" {
			categories[license.Category] = true
		}
	}

	result := make([]string, 0, len(categories))
	for cat := range categories {
		result = append(result, cat)
	}
	return result
}

// GetLicensesByCategory returns all licenses in a specific category
func (c *Client) GetLicensesByCategory(category string) []*LicenseInfo {
	var results []*LicenseInfo
	for _, license := range c.licenses {
		if license.Category == category {
			results = append(results, license)
		}
	}
	return results
}

// initializeLicenses populates the license database with common SPDX licenses
func (c *Client) initializeLicenses() {
	// Popular permissive licenses
	c.addLicense(&LicenseInfo{
		ID:            "MIT",
		Name:          "MIT License",
		IsOSIApproved: true,
		IsFSFLibre:    true,
		Category:      "Permissive",
		Compatibility: "Very High",
		Comments:      "Simple and permissive license allowing almost unrestricted freedom",
		SeeAlso:       []string{"https://opensource.org/licenses/MIT"},
	})

	c.addLicense(&LicenseInfo{
		ID:            "Apache-2.0",
		Name:          "Apache License 2.0",
		IsOSIApproved: true,
		IsFSFLibre:    true,
		Category:      "Permissive",
		Compatibility: "High",
		Comments:      "Permissive license with patent grant and trademark protection",
		SeeAlso:       []string{"https://www.apache.org/licenses/LICENSE-2.0"},
	})

	c.addLicense(&LicenseInfo{
		ID:            "BSD-3-Clause",
		Name:          "BSD 3-Clause \"New\" or \"Revised\" License",
		IsOSIApproved: true,
		IsFSFLibre:    true,
		Category:      "Permissive",
		Compatibility: "Very High",
		Comments:      "Permissive license similar to MIT but with explicit non-endorsement clause",
		SeeAlso:       []string{"https://opensource.org/licenses/BSD-3-Clause"},
	})

	c.addLicense(&LicenseInfo{
		ID:            "BSD-2-Clause",
		Name:          "BSD 2-Clause \"Simplified\" License",
		IsOSIApproved: true,
		IsFSFLibre:    true,
		Category:      "Permissive",
		Compatibility: "Very High",
		Comments:      "Simplified version of BSD license with fewer restrictions",
		SeeAlso:       []string{"https://opensource.org/licenses/BSD-2-Clause"},
	})

	c.addLicense(&LicenseInfo{
		ID:            "ISC",
		Name:          "ISC License",
		IsOSIApproved: true,
		IsFSFLibre:    true,
		Category:      "Permissive",
		Compatibility: "Very High",
		Comments:      "Functionally equivalent to MIT and BSD 2-Clause",
		SeeAlso:       []string{"https://opensource.org/licenses/ISC"},
	})

	// Copyleft licenses
	c.addLicense(&LicenseInfo{
		ID:            "GPL-3.0",
		Name:          "GNU General Public License v3.0",
		IsOSIApproved: true,
		IsFSFLibre:    true,
		Category:      "Copyleft",
		Compatibility: "Low",
		Comments:      "Strong copyleft license requiring source code disclosure",
		SeeAlso:       []string{"https://www.gnu.org/licenses/gpl-3.0.html"},
	})

	c.addLicense(&LicenseInfo{
		ID:            "GPL-2.0",
		Name:          "GNU General Public License v2.0",
		IsOSIApproved: true,
		IsFSFLibre:    true,
		Category:      "Copyleft",
		Compatibility: "Low",
		Comments:      "Earlier version of GPL with strong copyleft requirements",
		SeeAlso:       []string{"https://www.gnu.org/licenses/old-licenses/gpl-2.0.html"},
	})

	c.addLicense(&LicenseInfo{
		ID:            "LGPL-3.0",
		Name:          "GNU Lesser General Public License v3.0",
		IsOSIApproved: true,
		IsFSFLibre:    true,
		Category:      "Weak Copyleft",
		Compatibility: "Medium",
		Comments:      "Weaker copyleft allowing dynamic linking without license propagation",
		SeeAlso:       []string{"https://www.gnu.org/licenses/lgpl-3.0.html"},
	})

	c.addLicense(&LicenseInfo{
		ID:            "AGPL-3.0",
		Name:          "GNU Affero General Public License v3.0",
		IsOSIApproved: true,
		IsFSFLibre:    true,
		Category:      "Strong Copyleft",
		Compatibility: "Very Low",
		Comments:      "Strongest copyleft license including network use trigger",
		SeeAlso:       []string{"https://www.gnu.org/licenses/agpl-3.0.html"},
	})

	c.addLicense(&LicenseInfo{
		ID:            "MPL-2.0",
		Name:          "Mozilla Public License 2.0",
		IsOSIApproved: true,
		IsFSFLibre:    true,
		Category:      "Weak Copyleft",
		Compatibility: "Medium",
		Comments:      "File-level copyleft license balancing openness and commercial use",
		SeeAlso:       []string{"https://www.mozilla.org/MPL/2.0/"},
	})

	// Creative Commons
	c.addLicense(&LicenseInfo{
		ID:            "CC0-1.0",
		Name:          "Creative Commons Zero v1.0 Universal",
		IsOSIApproved: false,
		IsFSFLibre:    true,
		Category:      "Public Domain",
		Compatibility: "Very High",
		Comments:      "Public domain dedication for maximum freedom",
		SeeAlso:       []string{"https://creativecommons.org/publicdomain/zero/1.0/"},
	})

	c.addLicense(&LicenseInfo{
		ID:            "CC-BY-4.0",
		Name:          "Creative Commons Attribution 4.0 International",
		IsOSIApproved: false,
		IsFSFLibre:    true,
		Category:      "Permissive",
		Compatibility: "High",
		Comments:      "Requires attribution but allows commercial use and derivatives",
		SeeAlso:       []string{"https://creativecommons.org/licenses/by/4.0/"},
	})

	// Proprietary/Restrictive
	c.addLicense(&LicenseInfo{
		ID:            "Unlicense",
		Name:          "The Unlicense",
		IsOSIApproved: false,
		IsFSFLibre:    true,
		Category:      "Public Domain",
		Compatibility: "Very High",
		Comments:      "Release software into public domain",
		SeeAlso:       []string{"http://unlicense.org/"},
	})

	c.addLicense(&LicenseInfo{
		ID:            "WTFPL",
		Name:          "Do What The F*ck You Want To Public License",
		IsOSIApproved: false,
		IsFSFLibre:    true,
		Category:      "Public Domain",
		Compatibility: "Very High",
		Comments:      "Extremely permissive public domain-like license",
		SeeAlso:       []string{"http://www.wtfpl.net/"},
	})

	c.logger.Info("Initialized license database", zap.Int("count", len(c.licenses)))
}

// addLicense adds a license to the internal database
func (c *Client) addLicense(license *LicenseInfo) {
	c.licenses[license.ID] = license
}
