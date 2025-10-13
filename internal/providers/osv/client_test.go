package osv

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestOSVClientQuery(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewClient(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name      string
		ecosystem string
		pkg       string
		version   string
		wantError bool
	}{
		{
			name:      "npm lodash with known vulnerabilities",
			ecosystem: "npm",
			pkg:       "lodash",
			version:   "4.17.19",
			wantError: false,
		},
		{
			name:      "go package",
			ecosystem: "Go",
			pkg:       "github.com/gin-gonic/gin",
			version:   "1.7.0",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Query(ctx, tt.ecosystem, tt.pkg, tt.version)
			if (err != nil) != tt.wantError {
				t.Errorf("Query() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if err == nil {
				t.Logf("Found %d vulnerabilities", len(result.Vulns))
				for i, vuln := range result.Vulns {
					if i < 3 { // Log first 3
						t.Logf("  - %s: %s", vuln.ID, vuln.Summary)
					}
				}
			}
		})
	}
}

func TestOSVClientBatchQuery(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewClient(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	queries := []QueryRequest{
		{
			Package: Package{Name: "lodash", Ecosystem: "npm"},
			Version: "4.17.19",
		},
		{
			Package: Package{Name: "express", Ecosystem: "npm"},
			Version: "4.17.0",
		},
	}

	results, err := client.BatchQuery(ctx, queries)
	if err != nil {
		t.Fatalf("BatchQuery() error = %v", err)
	}

	if len(results) != len(queries) {
		t.Errorf("Expected %d results, got %d", len(queries), len(results))
	}

	for i, result := range results {
		t.Logf("Query %d: Found %d vulnerabilities", i, len(result.Vulns))
	}
}
