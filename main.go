package main

import (
	"context"

	"github.com/rayprogramming/PackagePulse/internal/resources"
	"github.com/rayprogramming/PackagePulse/internal/tools"
	"github.com/rayprogramming/hypermcp"
	"github.com/rayprogramming/hypermcp/cache"
	"go.uber.org/zap"
)

func main() {
	// Setup logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Configure server with optimized cache settings
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

	// Create base server
	srv, err := hypermcp.New(cfg, logger)
	if err != nil {
		logger.Fatal("failed to create server", zap.Error(err))
	}

	// Log startup message
	logger.Info("PackagePulse server initialized",
		zap.String("name", cfg.Name),
		zap.String("version", cfg.Version),
		zap.Bool("cache_enabled", cfg.CacheEnabled))

	// Register tools and resources
	if err := registerFeatures(srv, logger); err != nil {
		logger.Fatal("failed to register features", zap.Error(err))
	}

	// Log registration stats
	srv.LogRegistrationStats()

	// Run with stdio transport
	ctx := context.Background()
	if err := hypermcp.RunWithTransport(ctx, srv, hypermcp.TransportStdio, logger); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}

func registerFeatures(srv *hypermcp.Server, logger *zap.Logger) error {
	// Initialize tool registry
	toolRegistry, err := tools.NewToolRegistry(logger, srv.Cache())
	if err != nil {
		return err
	}

	// Register all tools
	if err := toolRegistry.Register(srv); err != nil {
		return err
	}

	// Initialize resource registry
	resourceRegistry, err := resources.NewResourceRegistry(logger)
	if err != nil {
		return err
	}

	// Register all resources
	if err := resourceRegistry.Register(srv); err != nil {
		return err
	}

	return nil
}
