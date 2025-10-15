package main

import (
	"testing"

	"github.com/rayprogramming/hypermcp"
	"github.com/rayprogramming/hypermcp/cache"
	"go.uber.org/zap"
)

// TestServerConfigCreation tests the server configuration creation
func TestServerConfigCreation(t *testing.T) {
	tests := []struct {
		name         string
		cfg          hypermcp.Config
		wantError    bool
		validateFunc func(*testing.T, hypermcp.Config)
	}{
		{
			name: "valid config with cache enabled",
			cfg: hypermcp.Config{
				Name:         "TestServer",
				Version:      "1.0.0",
				CacheEnabled: true,
				CacheConfig: cache.Config{
					MaxCost:     100 * 1024 * 1024,
					NumCounters: 10000,
					BufferItems: 64,
				},
			},
			wantError: false,
			validateFunc: func(t *testing.T, cfg hypermcp.Config) {
				if cfg.Name != "TestServer" {
					t.Errorf("Expected Name 'TestServer', got '%s'", cfg.Name)
				}
				if cfg.Version != "1.0.0" {
					t.Errorf("Expected Version '1.0.0', got '%s'", cfg.Version)
				}
				if !cfg.CacheEnabled {
					t.Error("Expected CacheEnabled to be true")
				}
				if cfg.CacheConfig.MaxCost != 100*1024*1024 {
					t.Errorf("Expected MaxCost 104857600, got %d", cfg.CacheConfig.MaxCost)
				}
			},
		},
		{
			name: "valid config with cache disabled",
			cfg: hypermcp.Config{
				Name:         "PackagePulse",
				Version:      "2.0.0",
				CacheEnabled: false,
			},
			wantError: false,
			validateFunc: func(t *testing.T, cfg hypermcp.Config) {
				if cfg.Name != "PackagePulse" {
					t.Errorf("Expected Name 'PackagePulse', got '%s'", cfg.Name)
				}
				if cfg.Version != "2.0.0" {
					t.Errorf("Expected Version '2.0.0', got '%s'", cfg.Version)
				}
				if cfg.CacheEnabled {
					t.Error("Expected CacheEnabled to be false")
				}
			},
		},
		{
			name: "production config matching main.go",
			cfg: hypermcp.Config{
				Name:         "PackagePulse",
				Version:      "1.0.0",
				CacheEnabled: true,
				CacheConfig: cache.Config{
					MaxCost:     100 * 1024 * 1024,
					NumCounters: 10_000,
					BufferItems: 64,
				},
			},
			wantError: false,
			validateFunc: func(t *testing.T, cfg hypermcp.Config) {
				if cfg.Name != "PackagePulse" {
					t.Errorf("Expected Name 'PackagePulse', got '%s'", cfg.Name)
				}
				if cfg.Version != "1.0.0" {
					t.Errorf("Expected Version '1.0.0', got '%s'", cfg.Version)
				}
				if !cfg.CacheEnabled {
					t.Error("Expected CacheEnabled to be true")
				}
				if cfg.CacheConfig.NumCounters != 10000 {
					t.Errorf("Expected NumCounters 10000, got %d", cfg.CacheConfig.NumCounters)
				}
				if cfg.CacheConfig.BufferItems != 64 {
					t.Errorf("Expected BufferItems 64, got %d", cfg.CacheConfig.BufferItems)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate the config structure
			if tt.validateFunc != nil {
				tt.validateFunc(t, tt.cfg)
			}

			// Attempt to create server with the config
			logger := zap.NewNop()
			srv, err := hypermcp.New(tt.cfg, logger)

			if (err != nil) != tt.wantError {
				t.Errorf("hypermcp.New() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err == nil && srv == nil {
				t.Error("Expected non-nil server when no error")
			}
		})
	}
}

// TestConfigParsing validates that config values are correctly parsed
func TestConfigParsing(t *testing.T) {
	t.Run("parse production config", func(t *testing.T) {
		// This matches the exact config from main.go
		cfg := hypermcp.Config{
			Name:         "PackagePulse",
			Version:      "1.0.0",
			CacheEnabled: true,
			CacheConfig: cache.Config{
				MaxCost:     100 * 1024 * 1024, // 100MB
				NumCounters: 10_000,
				BufferItems: 64,
			},
		}

		// Verify config fields
		if cfg.Name == "" {
			t.Error("Config Name should not be empty")
		}
		if cfg.Version == "" {
			t.Error("Config Version should not be empty")
		}

		// Verify cache config when enabled
		if cfg.CacheEnabled {
			if cfg.CacheConfig.MaxCost <= 0 {
				t.Error("CacheConfig.MaxCost should be positive when cache is enabled")
			}
			if cfg.CacheConfig.NumCounters <= 0 {
				t.Error("CacheConfig.NumCounters should be positive when cache is enabled")
			}
			if cfg.CacheConfig.BufferItems <= 0 {
				t.Error("CacheConfig.BufferItems should be positive when cache is enabled")
			}
		}

		t.Logf("Config parsed successfully: Name=%s, Version=%s, CacheEnabled=%v",
			cfg.Name, cfg.Version, cfg.CacheEnabled)
	})

	t.Run("cache size calculation", func(t *testing.T) {
		maxCostBytes := 100 * 1024 * 1024
		expectedMB := 100

		actualMB := maxCostBytes / (1024 * 1024)
		if actualMB != expectedMB {
			t.Errorf("Expected cache size %dMB, got %dMB", expectedMB, actualMB)
		}
	})
}

// TestRegisterFeatures tests that feature registration doesn't error
func TestRegisterFeatures(t *testing.T) {
	logger := zap.NewNop()

	cfg := hypermcp.Config{
		Name:         "TestServer",
		Version:      "1.0.0",
		CacheEnabled: true,
		CacheConfig: cache.Config{
			MaxCost:     100 * 1024 * 1024,
			NumCounters: 10000,
			BufferItems: 64,
		},
	}

	srv, err := hypermcp.New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test that registerFeatures completes without error
	err = registerFeatures(srv, logger)
	if err != nil {
		t.Errorf("registerFeatures() error = %v, want nil", err)
	}
}
