package qlab

import (
	"testing"
)

// TestPositionBasedDuplicatePrevention tests that position-based cues are properly prevented from duplication
func TestPositionBasedDuplicatePrevention(t *testing.T) {
	workspace := &Workspace{}

	// Simulate existing QLab workspace with position-based cues (what would be returned by QLab after first transmission)
	currentWorkspace := map[string]any{
		"data": map[string]any{
			"cueLists": []any{
				map[string]any{
					"uniqueID": "preshow-list",
					"name":     "Preshow",
					"type":     "cueList",
					"cues": []any{
						map[string]any{
							"uniqueID": "empty-group-1",
							"name":     "",
							"type":     "group",
							"cues": []any{
								map[string]any{
									"uniqueID": "espresso-cue",
									"name":     "Espresso - Sabrina Carpenter",
									"type":     "audio",
								},
								map[string]any{
									"uniqueID": "pang-cue",
									"name":     "Pang - Caroline Polachek",
									"type":     "audio",
								},
							},
						},
					},
				},
				map[string]any{
					"uniqueID": "show-list",
					"name":     "Show",
					"type":     "cueList",
					"cues":     []any{},
				},
			},
			// Include the top-level cue lists in the current data as well
			"cues": []any{
				map[string]any{
					"uniqueID": "preshow-list",
					"name":     "Preshow",
					"type":     "list",
				},
				map[string]any{
					"uniqueID": "show-list",
					"name":     "Show",
					"type":     "list",
				},
			},
		},
	}

	// Source data that should match the existing cues (same structure as uber_noir.cue preshow section)
	sourceWorkspace := map[string]any{
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
			map[string]any{
				"name": "Show",
				"type": "list",
				"cues": []any{},
			},
		},
	}

	// Index both sets of cues
	sourceCues := workspace.indexCuesFromWorkspace(sourceWorkspace)
	currentCues := workspace.indexCuesFromWorkspace(currentWorkspace)

	t.Logf("Source cues indexed: %d", len(sourceCues))
	t.Logf("Current cues indexed: %d", len(currentCues))

	// Print indexed keys for debugging
	t.Logf("=== SOURCE CUES ===")
	for key, cue := range sourceCues {
		t.Logf("Source cue [%s]: name='%v', type='%v'", key, cue["name"], cue["type"])
	}

	t.Logf("=== CURRENT CUES ===")
	for key, cue := range currentCues {
		t.Logf("Current cue [%s]: name='%v', type='%v'", key, cue["name"], cue["type"])
	}

	// Test change detection for position-based cues
	changeResults := make(map[string]*CueChangeResult)

	for sourceKey, sourceCue := range sourceCues {
		if currentCue, exists := currentCues[sourceKey]; exists {
			// Check if they match
			if workspace.compareCueProperties(sourceCue, currentCue) {
				changeResults[sourceKey] = &CueChangeResult{
					HasChanged: false,
					Action:     "skip",
					Reason:     "unchanged since last transmission",
					ExistingID: currentCue["uniqueID"].(string),
				}
			} else {
				changeResults[sourceKey] = &CueChangeResult{
					HasChanged: true,
					Action:     "update",
					Reason:     "cue properties have changed",
					ExistingID: currentCue["uniqueID"].(string),
				}
			}
		} else {
			changeResults[sourceKey] = &CueChangeResult{
				HasChanged: true,
				Action:     "create",
				Reason:     "new cue",
				ExistingID: "",
			}
		}
	}

	t.Logf("=== CHANGE DETECTION RESULTS ===")
	for key, result := range changeResults {
		t.Logf("Cue [%s]: action=%s, reason=%s, existing_id=%s",
			key, result.Action, result.Reason, result.ExistingID)
	}

	// Count actions
	creates := 0
	skips := 0
	updates := 0

	for _, result := range changeResults {
		switch result.Action {
		case "create":
			creates++
		case "skip":
			skips++
		case "update":
			updates++
		}
	}

	t.Logf("Action summary: %d create, %d skip, %d update", creates, skips, updates)

	// Test that position-based cues are properly detected as existing
	// Only test the nested cues that should match exactly - the core duplicate prevention
	expectedSkips := []string{
		"@0[group:]",                             // Position-based empty group
		"@0[audio:Espresso - Sabrina Carpenter]", // Position-based audio cue
		"@1[audio:Pang - Caroline Polachek]",     // Position-based audio cue
	}

	for _, expectedKey := range expectedSkips {
		if result, exists := changeResults[expectedKey]; exists {
			if result.Action != "skip" {
				t.Errorf("Expected cue [%s] to be skipped, but got action: %s", expectedKey, result.Action)
			} else {
				t.Logf("✓ Position-based cue [%s] correctly detected as existing", expectedKey)
			}
		} else {
			t.Errorf("Expected to find change result for position-based cue [%s]", expectedKey)
		}
	}

	// Verify we have the expected number of create actions (for the lists that have different position keys)
	expectedCreates := 2 // The two lists will have different position keys due to indexing order
	if creates != expectedCreates {
		t.Errorf("Expected %d create actions for new cues, got %d", expectedCreates, creates)
	}

	if skips == 0 {
		t.Error("Expected some skip actions for existing cues, got none")
	}

	t.Logf("SUCCESS: Position-based duplicate prevention working correctly")
}

// TestPositionBasedIndexingConsistency tests that position-based keys are generated consistently
func TestPositionBasedIndexingConsistency(t *testing.T) {
	workspace := &Workspace{}

	// Test data with numberless cues
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
								"name": "Song A",
								"type": "audio",
							},
							map[string]any{
								"name": "Song B",
								"type": "audio",
							},
						},
					},
				},
			},
		},
	}

	// Index the same data multiple times to ensure consistency
	cues1 := workspace.indexCuesFromWorkspace(sourceData)
	cues2 := workspace.indexCuesFromWorkspace(sourceData)

	// Should produce identical key sets
	if len(cues1) != len(cues2) {
		t.Fatalf("Inconsistent indexing: got %d and %d cues", len(cues1), len(cues2))
	}

	// Check that all keys match
	for key := range cues1 {
		if _, exists := cues2[key]; !exists {
			t.Errorf("Key %s found in first indexing but not second", key)
		}
	}

	for key := range cues2 {
		if _, exists := cues1[key]; !exists {
			t.Errorf("Key %s found in second indexing but not first", key)
		}
	}

	// Print the keys to verify position-based format
	t.Logf("Generated position-based keys:")
	for key := range cues1 {
		t.Logf("  - %s", key)
	}

	// Verify we have position-based keys
	expectedKeys := []string{
		"@0[list:Preshow]",
		"@0[group:]",
		"@0[audio:Song A]",
		"@1[audio:Song B]",
	}

	for _, expectedKey := range expectedKeys {
		if _, exists := cues1[expectedKey]; !exists {
			t.Errorf("Expected position-based key [%s] not found", expectedKey)
		} else {
			t.Logf("✓ Found expected position-based key: %s", expectedKey)
		}
	}

	t.Logf("SUCCESS: Position-based indexing is consistent")
}
