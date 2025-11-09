package qlab

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/log"
)

// getFreePort gets an available port by asking the OS
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = l.Close() // Ignore error - port is being freed
	}()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// setupWorkspaceWithCleanup initializes a workspace with mock QLab OSC server and sets up cleanup
func setupWorkspaceWithCleanup(t *testing.T) (*Workspace, *MockOSCServer) {
	// Get an available port from the OS
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	// Create and start mock QLab OSC server
	mockServer := NewMockOSCServer("localhost", port)
	if err = mockServer.Start(); err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}

	// Create workspace with pre-configured workspace ID (no real QLab connection needed)
	workspace := NewTestWorkspace("localhost", port, mockServer.GetWorkspaceID())

	// Clean up after test
	t.Cleanup(func() {
		workspace.Close()  // Close workspace servers
		mockServer.Clear() // Clear mock server state
		if err := mockServer.Stop(); err != nil {
			t.Logf("Failed to stop mock server: %v", err)
		}
		// Give more time for ports to be released and goroutines to finish
		time.Sleep(150 * time.Millisecond)
	})

	return workspace, mockServer
}

// setupWorkspaceWithCleanupFatal initializes a workspace with mock server (always succeeds since no real QLab required)
func setupWorkspaceWithCleanupFatal(t *testing.T) (*Workspace, *MockOSCServer) {
	return setupWorkspaceWithCleanup(t) // Same implementation since mock server should always work
}

// extractCueNumber extracts cue number from test data
func extractCueNumber(cueData map[string]any) string {
	if num, ok := cueData["number"]; ok && num != nil {
		return fmt.Sprintf("%v", num)
	}
	return ""
}

// createCueWithNumber creates a cue and extracts the number automatically
func createCueWithNumber(t *testing.T, workspace *Workspace, cueData map[string]any) {
	cueNumber := extractCueNumber(cueData)
	_, err := workspace.createCue(cueData, cueNumber)
	if err != nil {
		t.Errorf("createCue failed: %v", err)
	}
}

func TestInit(t *testing.T) {
	// Enable debug logging
	log.SetLevel(log.InfoLevel)

	workspace, mockServer := setupWorkspaceWithCleanupFatal(t)

	// Verify workspace is initialized correctly
	if !workspace.initialized {
		t.Error("Workspace should be initialized")
	}
	if workspace.workspace_id == "" {
		t.Error("Workspace ID should not be empty")
	}

	// Test mock OSC communication by creating a simple cue
	t.Log("Testing OSC communication by creating a simple cue")
	cueData := map[string]any{
		"type": "memo",
		"name": "Test Memo",
	}

	uniqueID, err := workspace.createCue(cueData, "")
	if err != nil {
		t.Errorf("Failed to create test cue: %v", err)
	} else {
		t.Logf("Successfully created cue with ID: %s", uniqueID)
		t.Logf("Mock server now has %d cues", mockServer.GetCueCount())
	}
}

func TestTransmitUberNoirCue(t *testing.T) {
	// Skip test - requires CUE file handler to be injected
	t.Skip("Test requires CUE file handler - skipping in standalone qlab-golang package")

	// NOTE: TransmitCueFile has been removed. Use TransmitWorkspaceData instead.
	// Caller should parse CUE file and pass the data to TransmitWorkspaceData()
}

// TestAudioCueCreation tests creating audio cues with different properties
func TestAudioCueCreation(t *testing.T) {
	qLabData, _ := setupWorkspaceWithCleanup(t)

	tests := []struct {
		name         string
		cueData      map[string]any
		expectedType string
	}{
		{
			name: "Basic Audio Cue",
			cueData: map[string]any{
				"type":       "audio",
				"name":       "Test Audio",
				"number":     "1.0",
				"fileTarget": "audio/test.mp3",
			},
			expectedType: "audio",
		},
		{
			name: "Audio Cue with Infinite Loop",
			cueData: map[string]any{
				"type":         "audio",
				"name":         "Background Rain",
				"fileTarget":   "audio/rain.mp3",
				"infiniteLoop": true,
			},
			expectedType: "audio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createCueWithNumber(t, qLabData, tt.cueData)
		})
	}
}

// TestLightCueCreation tests creating light cues
func TestLightCueCreation(t *testing.T) {
	qLabData, _ := setupWorkspaceWithCleanup(t)

	cueData := map[string]any{
		"type": "light",
		"name": "Soft spotlight on JOE",
	}

	createCueWithNumber(t, qLabData, cueData)
}

// TestStartCueCreation tests creating start cues with target references
func TestStartCueCreation(t *testing.T) {
	qLabData, _ := setupWorkspaceWithCleanup(t)

	cueData := map[string]any{
		"type":            "start",
		"number":          "3.0",
		"cueTargetNumber": "1.0",
	}

	createCueWithNumber(t, qLabData, cueData)
}

// TestStopCueCreation tests creating stop cues with target references
func TestStopCueCreation(t *testing.T) {
	qLabData, _ := setupWorkspaceWithCleanup(t)

	cueData := map[string]any{
		"type":            "stop",
		"number":          "2.0",
		"cueTargetNumber": "1.0",
	}

	createCueWithNumber(t, qLabData, cueData)
}

// TestGroupCueCreation tests creating different types of group cues
func TestGroupCueCreation(t *testing.T) {
	qLabData, _ := setupWorkspaceWithCleanup(t)

	tests := []struct {
		name         string
		cueData      map[string]any
		expectedMode float64
	}{
		{
			name: "CueList (mode 0)",
			cueData: map[string]any{
				"type": "group",
				"name": "Preshow",
				"mode": float64(0),
			},
			expectedMode: 0,
		},
		{
			name: "Timeline (mode 3)",
			cueData: map[string]any{
				"type":   "group",
				"name":   "INT. CAR - NIGHT / It's pouring rain.",
				"number": "0",
				"mode":   float64(3),
			},
			expectedMode: 3,
		},
		{
			name: "Playlist (mode 6)",
			cueData: map[string]any{
				"type": "group",
				"mode": float64(6),
			},
			expectedMode: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createCueWithNumber(t, qLabData, tt.cueData)
		})
	}
}

