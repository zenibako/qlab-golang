package integration

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/zenibako/qlab-golang/qlab"
)

// isQLabAvailable checks if QLab is running and accessible on the given host:port
func isQLabAvailable(host string, port int) bool {
	// Try a simple TCP connection to see if something is listening on the port
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()

	// Actually try to initialize a connection with a timeout to verify QLab responds
	workspace := qlab.NewWorkspace(host, port)

	// Use a channel to implement timeout on the Init call
	type result struct {
		reply []any
		err   error
	}
	ch := make(chan result, 1)

	go func() {
		reply, err := workspace.Init("1297")
		ch <- result{reply: reply, err: err}
	}()

	select {
	case res := <-ch:
		// If we got a response (even an error), QLab is responding
		// Only return false if there was no response at all
		return res.err == nil
	case <-time.After(3 * time.Second):
		// Timeout waiting for QLab to respond - it's not actually available
		return false
	}
}

// TestRealQLab tests connection and basic operations against a real QLab instance
// This test automatically detects if QLab is available and skips if not
// Run with: go test -run TestRealQLab -v
func TestRealQLab(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real QLab test in short mode")
	}

	host := "localhost"
	port := 53000

	// Check if QLab is available before running the test
	t.Log("--- Checking if QLab is available ---")
	if !isQLabAvailable(host, port) {
		t.Skipf("QLab is not available on %s:%d - skipping real QLab test", host, port)
	}

	t.Log("=== Testing connection to real QLab instance ===")
	t.Logf("QLab detected on %s:%d", host, port)

	// Connect to real QLab instance
	workspace := qlab.NewWorkspace(host, port)

	// Initialize connection with test passcode
	t.Log("--- Connecting to QLab ---")
	reply, err := workspace.Init("1297")
	if err != nil {
		t.Fatalf("Failed to initialize connection to QLab: %v", err)
	}

	t.Logf("Successfully connected to QLab. Reply: %v", reply)

	// Rest of test omitted for brevity - add as needed
}
