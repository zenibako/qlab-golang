package qlab

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRealWorldConflictResolution tests the complete workflow using real QLab data structure from debug output
func TestRealWorldConflictResolution(t *testing.T) {
	// Skip test - requires CUE file handler to be injected
	t.Skip("Test requires CUE file handler - skipping in standalone qlab-golang package")

	// Create a test CUE file that matches our real scenario
	testCueContent := `package qlab

import "github.com/zenibako/cuejitsu/lib/cj"

workspace: cj.#Workspace & {
	name:     "Test"
	passcode: ""
	cues: [
		cj.#Timeline & {
			number: "0"
			name:   "Source Changed Name"
			cues: []
		},
	]
}
`

	// Create temporary directory and file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_real_world.cue")

	err := os.WriteFile(testFile, []byte(testCueContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CUE file: %v", err)
	}

	// Print original content
	originalContent, _ := os.ReadFile(testFile)
	t.Logf("Original file content:\n%s", string(originalContent))

	// Create mock QLab data that matches what we see in debug output - this structure from real QLab
	mockQLabData := map[string]any{
		"address":      "/workspace/C03DF556-B03A-4D3A-90C7-8E77C0B1F36E/cueLists",
		"workspace_id": "C03DF556-B03A-4D3A-90C7-8E77C0B1F36E",
		"status":       "ok",
		"data": []any{
			map[string]any{
				"listName":       "Main Cue List",
				"name":           "Main Cue List",
				"type":           "cue_list",
				"armed":          false,
				"uniqueID":       "main-cue-list-id",
				"colorName":      "",
				"colorName/live": "",
				"flagged":        false,
				"number":         "",
				"cues":           []any{}, // Empty main cue list
			},
			map[string]any{
				"listName":       "Cuejitsu Inbox",
				"name":           "Cuejitsu Inbox",
				"type":           "cue_list",
				"armed":          false,
				"uniqueID":       "BC55B39A-A211-4199-9FDA-F3E983095143",
				"colorName":      "",
				"colorName/live": "",
				"flagged":        false,
				"number":         "",
				"cues": []any{
					// This is the cue that has the conflict
					map[string]any{
						"armed":          false,
						"colorName":      "",
						"colorName/live": "",
						"cues":           []any{},
						"flagged":        false,
						"listName":       "Cuejitsu Inbox",
						"name":           "QLab Changed Name", // This is the QLab version
						"number":         "0",
						"type":           "group",
						"uniqueID":       "24CB6DBB-A4DE-4A69-BC1E-03143A9B5976",
					},
				},
			},
		},
	}

	// Create comparison with QLab chosen cues (simulating user selecting "Keep QLab version")
	comparison := &ThreeWayComparison{
		QLabChosenCues: map[string]bool{
			"0": true, // User chose to keep QLab version for cue 0
		},
		CurrentQLabData: mockQLabData,
	}

	// Create mock workspace
	workspace := &Workspace{
		initialized:  true,
		workspace_id: "C03DF556-B03A-4D3A-90C7-8E77C0B1F36E",
	}

	// Test extraction of QLab cue values
	t.Run("ExtractQLabCueValues", func(t *testing.T) {
		cueUpdates := make(map[string]map[string]any)
		err := workspace.extractQLabCueValues(mockQLabData, comparison.QLabChosenCues, cueUpdates)
		if err != nil {
			t.Fatalf("Failed to extract QLab cue values: %v", err)
		}

		t.Logf("Extracted cue updates: %+v", cueUpdates)

		// Verify that we extracted updates for cue 0
		if len(cueUpdates) == 0 {
			t.Error("Expected at least one cue update to be extracted")
			return
		}

		// Check cue 0 updates
		if updates, exists := cueUpdates["0"]; exists {
			if updates["name"] != "QLab Changed Name" {
				t.Errorf("Expected name 'QLab Changed Name', got %v", updates["name"])
			}
			t.Logf("✅ Successfully extracted name update: %v", updates["name"])
		} else {
			t.Error("Missing updates for cue 0")
		}
	})

	// Test the complete update workflow
	t.Run("UpdateSourceFileWithQLabValues", func(t *testing.T) {
		// Test the complete update workflow
		// NOTE: updateSourceFileWithQLabValues has been removed from qlab-golang
		// The caller should now use ExtractQLabUpdates() and write the updates themselves
		cueUpdates, err := workspace.ExtractQLabUpdates(comparison)
		if err != nil {
			t.Errorf("Failed to extract QLab updates: %v", err)
		}
		// In real code, caller would use cue.UpdateCueFileWithQLabData(testFile, cueUpdates)
		_ = cueUpdates

		// Read updated file content
		updatedContent, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		t.Logf("Updated file content:\n%s", string(updatedContent))

		// Check if the file actually changed
		if string(originalContent) == string(updatedContent) {
			t.Errorf("⚠️  File content is identical - source file was not updated")
		} else {
			t.Logf("✅ File content changed - source file was updated")
		}

		// Verify the specific change we expect
		updatedContentStr := string(updatedContent)
		if !strings.Contains(updatedContentStr, "QLab Changed Name") {
			t.Errorf("Expected updated content to contain 'QLab Changed Name', but it does not")
		} else {
			t.Logf("✅ Source file now contains QLab name: 'QLab Changed Name'")
		}

		// Verify it no longer contains the old name
		if strings.Contains(updatedContentStr, "Source Changed Name") {
			t.Errorf("Updated content still contains old name 'Source Changed Name'")
		} else {
			t.Logf("✅ Source file no longer contains old name: 'Source Changed Name'")
		}
	})
}