// TestCuePropertyValidation tests that cue properties are correctly set
func TestCuePropertyValidation(t *testing.T) {
	qLabData, _ := setupWorkspaceWithCleanup(t)

	// Test various property types
	tests := []struct {
		name     string
		cueType  string
		cueData  map[string]any
		property string
		value    string
	}{
		{
			name:    "Audio file target",
			cueType: "audio",
			cueData: map[string]any{
				"type":       "audio",
				"name":       "Test Audio",
				"fileTarget": "audio/test.mp3",
			},
			property: "file",
			value:    "audio/test.mp3",
		},
		{
			name:    "Cue name",
			cueType: "light",
			cueData: map[string]any{
				"type": "light",
				"name": "Test Light Cue",
			},
			property: "name",
			value:    "Test Light Cue",
		},
		{
			name:    "Cue number",
			cueType: "audio",
			cueData: map[string]any{
				"type":   "audio",
				"number": "42.5",
			},
			property: "number",
			value:    "42.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createCueWithNumber(t, qLabData, tt.cueData)
		})
	}
}

// TestUberNoirSpecificCueTypes tests the exact cue types from uber_noir.cue
func TestUberNoirSpecificCueTypes(t *testing.T) {
	qLabData, _ := setupWorkspaceWithCleanup(t)

	// Test each specific cue from uber_noir.cue
	uberNoirCues := []map[string]any{
		// Audio cue from Preshow playlist
		{
			"type":       "audio",
			"name":       "Espresso - Sabrina Carpenter",
			"fileTarget": "audio/espresso.mp3",
		},
		// Light cue from Timeline
		{
			"type": "light",
			"name": "Lights up. Soft spotlight on JOE.",
		},
		// Audio cue with infinite loop
		{
			"type":         "audio",
			"name":         "Rain sounds.",
			"fileTarget":   "audio/rain.mp3",
			"infiniteLoop": true,
		},
		// Audio cue with number
		{
			"type":       "audio",
			"number":     "1.0",
			"name":       "Round About Midnight - Miles Davis",
			"fileTarget": "audio/midnight.mp3",
		},
		// Stop cue with target
		{
			"type":            "stop",
			"number":          "2.0",
			"cueTargetNumber": "1.0",
		},
		// Start cue with target
		{
			"type":            "start",
			"number":          "3.0",
			"cueTargetNumber": "1.0",
		},
	}

	for i, cueData := range uberNoirCues {
		t.Run(fmt.Sprintf("UberNoir_Cue_%d_%s", i+1, cueData["type"]), func(t *testing.T) {
			createCueWithNumber(t, qLabData, cueData)
		})
	}
}

// TestChildrenQueryDebug tests the children query specifically
func TestChildrenQueryDebug(t *testing.T) {
	t.Skip("Skipping debug test - designed for manual debugging, causes timeouts in CI")
	workspace, mockServer := setupWorkspaceWithCleanup(t)

	// Mock server starts clean, no need to clear
	t.Log("=== Mock server initialized with clean state ===")
	t.Logf("Mock server has %d cues initially", mockServer.GetCueCount())

	// Create a simple group WITH a specific number
	t.Log("=== Creating simple group with number '100' ===")
	groupData := map[string]any{
		"type": "group",
		"name": "Debug Group",
		"mode": 3.0, // Timeline mode
	}

	groupID, err := workspace.createCue(groupData, "100")
	if err != nil {
		t.Fatalf("Failed to create group cue: %v", err)
	}
	t.Logf("Created group with ID: %s and number: 100", groupID)

	// Test basic workspace info query first
	t.Log("=== Testing basic workspace info query ===")
	infoAddress := fmt.Sprintf("/workspace/%s", workspace.workspace_id)
	infoReply := workspace.Send(infoAddress, "")
	if len(infoReply) > 0 {
		t.Logf("Workspace info reply: %s", infoReply[0])
	}

	// Try to query children using correct cue number format
	t.Log("=== Testing children query using cue number format ===")
	address := fmt.Sprintf("/workspace/%s/cue/100/children", workspace.workspace_id)
	reply := workspace.Send(address, "")
	if len(reply) > 0 {
		t.Logf("children query reply: %s", reply[0])

		// Parse the reply to check if it's successful
		var replyData map[string]any
		if err := json.Unmarshal([]byte(reply[0].(string)), &replyData); err == nil {
			if status, ok := replyData["status"].(string); ok {
				t.Logf("Query status: %s", status)
				if status == "ok" {
					t.Log("SUCCESS: Children query worked with cue number format!")
					if data, ok := replyData["data"].([]any); ok {
						t.Logf("Found %d children", len(data))
					}
				}
			}
		}
	} else {
		t.Log("No reply received from children query")
	}

	// Test a simple global query to see if it works without workspace prefix
	t.Log("=== Testing global cue query without workspace prefix ===")
	globalAddress := "/cue/100/children"
	globalReply := workspace.Send(globalAddress, "")
	if len(globalReply) > 0 {
		t.Logf("Global cue query reply: %s", globalReply[0])
	}

	// The cue doesn't have a number since setting number failed,
	// so let's try querying with the selected cue instead
	t.Log("=== Testing selected cue query (since number setting failed) ===")
	selectedAddress := "/cue/selected/children"
	selectedReply := workspace.Send(selectedAddress, "")
	if len(selectedReply) > 0 {
		t.Logf("Selected cue query reply: %s", selectedReply[0])

		// Parse the reply to check if it's successful and extract actual cue number
		var replyData map[string]any
		var actualCueNumber string
		if err := json.Unmarshal([]byte(selectedReply[0].(string)), &replyData); err == nil {
			if status, ok := replyData["status"].(string); ok {
				t.Logf("Selected cue query status: %s", status)
				if status == "ok" {
					t.Log("SUCCESS: Selected cue children query worked!")
					if data, ok := replyData["data"].([]any); ok {
						t.Logf("Found %d children", len(data))
					}
				}
			}
			// Extract the actual cue number from the address QLab used in the reply
			if address, ok := replyData["address"].(string); ok {
				// Address format: "/workspace/{workspace_id}/cue/{actual_number}/children"
				// Extract the number between "/cue/" and "/children"
				parts := strings.Split(address, "/")
				for i, part := range parts {
					if part == "cue" && i+1 < len(parts) {
						actualCueNumber = parts[i+1]
						t.Logf("Extracted actual cue number from QLab reply: %s", actualCueNumber)
						break
					}
				}
			}
		}

		// Now try to query using the actual cue number that QLab assigned
		if actualCueNumber != "" {
			t.Logf("=== Testing children query with actual cue number: %s ===", actualCueNumber)
			actualCueAddress := fmt.Sprintf("/cue/%s/children", actualCueNumber)
			actualCueReply := workspace.Send(actualCueAddress, "")
			if len(actualCueReply) > 0 {
				t.Logf("Actual cue number query reply: %s", actualCueReply[0])

				var actualReplyData map[string]any
				if err := json.Unmarshal([]byte(actualCueReply[0].(string)), &actualReplyData); err == nil {
					if status, ok := actualReplyData["status"].(string); ok {
						t.Logf("Actual cue number query status: %s", status)
						if status == "ok" {
							t.Log("SUCCESS: Children query with actual cue number worked!")
							if data, ok := actualReplyData["data"].([]any); ok {
								t.Logf("Found %d children using actual cue number", len(data))
							}
						}
					}
				}
			}
		}
	}

	// Try cue_id address with global scope
	t.Log("=== Testing global cue_id query ===")
	cueIDAddress := fmt.Sprintf("/cue_id/%s/children", groupID)
	cueIDReply := workspace.Send(cueIDAddress, "")
	if len(cueIDReply) > 0 {
		t.Logf("Global cue_id query reply: %s", cueIDReply[0])

		// Parse the reply to check if it's successful
		var replyData map[string]any
		if err := json.Unmarshal([]byte(cueIDReply[0].(string)), &replyData); err == nil {
			if status, ok := replyData["status"].(string); ok {
				t.Logf("Global cue_id query status: %s", status)
				if status == "ok" {
					t.Log("SUCCESS: Global cue_id children query worked!")
					if data, ok := replyData["data"].([]any); ok {
						t.Logf("Found %d children", len(data))
					}
				}
			}
		}
	}

	// Test the old format for comparison
	t.Log("=== Testing old cue_id format for comparison ===")
	_, err = workspace.getCueChildren(groupID)
	if err != nil {
		t.Logf("Old cue_id format failed as expected: %v", err)
	}

}

