package resources

import (
	"github.com/rayprogramming/hypermcp"
	"go.uber.org/zap"
)

// ResourceRegistry manages all MCP resources
type ResourceRegistry struct {
	logger *zap.Logger
}

// NewResourceRegistry creates a new resource registry
func NewResourceRegistry(logger *zap.Logger) (*ResourceRegistry, error) {
	return &ResourceRegistry{
		logger: logger,
	}, nil
}

// Register registers all resources with the server
func (rr *ResourceRegistry) Register(srv *hypermcp.Server) error {
	// Resources will be added in future iterations
	return nil
}
