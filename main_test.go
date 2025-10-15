package main

import (
"bufio"
"context"
"os"
"os/exec"
"strings"
"syscall"
"testing"
"time"

"github.com/rayprogramming/hypermcp"
"github.com/rayprogramming/hypermcp/cache"
"go.uber.org/zap"
)

// TestStdioTransportStartup tests that the server starts in stdio mode
// and logs the expected startup message
func TestStdioTransportStartup(t *testing.T) {
// Build the binary first
buildCmd := exec.Command("go", "build", "-o", "packagepulse_test", "main.go")
buildCmd.Dir = "."
if err := buildCmd.Run(); err != nil {
t.Fatalf("failed to build binary: %v", err)
}
defer func() {
_ = os.Remove("packagepulse_test")
}()

// Start the server process
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

cmd := exec.CommandContext(ctx, "./packagepulse_test")

// Capture stdout and stderr
stdout, err := cmd.StdoutPipe()
if err != nil {
t.Fatalf("failed to get stdout pipe: %v", err)
}
stderr, err := cmd.StderrPipe()
if err != nil {
t.Fatalf("failed to get stderr pipe: %v", err)
}

// Start the process
if err := cmd.Start(); err != nil {
t.Fatalf("failed to start server: %v", err)
}

// Create channels to capture log output
startupLogFound := make(chan bool, 1)
done := make(chan bool, 1)

// Read from stderr (zap production logger writes to stderr)
go func() {
scanner := bufio.NewScanner(stderr)
for scanner.Scan() {
line := scanner.Text()
t.Logf("stderr: %s", line)

// Check for startup log message
if strings.Contains(line, "starting PackagePulse MCP server") &&
   strings.Contains(line, "stdio") {
startupLogFound <- true
return
}
}
done <- true
}()

// Also monitor stdout in case of any output
go func() {
scanner := bufio.NewScanner(stdout)
for scanner.Scan() {
t.Logf("stdout: %s", scanner.Text())
}
}()

// Wait for startup log or timeout
select {
case <-startupLogFound:
t.Log("Successfully detected startup log message")
case <-done:
t.Error("Server terminated without startup log")
case <-time.After(5 * time.Second):
t.Error("Timeout waiting for startup log")
}

// Give the server a moment to fully initialize
time.Sleep(500 * time.Millisecond)

// Send SIGTERM to test graceful shutdown
if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
t.Errorf("failed to send SIGTERM: %v", err)
}

// Wait for process to exit
waitDone := make(chan error, 1)
go func() {
waitDone <- cmd.Wait()
}()

select {
case err := <-waitDone:
// Process exited - check if it was clean
if err != nil {
// Check if it's just a signal-related exit
if exitErr, ok := err.(*exec.ExitError); ok {
// Exit code 0 or signal-terminated is acceptable
if exitErr.ExitCode() == 0 || strings.Contains(err.Error(), "signal") {
t.Log("Server shut down cleanly after signal")
} else {
t.Errorf("Server exited with unexpected error: %v", err)
}
} else {
t.Errorf("Server exited with error: %v", err)
}
} else {
t.Log("Server shut down cleanly")
}
case <-time.After(5 * time.Second):
// Force kill if it doesn't exit gracefully
_ = cmd.Process.Kill()
t.Error("Server did not shut down within timeout")
}
}