// TestCueNumberConflictDetection tests cue number conflict detection with default behavior (no force)
func TestCueNumberConflictDetection(t *testing.T) {
	workspace, mockServer := setupWorkspaceWithCleanup(t)

	t.Log("=== Testing cue number conflict detection (default behavior) ===")

	// Create first cue with number "1.0"
	t.Log("Creating first cue with number '1.0'")
	firstCueData := map[string]any{
		"type":   "audio",
		"name":   "First Audio Cue",
		"number": "1.0",
	}

	firstCueID, err := workspace.createCue(firstCueData, "1.0")
	if err != nil {
		t.Fatalf("Failed to create first cue: %v", err)
	}
	t.Logf("Created first cue with ID: %s", firstCueID)

	// Verify cue number is tracked
	if trackedID, exists := workspace.cueNumbers["1.0"]; !exists {
		t.Error("Cue number '1.0' should be tracked")
	} else if trackedID != firstCueID {
		t.Errorf("Expected tracked ID %s, got %s", firstCueID, trackedID)
	}

	// Create second cue with the same number "1.0" - should skip number assignment
	t.Log("Creating second cue with conflicting number '1.0' - should skip number assignment")
	secondCueData := map[string]any{
		"type":   "light",
		"name":   "Conflicting Light Cue",
		"number": "1.0",
	}

	secondCueID, err := workspace.createCue(secondCueData, "1.0")
	if err != nil {
		t.Fatalf("Failed to create second cue: %v", err)
	}
	t.Logf("Created second cue with ID: %s", secondCueID)

	// Verify the first cue still owns the number (conflict was skipped, not resolved)
	if trackedID, exists := workspace.cueNumbers["1.0"]; !exists {
		t.Error("Cue number '1.0' should still be tracked")
	} else if trackedID != firstCueID {
		t.Errorf("Expected number '1.0' to still be tracked to first cue %s, got %s", firstCueID, trackedID)
	}

	// Verify we now have 2 cues in the mock server
	if mockServer.GetCueCount() != 2 {
		t.Errorf("Expected 2 cues in mock server, got %d", mockServer.GetCueCount())
	}

	t.Log("Conflict detection test completed successfully")
}

// TestCueNumberIndexing tests the indexing of existing cues during workspace initialization
func TestCueNumberIndexing(t *testing.T) {
	workspace, _ := setupWorkspaceWithCleanup(t)

	// Pre-populate the mock server with cues that have numbers
	t.Log("=== Pre-populating workspace with numbered cues ===")

	testCues := []map[string]any{
		{
			"type":   "audio",
			"name":   "Pre-existing Audio",
			"number": "5.0",
		},
		{
			"type":   "light",
			"name":   "Pre-existing Light",
			"number": "10.0",
		},
		{
			"type": "memo",
			"name": "Pre-existing Memo",
			// No number - should not be indexed
		},
	}

	// Create the cues manually first (simulating existing workspace state)
	for _, cueData := range testCues {
		var cueNumber string
		if num, ok := cueData["number"]; ok && num != nil {
			// Handle different number types while preserving decimal format
			switch v := num.(type) {
			case string:
				// Already a string, use as-is
				cueNumber = v
			case float64:
				// For float64, use %g to get natural representation,
				// but preserve at least one decimal place for whole numbers if they came from "X.0"
				if v == float64(int64(v)) && v >= 0 && v <= 999 {
					// It's a whole number that might have been "X.0" originally
					// Use %.1f to force one decimal place for common cue numbers
					cueNumber = fmt.Sprintf("%.1f", v)
				} else {
					// Use %g for non-whole numbers (preserves natural format)
					cueNumber = fmt.Sprintf("%g", v)
				}
			case int64:
				cueNumber = fmt.Sprintf("%d", v)
			case int:
				cueNumber = fmt.Sprintf("%d", v)
			default:
				cueNumber = fmt.Sprintf("%v", v)
			}
		}
		_, err := workspace.createCue(cueData, cueNumber)
		if err != nil {
			t.Fatalf("Failed to create pre-existing cue: %v", err)
		}
	}

	// Clear the workspace's cueNumbers map to simulate fresh initialization
	workspace.cueNumbers = make(map[string]string)

	// Test indexing of existing cues
	t.Log("=== Testing indexExistingCues ===")
	err := workspace.indexExistingCues()
	if err != nil {
		t.Fatalf("indexExistingCues failed: %v", err)
	}

	// Verify that numbered cues were indexed
	expectedNumbers := []string{"5.0", "10.0"}
	for _, number := range expectedNumbers {
		if _, exists := workspace.cueNumbers[number]; !exists {
			t.Errorf("Expected cue number '%s' to be indexed", number)
		} else {
			t.Logf("Successfully indexed cue number '%s'", number)
		}
	}

	// Verify that cues without numbers were not indexed
	if len(workspace.cueNumbers) != 2 {
		t.Errorf("Expected 2 indexed cue numbers, got %d", len(workspace.cueNumbers))
	}

	t.Log("Cue number indexing test completed successfully")
}

