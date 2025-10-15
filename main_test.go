package main

import (
	"testing"

	"github.com/rayprogramming/hypermcp"
	"github.com/rayprogramming/hypermcp/cache"
	"go.uber.org/zap"
)

// TestRegisterFeatures verifies that registerFeatures properly registers tools and resources
func TestRegisterFeatures(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Create a test server
	cfg := hypermcp.Config{
		Name:         "PackagePulse-Test",
		Version:      "1.0.0",
		CacheEnabled: true,
		CacheConfig: cache.Config{
			MaxCost:     100 * 1024 * 1024, // 100MB
			NumCounters: 10_000,
			BufferItems: 64,
		},
	}

	srv, err := hypermcp.New(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Call registerFeatures
	if err := registerFeatures(srv, logger); err != nil {
		t.Fatalf("registerFeatures failed: %v", err)
	}

	// Verify that tools were registered
	// The current implementation should register 4 tools:
	// - deps.vulns
	// - deps.health
	// - license.info
	// - deps.upgrade_plan
	// We can't directly access the count, but we can verify no error occurred
	t.Log("registerFeatures completed successfully")
}

// TestServerInitialization verifies the complete server initialization flow
func TestServerInitialization(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Configure server
	cfg := hypermcp.Config{
		Name:         "PackagePulse-Test",
		Version:      "1.0.0",
		CacheEnabled: true,
		CacheConfig: cache.Config{
			MaxCost:     100 * 1024 * 1024,
			NumCounters: 10_000,
			BufferItems: 64,
		},
	}

	// Create base server
	srv, err := hypermcp.New(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Register features
	if err := registerFeatures(srv, logger); err != nil {
		t.Fatalf("failed to register features: %v", err)
	}

	// Call LogRegistrationStats (this should not error)
	srv.LogRegistrationStats()

	t.Log("Server initialization and feature registration completed successfully")
}
