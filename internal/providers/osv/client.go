package osv

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const (
	APIBaseURL = "https://api.osv.dev/v1"
	QueryPath  = "/query"
	BatchPath  = "/querybatch"
	Timeout    = 30 * time.Second
)

// Client handles OSV API interactions
type Client struct {
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a new OSV API client
func NewClient(logger *zap.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: Timeout,
		},
		logger: logger,
	}
}

// QueryRequest represents an OSV vulnerability query
type QueryRequest struct {
	Package Package `json:"package"`
	Version string  `json:"version,omitempty"`
}

// Package identifies the package ecosystem and name
type Package struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

// QueryResponse contains vulnerability results
type QueryResponse struct {
	Vulns []Vulnerability `json:"vulns"`
}

// Vulnerability represents a single vulnerability entry
type Vulnerability struct {
	ID         string      `json:"id"`
	Summary    string      `json:"summary"`
	Details    string      `json:"details"`
	Published  time.Time   `json:"published"`
	Modified   time.Time   `json:"modified"`
	Severity   []Severity  `json:"severity,omitempty"`
	Affected   []Affected  `json:"affected,omitempty"`
	References []Reference `json:"references,omitempty"`
	Aliases    []string    `json:"aliases,omitempty"`
}

// Severity contains severity scoring information
type Severity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

// Affected describes affected package versions
type Affected struct {
	Package           Package                `json:"package"`
	Ranges            []VersionRange         `json:"ranges,omitempty"`
	DatabaseSpecific  map[string]interface{} `json:"database_specific,omitempty"`
	EcosystemSpecific map[string]interface{} `json:"ecosystem_specific,omitempty"`
}

// VersionRange specifies the range of affected versions
type VersionRange struct {
	Type   string  `json:"type"`
	Events []Event `json:"events"`
}

// Event represents a version event (introduced/fixed)
type Event struct {
	Introduced string `json:"introduced,omitempty"`
	Fixed      string `json:"fixed,omitempty"`
}

// Reference contains external reference links
type Reference struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// Query queries OSV for vulnerabilities in a specific package version
// Example: client.Query(ctx, "npm", "lodash", "4.17.19")
func (c *Client) Query(ctx context.Context, ecosystem, name, version string) (*QueryResponse, error) {
	req := QueryRequest{
		Package: Package{
			Name:      name,
			Ecosystem: ecosystem,
		},
		Version: version,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, APIBaseURL+QueryPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	c.logger.Debug("querying OSV",
		zap.String("ecosystem", ecosystem),
		zap.String("package", name),
		zap.String("version", version))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OSV API error: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var result QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	c.logger.Debug("OSV query complete",
		zap.Int("vulns_found", len(result.Vulns)))

	return &result, nil
}

// BatchQuery queries multiple packages in a single request
func (c *Client) BatchQuery(ctx context.Context, queries []QueryRequest) ([]QueryResponse, error) {
	if len(queries) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(map[string]interface{}{
		"queries": queries,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal batch request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, APIBaseURL+BatchPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create batch request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	c.logger.Debug("batch querying OSV", zap.Int("query_count", len(queries)))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute batch request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OSV batch API error: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Results []QueryResponse `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode batch response: %w", err)
	}

	c.logger.Debug("OSV batch query complete", zap.Int("results", len(result.Results)))

	return result.Results, nil
}
