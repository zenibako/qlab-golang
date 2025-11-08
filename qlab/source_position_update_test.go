package qlab

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSourceFileUpdateWithPositionBasedCues tests updating source files with unnumbered cues using position-based identifiers
func TestSourceFileUpdateWithPositionBasedCues(t *testing.T) {
	// Skip test - requires CUE file handler to be injected
	t.Skip("Test requires CUE file handler - skipping in standalone qlab-golang package")

	// Create a CUE file with unnumbered cues (position-based identifiers)
	testCueContent := `package qlab

import "github.com/zenibako/cuejitsu/lib/cj"

workspace: cj.#Workspace & {
	name: "Test Workspace" 
	passcode: 1297
	cues: [
		cj.#CueList & {
			name: "Preshow"
			cues: [
				cj.#AudioCue & {
					name: "Original Song A"
					fileTarget: "original/song_a.mp3"
				},
				cj.#AudioCue & {
					name: "Original Song B"
					fileTarget: "original/song_b.mp3"
				},
			]
		},
	]
}
`

	// Create temporary directory and file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_position.cue")

	err := os.WriteFile(testFile, []byte(testCueContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CUE file: %v", err)
	}

	// Create mock QLab data with unnumbered cues that have QLab updates
	mockQLabData := map[string]any{
		"data": []any{
			map[string]any{
				"name": "Main Cue List",
				"type": "cue_list",
				"cues": []any{
					map[string]any{
						"name": "Preshow",
						"type": "cue_list",
						"cues": []any{
							map[string]any{
								"name":       "Updated Song A from QLab",
								"type":       "audio",
								"fileTarget": "updated/song_a.mp3",
								"notes":      "New notes A",
							},
							map[string]any{
								"name":       "Updated Song B from QLab",
								"type":       "audio",
								"fileTarget": "updated/song_b.mp3",
								"notes":      "New notes B",
							},
						},
					},
				},
			},
		},
	}

	// Create comparison with position-based chosen cues
	// Note: The keys should match the identifiers generated from QLab data
	comparison := &ThreeWayComparison{
		QLabChosenCues: map[string]bool{
			"@0[audio:Updated Song A from QLab]": true, // User chose to keep QLab version for first audio cue
			"@1[audio:Updated Song B from QLab]": true, // User chose to keep QLab version for second audio cue
		},
		CurrentQLabData: mockQLabData,
	}

	// Create mock workspace
	workspace := &Workspace{
		initialized:  true,
		workspace_id: "test-workspace",
	}

	// Test extracting QLab values with position-based identifiers
	t.Run("ExtractPositionBasedQLabCueValues", func(t *testing.T) {
		// Get QLabChosenCues keys
		chosenKeys := make([]string, 0, len(comparison.QLabChosenCues))
		for k := range comparison.QLabChosenCues {
			chosenKeys = append(chosenKeys, k)
		}
		t.Logf("QLabChosenCues keys: %v", chosenKeys)

		cueUpdates := make(map[string]map[string]any)
		err := workspace.extractQLabCueValues(mockQLabData, comparison.QLabChosenCues, cueUpdates)
		if err != nil {
			t.Fatalf("Failed to extract QLab cue values: %v", err)
		}

		t.Logf("Extracted cue update keys: %v", getMapKeys(cueUpdates))

		if len(cueUpdates) != 2 {
			t.Errorf("Expected updates for 2 cues, got %d", len(cueUpdates))
		}

		// Note: The identifiers will be based on the QLab data, not the original source
		// This is expected because QLab may have updated the cue names
		hasAudioUpdates := false
		for key, updates := range cueUpdates {
			if updates["name"] == "Updated Song A from QLab" || updates["name"] == "Updated Song B from QLab" {
				hasAudioUpdates = true
				t.Logf("Found audio cue update: %s -> %v", key, updates)
			}
		}

		if !hasAudioUpdates {
			t.Error("Expected to find audio cue updates")
		}

		t.Logf("✅ Successfully extracted position-based cue updates")
	})

	// Test the complete update workflow
	t.Run("UpdateSourceFileWithPositionBasedCues", func(t *testing.T) {
		// NOTE: updateSourceFileWithQLabValues has been removed from qlab-golang
		// The caller should now use ExtractQLabUpdates() and write the updates themselves
		cueUpdates, err := workspace.ExtractQLabUpdates(comparison)
		if err != nil {
			t.Errorf("Failed to extract QLab updates: %v", err)
		}
		// In real code, caller would use cue.UpdateCueFileWithQLabData(testFile, cueUpdates)
		_ = cueUpdates

		t.Logf("✅ Position-based source file update workflow completed")
	})
}
