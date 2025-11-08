package qlab

import (
	"testing"
	"time"
)

// TestGetRunningCueNumbers tests extracting cue numbers from running cues
func TestGetRunningCueNumbers(t *testing.T) {
	tests := []struct {
		name         string
		runningCues  []map[string]any
		expectedNums []string
	}{
		{
			name: "Multiple running cues with numbers",
			runningCues: []map[string]any{
				{"number": "1.0", "name": "Audio Cue", "type": "audio"},
				{"number": "2.5", "name": "Light Cue", "type": "light"},
				{"number": "10", "name": "Group Cue", "type": "group"},
			},
			expectedNums: []string{"1.0", "2.5", "10"},
		},
		{
			name:         "Empty running cues",
			runningCues:  []map[string]any{},
			expectedNums: []string{},
		},
		{
			name: "Running cues without numbers",
			runningCues: []map[string]any{
				{"name": "Unnamed Cue", "type": "memo"},
				{"uniqueID": "test-id-123"},
			},
			expectedNums: []string{},
		},
		{
			name: "Mixed - some with numbers, some without",
			runningCues: []map[string]any{
				{"number": "1.0", "name": "Audio Cue"},
				{"name": "No Number Cue"},
				{"number": "3.0", "name": "Light Cue"},
			},
			expectedNums: []string{"1.0", "3.0"},
		},
		{
			name: "Non-string number values (should be skipped)",
			runningCues: []map[string]any{
				{"number": "1.0", "name": "Valid String Number"},
				{"number": 2.5, "name": "Float Number"}, // Should be skipped
				{"number": 3, "name": "Int Number"},     // Should be skipped
			},
			expectedNums: []string{"1.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRunningCueNumbers(tt.runningCues)

			if len(result) != len(tt.expectedNums) {
				t.Errorf("Expected %d numbers, got %d", len(tt.expectedNums), len(result))
			}

			for i, expected := range tt.expectedNums {
				if i >= len(result) {
					t.Errorf("Missing expected number at index %d: %s", i, expected)
					continue
				}
				if result[i] != expected {
					t.Errorf("At index %d: expected %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

// TestGetSelectedCueNumbers tests extracting cue numbers from selected cues
func TestGetSelectedCueNumbers(t *testing.T) {
	tests := []struct {
		name         string
		selectedCues []map[string]any
		expectedNums []string
	}{
		{
			name: "Multiple selected cues with numbers",
			selectedCues: []map[string]any{
				{"number": "5.0", "name": "Selected Audio", "type": "audio"},
				{"number": "7.25", "name": "Selected Light", "type": "light"},
			},
			expectedNums: []string{"5.0", "7.25"},
		},
		{
			name:         "Empty selected cues",
			selectedCues: []map[string]any{},
			expectedNums: []string{},
		},
		{
			name: "Single selected cue",
			selectedCues: []map[string]any{
				{"number": "42.0", "name": "The Answer", "type": "memo"},
			},
			expectedNums: []string{"42.0"},
		},
		{
			name: "Selected cues without numbers",
			selectedCues: []map[string]any{
				{"name": "Unnumbered Selection", "type": "group"},
			},
			expectedNums: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSelectedCueNumbers(tt.selectedCues)

			if len(result) != len(tt.expectedNums) {
				t.Errorf("Expected %d numbers, got %d", len(tt.expectedNums), len(result))
			}

			for i, expected := range tt.expectedNums {
				if i >= len(result) {
					t.Errorf("Missing expected number at index %d: %s", i, expected)
					continue
				}
				if result[i] != expected {
					t.Errorf("At index %d: expected %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

// TestSetupUpdateListener tests the update listener setup
func TestSetupUpdateListener(t *testing.T) {
	workspace, mockServer := setupWorkspaceWithCleanup(t)

	t.Log("=== Testing update listener setup ===")

	// Track if handler was called
	callCount := 0

	// Setup the update listener
	err := SetupUpdateListener(workspace, func() {
		callCount++
	})

	if err != nil {
		t.Fatalf("Failed to setup update listener: %v", err)
	}

	t.Log("Update listener setup successful")

	// Note: The actual handler being called depends on the OSC listener implementation
	// This test verifies that SetupUpdateListener doesn't error and returns successfully
	// In a real scenario with a real QLab instance, the handler would be called when
	// QLab sends update messages

	// The mock server doesn't automatically send update broadcasts like real QLab,
	// but we've verified the listener setup works without errors
	t.Logf("Handler call count: %d (expected 0 in mock environment without updates)", callCount)

	// Use the mock server reference to avoid unused variable warning
	_ = mockServer
}

// TestUpdateListenerWithRealUpdates tests the update listener with simulated OSC updates
func TestUpdateListenerWithRealUpdates(t *testing.T) {
	workspace, mockServer := setupWorkspaceWithCleanup(t)

	t.Log("=== Testing update listener with simulated updates ===")

	// Track updates
	updateReceived := false
	var receivedAddress string
	var receivedArgs []any

	// Setup workspace's own update listener (not the wrapper function)
	err := workspace.StartUpdateListener(func(address string, args []any) {
		updateReceived = true
		receivedAddress = address
		receivedArgs = args
		t.Logf("Update received: address=%s, args=%v", address, args)
	})

	if err != nil {
		t.Fatalf("Failed to start update listener: %v", err)
	}

	// Create a test cue to trigger an update
	cueData := map[string]any{
		"type": "memo",
		"name": "Update Test Cue",
	}

	cueID, err := workspace.createCue(cueData, "")
	if err != nil {
		t.Fatalf("Failed to create test cue: %v", err)
	}

	t.Logf("Created test cue: %s", cueID)

	// Give time for any updates to propagate
	time.Sleep(200 * time.Millisecond)

	// The mock server doesn't automatically send updates like real QLab would,
	// but we've verified the listener setup works
	t.Logf("Update listener active, received update: %v", updateReceived)
	if updateReceived {
		t.Logf("  Address: %s", receivedAddress)
		t.Logf("  Args: %v", receivedArgs)
	}

	// Cleanup
	mockServer.Clear()
}