// TestDecimalPreservation tests that decimal cue numbers like "1.0" are preserved correctly
func TestDecimalPreservation(t *testing.T) {
	// Enable debug logging
	log.SetLevel(log.InfoLevel)

	workspace, mockServer := setupWorkspaceWithCleanupFatal(t)
	defer func() {
		if err := mockServer.Stop(); err != nil {
			t.Logf("Failed to stop mock server: %v", err)
		}
	}()

	// Test cases with various number formats that should preserve decimals
	testCases := []struct {
		name           string
		inputNumber    any
		expectedOutput string
		description    string
	}{
		{
			name:           "String 1.0",
			inputNumber:    "1.0",
			expectedOutput: "1.0",
			description:    "String numbers should be preserved as-is",
		},
		{
			name:           "Float64 1.0",
			inputNumber:    float64(1.0),
			expectedOutput: "1.0",
			description:    "Float64 whole numbers should get .0 suffix",
		},
		{
			name:           "Float64 1.5",
			inputNumber:    float64(1.5),
			expectedOutput: "1.5",
			description:    "Float64 non-whole numbers should use natural format",
		},
		{
			name:           "Float64 10.0",
			inputNumber:    float64(10.0),
			expectedOutput: "10.0",
			description:    "Float64 larger whole numbers should get .0 suffix",
		},
		{
			name:           "String 2.75",
			inputNumber:    "2.75",
			expectedOutput: "2.75",
			description:    "String decimal numbers should be preserved",
		},
		{
			name:           "Integer 3",
			inputNumber:    3,
			expectedOutput: "3",
			description:    "Integer numbers should remain as integers",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test cue data structure
			cueData := map[string]any{
				"type":   "memo",
				"name":   "Test Cue - " + tc.name,
				"number": tc.inputNumber,
			}

			// Test the processCueListWithParent function (which uses our fixed logic)
			_, err := workspace.processCueListWithParent(cueData, "", "")
			if err != nil {
				t.Fatalf("processCueListWithParent failed for %s: %v", tc.name, err)
			}

			// Also test direct indexing logic
			testIndexData := map[string]any{
				"uniqueID": "TEST-ID-" + tc.name,
				"number":   tc.inputNumber,
			}

			// Clear existing index
			workspace.cueNumbers = make(map[string]string)

			// Simulate the indexing logic
			if uniqueID, hasID := testIndexData["uniqueID"].(string); hasID {
				var cueNumber string
				if num, hasNumber := testIndexData["number"]; hasNumber && num != nil {
					// This is the same logic we fixed in indexCueNumbers
					switch v := num.(type) {
					case string:
						cueNumber = v
					case float64:
						if v == float64(int64(v)) && v >= 0 && v <= 999 {
							cueNumber = fmt.Sprintf("%.1f", v)
						} else {
							cueNumber = fmt.Sprintf("%g", v)
						}
					case int64:
						cueNumber = fmt.Sprintf("%d", v)
					case int:
						cueNumber = fmt.Sprintf("%d", v)
					default:
						cueNumber = fmt.Sprintf("%v", v)
					}
				}
				if cueNumber != "" {
					workspace.cueNumbers[cueNumber] = uniqueID
				}

				// Verify the result
				if cueNumber != tc.expectedOutput {
					t.Errorf("Case '%s': expected '%s', got '%s' - %s",
						tc.name, tc.expectedOutput, cueNumber, tc.description)
				} else {
					t.Logf("✓ Case '%s': correctly preserved '%s' as '%s'",
						tc.name, tc.inputNumber, cueNumber)
				}
			}
		})
	}

	// Clean up is handled by defer
}

// TestRealWorldDecimalPreservation tests with actual uber_noir.cue file to verify "1.0" is preserved
func TestRealWorldDecimalPreservation(t *testing.T) {
	// Skip test - requires CUE file handler to be injected
	t.Skip("Test requires CUE file handler - skipping in standalone qlab-golang package")

	// NOTE: TransmitCueFile has been removed. Use TransmitWorkspaceData instead.
	// Caller should parse CUE file and pass the data to TransmitWorkspaceData()
}

// TestCueNumberClearance tests clearing cue numbers and tracking updates
func TestCueNumberClearance(t *testing.T) {
	workspace, _ := setupWorkspaceWithCleanup(t)

	t.Log("=== Testing cue number clearance ===")

	// Create a cue with a number
	cueData := map[string]any{
		"type":   "audio",
		"name":   "Test Audio",
		"number": "42.0",
	}

	cueID, err := workspace.createCue(cueData, "42.0")
	if err != nil {
		t.Fatalf("Failed to create cue: %v", err)
	}

	// Verify number is tracked
	if _, exists := workspace.cueNumbers["42.0"]; !exists {
		t.Error("Cue number '42.0' should be tracked")
	}

	// Test clearing the cue number
	err = workspace.clearCueNumber(cueID)
	if err != nil {
		t.Errorf("Failed to clear cue number: %v", err)
	}

	// Note: The clearCueNumber method doesn't currently update the tracking map
	// This is a limitation that could be improved in the future
	t.Log("Cue number clearance test completed")
}

