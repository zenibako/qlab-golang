package qlab

import (
	"testing"
)

// TestPositionBasedDuplicatePreventionCore tests the core position-based duplicate prevention logic
func TestPositionBasedDuplicatePreventionCore(t *testing.T) {
	workspace := &Workspace{}

	// Test the exact scenario from uber_noir: numberless cues in a playlist
	sourceData := map[string]any{
		"cues": []any{
			map[string]any{
				"name": "Preshow",
				"type": "list",
				"cues": []any{
					map[string]any{
						"name": "",
						"type": "group",
						"cues": []any{
							map[string]any{
								"name": "Espresso - Sabrina Carpenter",
								"type": "audio",
							},
							map[string]any{
								"name": "Pang - Caroline Polachek",
								"type": "audio",
							},
						},
					},
				},
			},
		},
	}

	// Simulate the same structure after first transmission (QLab format)
	currentData := map[string]any{
		"data": map[string]any{
			"cueLists": []any{
				map[string]any{
					"uniqueID": "preshow-list",
					"name":     "Preshow",
					"type":     "cueList",
					"cues": []any{
						map[string]any{
							"uniqueID": "playlist-group",
							"name":     "",
							"type":     "group",
							"cues": []any{
								map[string]any{
									"uniqueID": "espresso-id",
									"name":     "Espresso - Sabrina Carpenter",
									"type":     "audio",
								},
								map[string]any{
									"uniqueID": "pang-id",
									"name":     "Pang - Caroline Polachek",
									"type":     "audio",
								},
							},
						},
					},
				},
			},
		},
	}

	// Index both datasets
	sourceCues := workspace.indexCuesFromWorkspace(sourceData)
	currentCues := workspace.indexCuesFromWorkspace(currentData)

	t.Logf("Source cues: %d, Current cues: %d", len(sourceCues), len(currentCues))

	// Test position-based keys are generated consistently
	expectedKeys := []string{
		"@0[group:]",                             // The playlist group
		"@0[audio:Espresso - Sabrina Carpenter]", // First audio cue
		"@1[audio:Pang - Caroline Polachek]",     // Second audio cue
	}

	for _, key := range expectedKeys {
		sourceCue, sourceExists := sourceCues[key]
		currentCue, currentExists := currentCues[key]

		if !sourceExists {
			t.Errorf("Expected source cue with key %s not found", key)
			continue
		}

		if !currentExists {
			t.Errorf("Expected current cue with key %s not found", key)
			continue
		}

		// Test the comparison
		matches := workspace.compareCueProperties(sourceCue, currentCue)
		if !matches {
			t.Errorf("Position-based cue %s should match between source and current", key)
		} else {
			t.Logf("✓ Position-based cue %s correctly matched", key)
		}
	}

	// Simulate change detection
	creates := 0
	skips := 0

	for sourceKey, sourceCue := range sourceCues {
		if currentCue, exists := currentCues[sourceKey]; exists {
			if workspace.compareCueProperties(sourceCue, currentCue) {
				skips++
				t.Logf("SKIP: %s (matched existing cue)", sourceKey)
			}
		} else {
			creates++
			t.Logf("CREATE: %s (new cue)", sourceKey)
		}
	}

	t.Logf("Summary: %d creates, %d skips", creates, skips)

	// For a proper duplicate prevention test, we should have more skips than creates
	// since the numberless cues should be detected as existing
	if skips == 0 {
		t.Error("Expected at least some skip actions for duplicate prevention")
	}

	// Specifically, the 3 position-based cues should be skipped
	if skips < 3 {
		t.Errorf("Expected at least 3 skips for position-based cues, got %d", skips)
	}

	t.Logf("SUCCESS: Position-based duplicate prevention core logic verified")
}
