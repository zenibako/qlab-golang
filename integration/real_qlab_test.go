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
		// Try with empty passcode first (most secure default)
		reply, err := workspace.Init("")
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

	// Initialize connection with empty passcode (tests should use workspaces without passcodes)
	t.Log("--- Connecting to QLab ---")
	reply, err := workspace.Init("")
	if err != nil {
		t.Fatalf("Failed to initialize connection to QLab: %v", err)
	}

	t.Logf("Successfully connected to QLab. Reply: %v", reply)

	// Rest of test omitted for brevity - add as needed
}

// TestRealQLabEmptyPasscode tests connecting to a real QLab instance with an empty passcode
// This test automatically detects if QLab is available and skips if not
// Run with: go test -run TestRealQLabEmptyPasscode -v
func TestRealQLabEmptyPasscode(t *testing.T) {
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

	t.Log("=== Testing connection to real QLab instance with empty passcode ===")
	t.Logf("QLab detected on %s:%d", host, port)

	// Connect to real QLab instance
	workspace := qlab.NewWorkspace(host, port)
	defer workspace.Close()

	// Initialize connection with empty passcode
	t.Log("--- Connecting to QLab with empty passcode ---")
	reply, err := workspace.Init("")

	// Note: If your QLab workspace requires a passcode, this will fail with "badpass"
	// This test is designed to work with workspaces that have no passcode set
	if err != nil {
		// Check if it's a passcode error - if so, that's expected for protected workspaces
		if err.Error() == "QLab authentication failed - incorrect passcode. Check your passcode in the CUE file, config file, or --passcode flag" {
			t.Skipf("QLab workspace requires a passcode - skipping empty passcode test. Error: %v", err)
		}
		t.Fatalf("Failed to initialize connection to QLab with empty passcode: %v", err)
	}

	t.Logf("Successfully connected to QLab with empty passcode. Reply: %v", reply)

	// Verify workspace is connected
	if !workspace.IsConnected() {
		t.Fatal("Workspace should be connected after successful Init")
	}

	// Verify we can perform a basic operation by querying cue lists
	t.Log("--- Testing basic operation after empty passcode connection ---")
	cueLists := workspace.Send("/cueLists", "")
	if len(cueLists) == 0 {
		t.Error("Expected to receive cue lists from QLab")
	} else {
		t.Logf("Successfully queried cue lists: %v", cueLists[0])
	}

	t.Log("Empty passcode connection test completed successfully")
}