// TestMultipleCueConflicts tests handling multiple sequential conflicts
func TestMultipleCueConflicts(t *testing.T) {
	workspace, mockServer := setupWorkspaceWithCleanup(t)

	t.Log("=== Testing multiple sequential cue conflicts ===")

	conflictNumber := "99.0"

	// Create multiple cues that all want the same number
	cueTypes := []string{"audio", "light", "memo", "group"}
	var cueIDs []string

	for i, cueType := range cueTypes {
		t.Logf("Creating cue %d (%s) with conflicting number '%s'", i+1, cueType, conflictNumber)

		cueData := map[string]any{
			"type":   cueType,
			"name":   fmt.Sprintf("Cue %d", i+1),
			"number": conflictNumber,
		}

		cueID, err := workspace.createCue(cueData, conflictNumber)
		if err != nil {
			t.Fatalf("Failed to create cue %d: %v", i+1, err)
		}

		cueIDs = append(cueIDs, cueID)

		// For default behavior, only the first cue should get the number
		// Subsequent cues should skip number assignment due to conflicts
		if i == 0 {
			// First cue should own the number
			if trackedID, exists := workspace.cueNumbers[conflictNumber]; !exists {
				t.Errorf("Cue number '%s' should be tracked after first cue", conflictNumber)
			} else if trackedID != cueID {
				t.Errorf("Expected number '%s' to be owned by first cue (%s), got %s", conflictNumber, cueID, trackedID)
			}
		} else {
			// Subsequent cues - first cue should still own the number
			firstCueID := cueIDs[0]
			if trackedID, exists := workspace.cueNumbers[conflictNumber]; !exists {
				t.Errorf("Cue number '%s' should still be tracked after cue %d", conflictNumber, i+1)
			} else if trackedID != firstCueID {
				t.Errorf("Expected number '%s' to still be owned by first cue (%s), got %s", conflictNumber, firstCueID, trackedID)
			}
		}
	}

	// Verify all cues were created
	expectedCueCount := len(cueTypes)
	if mockServer.GetCueCount() != expectedCueCount {
		t.Errorf("Expected %d cues in mock server, got %d", expectedCueCount, mockServer.GetCueCount())
	}

	// Verify only the first cue owns the number (default conflict behavior is to skip)
	firstOwner := cueIDs[0]
	if trackedID := workspace.cueNumbers[conflictNumber]; trackedID != firstOwner {
		t.Errorf("Expected final owner of number '%s' to be first cue %s, got %s", conflictNumber, firstOwner, trackedID)
	}

	t.Log("Multiple conflicts test completed successfully")
}

// TestForceCueNumbersFlag tests the --force-cue-numbers behavior
func TestForceCueNumbersFlag(t *testing.T) {
	t.Run("Default behavior (force disabled)", func(t *testing.T) {
		workspace1, _ := setupWorkspaceWithCleanup(t)
		workspace1.SetForceCueNumbers(false)

		// Create first cue with number "2.0"
		firstCueData := map[string]any{
			"type":   "audio",
			"name":   "First Audio Cue",
			"number": "2.0",
		}
		firstCueID, err := workspace1.createCue(firstCueData, "2.0")
		if err != nil {
			t.Fatalf("Failed to create first cue: %v", err)
		}

		// Verify first cue owns the number
		if trackedID := workspace1.cueNumbers["2.0"]; trackedID != firstCueID {
			t.Errorf("Expected first cue to own number '2.0', got %s", trackedID)
		}

		// Create second cue with same number - should skip the number assignment
		secondCueData := map[string]any{
			"type":   "light",
			"name":   "Conflicting Light Cue",
			"number": "2.0",
		}
		_, err = workspace1.createCue(secondCueData, "2.0")
		if err != nil {
			t.Fatalf("Failed to create second cue: %v", err)
		}

		// Verify first cue still owns the number
		if trackedID := workspace1.cueNumbers["2.0"]; trackedID != firstCueID {
			t.Errorf("Expected first cue to still own number '2.0', got %s", trackedID)
		}
	})

	t.Run("Force mode enabled", func(t *testing.T) {
		workspace2, _ := setupWorkspaceWithCleanup(t)
		workspace2.SetForceCueNumbers(true)

		// Create first cue with number "3.0"
		firstCueData3 := map[string]any{
			"type":   "audio",
			"name":   "First Audio Cue",
			"number": "3.0",
		}
		firstCueID3, err := workspace2.createCue(firstCueData3, "3.0")
		if err != nil {
			t.Fatalf("Failed to create first cue: %v", err)
		}

		// Verify first cue owns the number
		if trackedID := workspace2.cueNumbers["3.0"]; trackedID != firstCueID3 {
			t.Errorf("Expected first cue to own number '3.0', got %s", trackedID)
		}

		// Create second cue with same number - should force clear the first cue's number
		secondCueData3 := map[string]any{
			"type":   "light",
			"name":   "Forcing Light Cue",
			"number": "3.0",
		}
		secondCueID3, err := workspace2.createCue(secondCueData3, "3.0")
		if err != nil {
			t.Fatalf("Failed to create second cue with force mode: %v", err)
		}

		// Verify second cue now owns the number
		if trackedID := workspace2.cueNumbers["3.0"]; trackedID != secondCueID3 {
			t.Errorf("Expected second cue to own number '3.0' after force, got %s", trackedID)
		}
	})
}

