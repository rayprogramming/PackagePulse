package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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

	// Register tools and resources
	if err := registerFeatures(srv, logger); err != nil {
		logger.Fatal("failed to register features", zap.Error(err))
	}

	// Log registration stats
	srv.LogRegistrationStats()

	// Setup context with signal handling for clean shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT (Ctrl+C) and SIGTERM for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Goroutine to handle shutdown signal
	go func() {
		sig := <-sigChan
		logger.Info("received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	}()

	// Run with stdio transport
	logger.Info("starting PackagePulse MCP server", zap.String("transport", "stdio"))
	if err := hypermcp.RunWithTransport(ctx, srv, hypermcp.TransportStdio, logger); err != nil {
		// Context cancellation is expected during graceful shutdown
		if ctx.Err() == context.Canceled {
			logger.Info("server shutdown complete")
			return
		}
		logger.Fatal("server failed", zap.Error(err))
	}

	logger.Info("server shutdown complete")
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
