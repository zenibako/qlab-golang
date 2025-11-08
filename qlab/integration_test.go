package qlab

import (
	"testing"
)

// TestChangeDetectionDuplicatePrevention tests the complete flow of change detection and duplicate prevention
func TestChangeDetectionDuplicatePrevention(t *testing.T) {
	// Create a mock workspace with existing data
	mockWorkspace := map[string]any{
		"data": map[string]any{
			"cueLists": []any{
				map[string]any{
					"uniqueID": "list-1",
					"name":     "Main Cue List",
					"type":     "cueList",
					"cues": []any{
						map[string]any{
							"uniqueID": "existing-cue-1",
							"number":   "1.0",
							"name":     "Existing Audio",
							"type":     "audio",
						},
						map[string]any{
							"uniqueID": "existing-cue-2",
							"name":     "Existing Memo",
							"type":     "memo",
						},
					},
				},
			},
		},
	}

	// Create source data that matches existing cues
	sourceWorkspace := map[string]any{
		"cues": []any{
			map[string]any{
				"number": "1.0",
				"name":   "Existing Audio",
				"type":   "audio",
			},
			map[string]any{
				"name": "Existing Memo",
				"type": "memo",
			},
		},
	}

	workspace := &Workspace{}

	// Test indexing functions directly (bypass OSC calls)
	// This tests the core logic without requiring a real QLab connection
	sourceCues := workspace.indexCuesFromWorkspace(sourceWorkspace)
	currentCues := workspace.indexCuesFromWorkspace(mockWorkspace)

	t.Logf("Source cues indexed: %d", len(sourceCues))
	t.Logf("Current cues indexed: %d", len(currentCues))

	// Test change detection logic manually
	changeResults := make(map[string]*CueChangeResult)

	// Compare source cues against current cues
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

	t.Logf("Change detection results: %d", len(changeResults))

	// Print all change detection results
	for key, result := range changeResults {
		t.Logf("Change result for key '%s': action=%s, reason=%s, existing_id=%s",
			key, result.Action, result.Reason, result.ExistingID)
	}

	// Verify that we have results for both cues
	if len(changeResults) == 0 {
		t.Fatal("No change detection results generated")
	}

	// Test that we get "skip" actions for existing cues that match
	foundSkip := false
	for _, result := range changeResults {
		if result.Action == "skip" {
			foundSkip = true
			if result.ExistingID == "" {
				t.Error("Skip action should have an existing ID")
			}
			t.Logf("Found skip action with existing ID: %s", result.ExistingID)
		}
	}

	if !foundSkip {
		t.Error("Expected at least one 'skip' action for unchanged cues, but found none")
	}
}

// TestThreeWayComparisonLogic tests the specific logic of the three-way comparison
func TestThreeWayComparisonLogic(t *testing.T) {
	workspace := &Workspace{}

	// Test case 1: Source matches current QLab (no cache)
	sourceCue := map[string]any{
		"number": "1.0",
		"name":   "Test Cue",
		"type":   "audio",
	}

	currentCue := map[string]any{
		"uniqueID": "existing-123",
		"number":   "1.0",
		"name":     "Test Cue",
		"type":     "audio",
	}

	result := workspace.compareCueProperties(sourceCue, currentCue)
	t.Logf("Source matches current: %v", result)

	// Test the indexing function
	sourceData := map[string]any{
		"cues": []any{sourceCue},
	}
	currentData := map[string]any{
		"data": map[string]any{
			"cueLists": []any{
				map[string]any{
					"cues": []any{currentCue},
				},
			},
		},
	}

	sourceCues := workspace.indexCuesFromWorkspace(sourceData)
	currentCues := workspace.indexCuesFromWorkspace(currentData)

	t.Logf("Source cues indexed: %v", len(sourceCues))
	t.Logf("Current cues indexed: %v", len(currentCues))

	for key := range sourceCues {
		t.Logf("Source cue key: %s", key)
	}
	for key := range currentCues {
		t.Logf("Current cue key: %s", key)
	}
}