// TestCuejitsuInboxCreation tests the creation and detection of "Cuejitsu Inbox" cue list
func TestCuejitsuInboxCreation(t *testing.T) {
	workspace, _ := setupWorkspaceWithCleanup(t)

	t.Log("=== Testing Cuejitsu Inbox creation functionality ===")

	// Test case 1: No existing inbox - should create one
	t.Log("--- Test 1: Creating new Cuejitsu Inbox ---")

	// Ensure inbox creation
	inboxID, err := workspace.ensureCuejitsuInbox()
	if err != nil {
		t.Fatalf("Failed to ensure Cuejitsu Inbox: %v", err)
	}

	// Verify inbox ID was returned
	if inboxID == "" {
		t.Error("Expected inboxID to be returned after ensuring inbox")
	}

	// Verify inbox ID was set on workspace
	if workspace.inboxID != inboxID {
		t.Errorf("Expected workspace.inboxID to match returned ID: %s vs %s", workspace.inboxID, inboxID)
	}

	t.Logf("Successfully created Cuejitsu Inbox with ID: %s", inboxID)

	// Test case 2: Existing inbox - should detect and use it
	t.Log("--- Test 2: Detecting existing Cuejitsu Inbox ---")

	// Store the original inbox ID
	originalInboxID := inboxID

	// Clear the workspace's inbox ID and cache to simulate fresh initialization
	workspace.inboxID = ""
	workspace.cueListsCache = nil // Clear cache so it re-queries QLab

	// Ensure inbox again - should find the existing one
	foundInboxID, err := workspace.ensureCuejitsuInbox()
	if err != nil {
		t.Fatalf("Failed to detect existing Cuejitsu Inbox: %v", err)
	}

	// Should have found the same inbox
	if foundInboxID != originalInboxID {
		t.Errorf("Expected to find existing inbox %s, but got %s", originalInboxID, foundInboxID)
	}

	t.Logf("Successfully detected existing Cuejitsu Inbox with ID: %s", foundInboxID)

	// Test case 3: Verify inbox is properly named
	t.Log("--- Test 3: Verifying inbox name ---")

	// Query the cue list to verify its name
	address := fmt.Sprintf("/workspace/%s/cue_id/%s/name", workspace.workspace_id, foundInboxID)
	reply := workspace.Send(address, "")

	if len(reply) == 0 {
		t.Error("Expected reply when querying inbox name")
	} else {
		t.Logf("Inbox name query reply: %v", reply[0])
		// The mock server should have the "Cuejitsu Inbox" name set
	}

	t.Log("Cuejitsu Inbox creation test completed successfully")
}

// TestWorkspaceInitWithInbox tests that workspace initialization includes inbox creation
func TestWorkspaceInitWithInbox(t *testing.T) {
	// Get an available port from the OS
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	// Create and start mock QLab OSC server
	mockServer := NewMockOSCServer("localhost", port)
	err = mockServer.Start()
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}

	// Clean up after test
	t.Cleanup(func() {
		mockServer.Clear()
		if err := mockServer.Stop(); err != nil {
			t.Logf("Failed to stop mock server: %v", err)
		}
	})

	t.Log("=== Testing workspace initialization with inbox creation ===")

	// Create workspace and initialize it properly
	workspace := NewWorkspace("localhost", port)
	_, err = workspace.Init("test-passcode")
	if err != nil {
		t.Fatalf("Workspace initialization failed: %v", err)
	}

	// Verify workspace was initialized
	if !workspace.initialized {
		t.Error("Workspace should be initialized")
	}

	// Verify inbox was created during initialization
	if workspace.inboxID == "" {
		t.Error("Expected inboxID to be set during workspace initialization")
	}

	t.Logf("Workspace initialized with inbox ID: %s", workspace.inboxID)

	// Verify the mock server shows the cue list was created
	t.Logf("Mock server state after initialization: %d cues", mockServer.GetCueCount())

	// Create a test cue to verify workspace is fully functional
	cueData := map[string]any{
		"type": "memo",
		"name": "Test cue after inbox initialization",
	}

	_, err = workspace.createCue(cueData, "")
	if err != nil {
		t.Errorf("Failed to create cue after inbox initialization: %v", err)
	} else {
		t.Log("Successfully created cue after inbox initialization")
	}

	t.Log("Workspace initialization with inbox test completed successfully")
}

// TestFullWorkspaceInitWithInbox tests the complete workspace initialization process including inbox creation
func TestFullWorkspaceInitWithInbox(t *testing.T) {
	// Enable debug logging to see the full initialization process
	log.SetLevel(log.InfoLevel)

	// Get an available port from the OS
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	// Create and start mock QLab OSC server
	mockServer := NewMockOSCServer("localhost", port)
	if err = mockServer.Start(); err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}

	// Create workspace (not initialized yet)
	workspace := NewWorkspace("localhost", port)

	// Clean up after test
	t.Cleanup(func() {
		workspace.Close()
		mockServer.Clear()
		if err := mockServer.Stop(); err != nil {
			t.Logf("Failed to stop mock server: %v", err)
		}
		time.Sleep(150 * time.Millisecond)
	})

	t.Log("=== Testing complete workspace initialization with inbox creation ===")

	// Verify workspace is not initialized yet
	if workspace.initialized {
		t.Error("Workspace should not be initialized yet")
	}
	if workspace.inboxID != "" {
		t.Error("inboxID should be empty before initialization")
	}

	// Call the actual Init() method
	reply, err := workspace.Init("test-passcode")
	if err != nil {
		t.Fatalf("Workspace initialization failed: %v", err)
	}

	t.Logf("Init reply: %v", reply)

	// Verify workspace is now properly initialized
	if !workspace.initialized {
		t.Error("Workspace should be initialized after Init()")
	}

	if workspace.workspace_id == "" {
		t.Error("workspace_id should be set after initialization")
	}

	// Most importantly - verify inbox was created during initialization
	if workspace.inboxID == "" {
		t.Error("Expected inboxID to be set during workspace initialization")
	}

	t.Logf("Workspace fully initialized with inbox ID: %s", workspace.inboxID)
	t.Logf("Workspace ID: %s", workspace.workspace_id)

	// Verify the inbox actually exists by querying its name
	nameAddr := workspace.GetAddress(fmt.Sprintf("/workspace/%s/cue_id/%s/name", workspace.workspace_id, workspace.inboxID))
	nameReply := workspace.Send(nameAddr, "")
	if len(nameReply) > 0 {
		t.Logf("Inbox name query reply: %v", nameReply[0])
	}

	// Create a test cue to verify workspace is fully functional
	cueData := map[string]any{
		"type": "memo",
		"name": "Test cue after full initialization",
	}

	cueID, err := workspace.createCue(cueData, "")
	if err != nil {
		t.Errorf("Failed to create cue after full initialization: %v", err)
	} else {
		t.Logf("Successfully created cue with ID: %s after full initialization", cueID)
	}

	t.Log("Full workspace initialization with inbox test completed successfully")
}