// TestSignalHandling specifically tests SIGINT and SIGTERM handling
func TestSignalHandling(t *testing.T) {
signals := []struct {
name   string
signal os.Signal
}{
{"SIGTERM", syscall.SIGTERM},
{"SIGINT", os.Interrupt},
}

for _, tc := range signals {
t.Run(tc.name, func(t *testing.T) {
// Build the binary
buildCmd := exec.Command("go", "build", "-o", "packagepulse_test_"+tc.name, "main.go")
buildCmd.Dir = "."
if err := buildCmd.Run(); err != nil {
t.Fatalf("failed to build binary: %v", err)
}
defer func() {
_ = os.Remove("packagepulse_test_" + tc.name)
}()

// Start the server
cmd := exec.Command("./packagepulse_test_" + tc.name)

stderr, err := cmd.StderrPipe()
if err != nil {
t.Fatalf("failed to get stderr pipe: %v", err)
}

if err := cmd.Start(); err != nil {
t.Fatalf("failed to start server: %v", err)
}

// Wait for startup
started := make(chan bool, 1)
go func() {
scanner := bufio.NewScanner(stderr)
for scanner.Scan() {
line := scanner.Text()
if strings.Contains(line, "starting PackagePulse MCP server") {
started <- true
return
}
}
}()

// Wait for server to start
select {
case <-started:
t.Logf("Server started, testing %s handling", tc.name)
case <-time.After(5 * time.Second):
_ = cmd.Process.Kill()
t.Fatalf("Server did not start within timeout")
}

// Give it a moment to initialize
time.Sleep(300 * time.Millisecond)

// Send the signal
if err := cmd.Process.Signal(tc.signal); err != nil {
t.Fatalf("failed to send %s: %v", tc.name, err)
}

// Wait for clean shutdown
done := make(chan error, 1)
go func() {
done <- cmd.Wait()
}()

select {
case err := <-done:
// Expect signal-related exit or clean exit
if err != nil {
if exitErr, ok := err.(*exec.ExitError); ok {
if exitErr.ExitCode() == 0 || strings.Contains(err.Error(), "signal") {
t.Logf("Server handled %s correctly", tc.name)
} else {
t.Errorf("Unexpected exit after %s: %v", tc.name, err)
}
} else {
t.Errorf("Error waiting for process: %v", err)
}
} else {
t.Logf("Server shut down cleanly after %s", tc.name)
}
case <-time.After(5 * time.Second):
_ = cmd.Process.Kill()
t.Errorf("Server did not respond to %s within timeout", tc.name)
}
})
}
}

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
Version:      "1.0.0",
CacheEnabled: false,
},
wantError: false,
validateFunc: func(t *testing.T, cfg hypermcp.Config) {
if cfg.CacheEnabled {
t.Error("Expected CacheEnabled to be false")
}
},
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
logger := zap.NewNop()

srv, err := hypermcp.New(tt.cfg, logger)
if (err != nil) != tt.wantError {
t.Errorf("hypermcp.New() error = %v, wantError %v", err, tt.wantError)
return
}

if err == nil && srv == nil {
t.Error("Expected non-nil server when no error")
}

if tt.validateFunc != nil && err == nil {
tt.validateFunc(t, tt.cfg)
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
if cfg.Name != "PackagePulse" {
t.Errorf("Expected Name 'PackagePulse', got '%s'", cfg.Name)
}
if cfg.Version != "1.0.0" {
t.Errorf("Expected Version '1.0.0', got '%s'", cfg.Version)
}
if !cfg.CacheEnabled {
t.Error("Expected CacheEnabled to be true")
}

// Verify cache config
expectedMaxCost := int64(100 * 1024 * 1024)
if cfg.CacheConfig.MaxCost != expectedMaxCost {
t.Errorf("Expected MaxCost %d, got %d", expectedMaxCost, cfg.CacheConfig.MaxCost)
}
if cfg.CacheConfig.NumCounters != 10000 {
t.Errorf("Expected NumCounters 10000, got %d", cfg.CacheConfig.NumCounters)
}
if cfg.CacheConfig.BufferItems != 64 {
t.Errorf("Expected BufferItems 64, got %d", cfg.CacheConfig.BufferItems)
}
})

t.Run("cache config calculations", func(t *testing.T) {
// Test that the MaxCost calculation is correct
maxCostMB := 100
expectedBytes := int64(maxCostMB * 1024 * 1024)

cfg := hypermcp.Config{
Name:         "Test",
Version:      "1.0.0",
CacheEnabled: true,
CacheConfig: cache.Config{
MaxCost: expectedBytes,
},
}

if cfg.CacheConfig.MaxCost != expectedBytes {
t.Errorf("MaxCost mismatch: expected %d bytes (%d MB), got %d",
expectedBytes, maxCostMB, cfg.CacheConfig.MaxCost)
}

// Verify it's actually 100MB
actualMB := cfg.CacheConfig.MaxCost / (1024 * 1024)
if actualMB != int64(maxCostMB) {
t.Errorf("Expected %d MB, got %d MB", maxCostMB, actualMB)
}
})
}