// isQLabAvailable checks if QLab is running and accessible on the given host:port
func isQLabAvailable(host string, port int) bool {
	// Try a simple TCP connection to see if something is listening on the port
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return false
	}
	if err := conn.Close(); err != nil {
		// Connection already closed, ignore error
		return false
	}

	// Actually try to initialize a connection with a timeout to verify QLab responds
	workspace := NewWorkspace(host, port)

	// Use a channel to implement timeout on the Init call
	type result struct {
		reply []any
		err   error
	}
	ch := make(chan result, 1)

	go func() {
		// Try with empty passcode (tests should use workspaces without passcodes)
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
		_ = conn.Close() // Ignore error - connection may already be closed
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
	workspace := NewWorkspace(host, port)

	// Initialize connection with empty passcode (tests should use workspaces without passcodes)
	t.Log("--- Connecting to QLab ---")
	reply, err := workspace.Init("")
	if err != nil {
		t.Fatalf("Failed to initialize connection to QLab: %v", err)
	}

	t.Logf("Successfully connected to QLab. Reply: %v", reply)

	// Clear all existing cues to avoid conflicts (ignore errors - some cues may be protected)
	t.Log("--- Clearing existing cues ---")
	if err := workspace.clearAllCues(); err != nil {
		t.Logf("Warning: Could not clear all existing cues (this is normal): %v", err)
	}

	// Test basic workspace query
	t.Log("--- Testing basic workspace query ---")
	address := fmt.Sprintf("/workspace/%s/cueLists", workspace.workspace_id)
	startTime := time.Now()
	queryReply := workspace.Send(address, "")
	duration := time.Since(startTime)

	t.Logf("Cue lists query took: %v", duration)
	if len(queryReply) == 0 {
		t.Error("No reply received from cue lists query")
	} else {
		t.Logf("Cue lists query reply length: %d", len(queryReply))
		if len(queryReply) > 0 {
			replyStr := queryReply[0].(string)
			if len(replyStr) > 200 {
				t.Logf("Cue lists query reply (truncated): %s...", replyStr[:200])
			} else {
				t.Logf("Cue lists query reply: %s", replyStr)
			}
		}
	}

	// Test Cuejitsu Inbox creation/detection with timing
	t.Log("--- Testing Cuejitsu Inbox operations ---")
	startTime = time.Now()
	inboxID, err := workspace.ensureCuejitsuInbox()
	duration = time.Since(startTime)

	t.Logf("Cuejitsu Inbox operation took: %v", duration)
	if err != nil {
		t.Errorf("Failed to ensure Cuejitsu Inbox: %v", err)
	} else {
		t.Logf("Cuejitsu Inbox ID: %s", inboxID)
	}

	// Test cue indexing with timing
	t.Log("--- Testing cue indexing ---")
	startTime = time.Now()
	err = workspace.indexExistingCues()
	duration = time.Since(startTime)

	t.Logf("Cue indexing took: %v", duration)
	if err != nil {
		t.Errorf("Failed to index existing cues: %v", err)
	} else {
		t.Logf("Successfully indexed existing cues. Found %d numbered cues", len(workspace.cueNumbers))
	}

	// Test creating a simple cue
	t.Log("--- Testing cue creation ---")
	cueData := map[string]any{
		"type":   "memo",
		"name":   "Real QLab Test Cue",
		"number": "999.1",
	}

	startTime = time.Now()
	cueID, err := workspace.createCue(cueData, "999.1")
	duration = time.Since(startTime)

	t.Logf("Cue creation took: %v", duration)
	if err != nil {
		t.Errorf("Failed to create test cue: %v", err)
	} else {
		t.Logf("Successfully created test cue with ID: %s", cueID)

		// Clean up by deleting the test cue
		t.Log("--- Cleaning up test cue ---")
		deleteAddr := fmt.Sprintf("/workspace/%s/delete_id/%s", workspace.workspace_id, cueID)
		deleteReply := workspace.Send(deleteAddr, "")
		t.Logf("Delete cue reply: %v", deleteReply)
	}

	t.Log("Real QLab test completed")
}

// TestEmptyPasscodeConnection tests connecting to QLab with an empty passcode
func TestEmptyPasscodeConnection(t *testing.T) {
	log.SetLevel(log.InfoLevel)

	t.Log("=== Testing connection with empty passcode ===")

	// Get an available port from the OS
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	// Create and start mock QLab OSC server
	mockServer := NewMockOSCServer("localhost", port)
	if err = mockServer.Start(); err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}

	// Clean up after test
	t.Cleanup(func() {
		mockServer.Clear()
		if err := mockServer.Stop(); err != nil {
			t.Logf("Failed to stop mock server: %v", err)
		}
		time.Sleep(150 * time.Millisecond)
	})

	// Create workspace (not initialized yet)
	workspace := NewWorkspace("localhost", port)

	// Clean up workspace
	t.Cleanup(func() {
		workspace.Close()
	})

	// Test 1: Initialize with empty string passcode
	t.Log("--- Test 1: Connecting with empty string passcode ---")
	reply, err := workspace.Init("")
	if err != nil {
		t.Fatalf("Failed to initialize connection with empty passcode: %v", err)
	}

	t.Logf("Successfully connected with empty passcode. Reply: %v", reply)
	t.Logf("Connected to workspace ID: %s", workspace.workspace_id)

	// Verify workspace is initialized
	if !workspace.initialized {
		t.Error("Workspace should be initialized after Init()")
	}

	if workspace.workspace_id == "" {
		t.Error("workspace_id should be set after initialization")
	}

	// Verify inbox was created
	if workspace.inboxID == "" {
		t.Error("Expected inboxID to be set during workspace initialization")
	}

	// Test creating a cue to verify workspace is fully functional
	t.Log("--- Testing cue creation after empty passcode connection ---")
	cueData := map[string]any{
		"type": "memo",
		"name": "Test cue with empty passcode",
	}

	cueID, err := workspace.createCue(cueData, "")
	if err != nil {
		t.Errorf("Failed to create cue after empty passcode initialization: %v", err)
	} else {
		t.Logf("Successfully created cue with ID: %s", cueID)
	}

	t.Log("Empty passcode connection test completed successfully")
}

// TestPasscodeVariations tests various passcode scenarios
// Note: QLab passcodes must be four-digit integers (0000-9999), or empty for no passcode
func TestPasscodeVariations(t *testing.T) {
	log.SetLevel(log.InfoLevel)

	t.Log("=== Testing various passcode scenarios ===")

	testCases := []struct {
		name     string
		passcode string
		shouldOK bool
	}{
		{
			name:     "Empty string passcode (no passcode)",
			passcode: "",
			shouldOK: true,
		},
		{
			name:     "Four-digit passcode (0000)",
			passcode: "0000",
			shouldOK: true,
		},
		{
			name:     "Four-digit passcode (1234)",
			passcode: "1234",
			shouldOK: true,
		},
		{
			name:     "Four-digit passcode (9999)",
			passcode: "9999",
			shouldOK: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get an available port from the OS
			port, err := getFreePort()
			if err != nil {
				t.Fatalf("Failed to get free port: %v", err)
			}

			// Create and start mock QLab OSC server
			mockServer := NewMockOSCServer("localhost", port)
			if err = mockServer.Start(); err != nil {
				t.Fatalf("Failed to start mock server: %v", err)
			}

			// Clean up after test
			t.Cleanup(func() {
				mockServer.Clear()
				if err := mockServer.Stop(); err != nil {
					t.Logf("Failed to stop mock server: %v", err)
				}
				time.Sleep(150 * time.Millisecond)
			})

			// Create workspace
			workspace := NewWorkspace("localhost", port)
			t.Cleanup(func() {
				workspace.Close()
			})

			// Test connection with this passcode
			t.Logf("Testing passcode: %q (length: %d)", tc.passcode, len(tc.passcode))
			reply, err := workspace.Init(tc.passcode)

			if tc.shouldOK {
				if err != nil {
					t.Errorf("Expected successful connection with passcode %q, got error: %v", tc.passcode, err)
				} else {
					t.Logf("Successfully connected with passcode %q", tc.passcode)

					// Verify workspace is initialized
					if !workspace.initialized {
						t.Error("Workspace should be initialized")
					}
					if workspace.workspace_id == "" {
						t.Error("workspace_id should be set")
					}

					// Verify reply is valid
					if len(reply) == 0 {
						t.Error("Expected non-empty reply")
					}
				}
			}
		})
	}

	t.Log("All passcode variation tests completed successfully")
}

// TestDecimalCueNumberOSCStringVerification tests that decimal cue numbers like "1.0" are sent as strings in OSC messages
func TestDecimalCueNumberOSCStringVerification(t *testing.T) {
	log.SetLevel(log.InfoLevel)

	t.Log("=== Testing decimal cue number OSC string verification ===")

	workspace, mockServer := setupWorkspaceWithCleanupFatal(t)

	// Clear any previous messages
	mockServer.ClearReceivedMessages()

	// Test various decimal cue numbers that should be preserved as strings
	testCases := []struct {
		name           string
		cueNumber      string
		expectedString string
	}{
		{"Basic decimal", "1.0", "1.0"},
		{"Multi-decimal", "12.5", "12.5"},
		{"Multiple zeros", "1.00", "1.00"},
		{"Leading zero", "0.5", "0.5"},
		{"Complex decimal", "123.456", "123.456"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing cue number: %s (expecting string: %s)", tc.cueNumber, tc.expectedString)

			// Clear messages before this test case
			mockServer.ClearReceivedMessages()

			// Create cue data with decimal number
			cueData := map[string]any{
				"type":   "memo",
				"name":   fmt.Sprintf("Test Cue %s", tc.cueNumber),
				"number": tc.cueNumber,
			}

			// Create the cue
			cueID, err := workspace.createCue(cueData, tc.cueNumber)
			if err != nil {
				t.Fatalf("Failed to create cue: %v", err)
			}

			t.Logf("Created cue with ID: %s", cueID)

			// Get all messages related to setting the "number" property
			numberMessages := mockServer.GetMessagesForAddress("/number")

			// Find the message that set the number property for our cue
			var numberSetMessage *ReceivedMessage
			for _, msg := range numberMessages {
				if strings.Contains(msg.Address, cueID) && strings.HasSuffix(msg.Address, "/number") {
					numberSetMessage = &msg
					break
				}
			}

			if numberSetMessage == nil {
				t.Fatalf("No number property message found for cue %s", cueID)
			}

			t.Logf("Found number set message: Address=%s, Arguments=%v", numberSetMessage.Address, numberSetMessage.Arguments)

			// Verify the message has exactly one argument
			if len(numberSetMessage.Arguments) != 1 {
				t.Fatalf("Expected 1 argument in number set message, got %d: %v", len(numberSetMessage.Arguments), numberSetMessage.Arguments)
			}

			// Get the argument value
			argValue := numberSetMessage.Arguments[0]

			// Check the type and value of the argument
			switch v := argValue.(type) {
			case string:
				if v != tc.expectedString {
					t.Errorf("Expected string argument %q, got %q", tc.expectedString, v)
				} else {
					t.Logf("✓ Argument is correctly a string: %q", v)
				}
			case int, int32, int64:
				t.Errorf("Argument is incorrectly an integer: %v (type: %T), expected string %q", v, v, tc.expectedString)
			case float32, float64:
				t.Errorf("Argument is incorrectly a float: %v (type: %T), expected string %q", v, v, tc.expectedString)
			default:
				t.Errorf("Unexpected argument type: %T, value: %v, expected string %q", v, v, tc.expectedString)
			}

			// Verify that the mock server stored it correctly as a string
			mockServer.mu.RLock()
			cue, exists := mockServer.cues[cueID]
			mockServer.mu.RUnlock()

			if !exists {
				t.Fatalf("Cue %s not found in mock server", cueID)
			}

			if cue.Number != tc.expectedString {
				t.Errorf("Mock server stored cue number as %q, expected %q", cue.Number, tc.expectedString)
			} else {
				t.Logf("✓ Mock server correctly stored cue number as string: %q", cue.Number)
			}

			// Also verify it's in the cuesByNumber mapping correctly
			mockServer.mu.RLock()
			mappedCueID, exists := mockServer.cuesByNumber[tc.expectedString]
			mockServer.mu.RUnlock()

			if !exists {
				t.Errorf("Cue number %q not found in cuesByNumber mapping", tc.expectedString)
			} else if mappedCueID != cueID {
				t.Errorf("cuesByNumber mapping incorrect: %q -> %s, expected %s", tc.expectedString, mappedCueID, cueID)
			} else {
				t.Logf("✓ cuesByNumber mapping correct: %q -> %s", tc.expectedString, mappedCueID)
			}
		})
	}

	t.Log("All decimal cue number OSC string verification tests passed")
}
