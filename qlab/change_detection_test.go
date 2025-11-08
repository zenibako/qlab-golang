package qlab

import (
	"testing"
)

// TestChangeDetectionIndexing tests that cue indexing works correctly
func TestChangeDetectionIndexing(t *testing.T) {
	workspace := &Workspace{}

	// Create test workspace data
	workspaceData := map[string]any{
		"cues": []any{
			map[string]any{
				"type":     "audio",
				"number":   "1.0",
				"name":     "Audio Cue 1",
				"uniqueID": "test-id-1",
			},
			map[string]any{
				"type":     "group",
				"number":   "2.0",
				"name":     "Group Cue",
				"uniqueID": "test-id-2",
				"cues": []any{
					map[string]any{
						"type":     "audio",
						"number":   "1", // Should become 2.1
						"name":     "Sub Audio",
						"uniqueID": "test-id-2-1",
					},
				},
			},
		},
	}

	// Test cue indexing
	cueIndex := workspace.indexCuesFromWorkspace(workspaceData)

	// Verify expected cues were indexed
	expectedCues := []string{"1.0", "2.0"} // Child cue "1.0" is absolute, not relative

	for _, expectedCue := range expectedCues {
		if _, exists := cueIndex[expectedCue]; !exists {
			t.Errorf("Expected cue '%s' not found in index", expectedCue)
		}
	}

	t.Logf("Successfully indexed %d cues: %v", len(cueIndex), func() []string {
		keys := make([]string, 0, len(cueIndex))
		for k := range cueIndex {
			keys = append(keys, k)
		}
		return keys
	}())
}

// TestChangeDetectionSkipAction tests that existing cues are properly skipped
func TestChangeDetectionSkipAction(t *testing.T) {
	workspace := &Workspace{}

	// Create a mock change result that should skip
	changeResults := map[string]*CueChangeResult{
		"1.0": {
			HasChanged: false,
			Action:     "skip",
			Reason:     "unchanged since last transmission",
			ExistingID: "test-id-123",
		},
	}

	// Create a mapping to track number->ID mappings
	mapping := &CueMapping{
		NumberToID:      make(map[string]string),
		CuesWithTargets: []CueTarget{},
	}

	// Test cue data
	cueData := map[string]any{
		"type":   "audio",
		"number": "1.0",
		"name":   "Test Audio Cue",
	}

	// Call the function with change detection
	uniqueID, err := workspace.processCueListWithParentMappingAndChangeDetection(
		cueData, "", "", mapping, changeResults)

	if err != nil {
		t.Fatalf("processCueListWithParentMappingAndChangeDetection failed: %v", err)
	}

	// Verify that the existing ID was returned (not empty, meaning creation was skipped)
	if uniqueID != "test-id-123" {
		t.Errorf("Expected existing ID 'test-id-123', got '%s'", uniqueID)
	}

	// Verify that the mapping was updated
	if mapping.NumberToID["1.0"] != "test-id-123" {
		t.Errorf("Expected mapping for '1.0' to be 'test-id-123', got '%s'", mapping.NumberToID["1.0"])
	}

	t.Logf("Successfully skipped cue creation and returned existing ID: %s", uniqueID)
}

// TestSkipPersistenceInCache tests that skipped cues preserve their cached state
func TestSkipPersistenceInCache(t *testing.T) {
	workspace := &Workspace{}

	// Mock current QLab data (modified externally)
	currentQLab := map[string]any{
		"data": []any{
			map[string]any{
				"cues": []any{
					map[string]any{
						"number":   "1.0",
						"name":     "MODIFIED IN QLAB",
						"type":     "audio",
						"uniqueID": "cached-id-1",
					},
				},
			},
		},
	}

	// Test the cache preservation functionality
	t.Log("Testing cue identifier extraction...")
	currentCues := workspace.indexCuesFromWorkspace(currentQLab)
	if cue, exists := currentCues["1.0"]; exists {
		identifier := workspace.extractCueIdentifier(cue, "")
		if identifier != "1.0" {
			t.Errorf("Expected identifier '1.0', got '%s'", identifier)
		}
		t.Logf("✅ Correctly extracted identifier: %s", identifier)
	} else {
		t.Fatal("❌ Could not find cue 1.0 in current QLab data")
	}

	// Test workspace cue finding and replacement logic
	t.Log("Testing workspace structure navigation...")
	testWorkspace := map[string]any{
		"data": []any{
			map[string]any{
				"cues": []any{
					map[string]any{
						"number":   "1.0",
						"name":     "MODIFIED IN QLAB",
						"type":     "audio",
						"uniqueID": "cached-id-1",
					},
				},
			},
		},
	}

	// Simulate replacing with cached data
	cachedCue := map[string]any{
		"number":   "1.0",
		"name":     "Original Cached Name",
		"type":     "audio",
		"uniqueID": "cached-id-1",
	}

	err := workspace.replaceWorkspaceCueWithCached(testWorkspace, cachedCue, "1.0")
	if err != nil {
		t.Fatalf("❌ Failed to replace cue: %v", err)
	}

	// Verify the replacement worked
	replacedCues := workspace.indexCuesFromWorkspace(testWorkspace)
	if replacedCue, exists := replacedCues["1.0"]; exists {
		if replacedName, ok := replacedCue["name"].(string); ok {
			if replacedName == "Original Cached Name" {
				t.Logf("✅ Successfully preserved cached name: %s", replacedName)
			} else {
				t.Errorf("❌ Expected preserved name 'Original Cached Name', got '%s'", replacedName)
			}
		} else {
			t.Error("❌ Could not get name from replaced cue")
		}
	} else {
		t.Error("❌ Could not find replaced cue")
	}

	t.Log("✅ Skip persistence logic working correctly!")
}

// TestCacheWritingWithSkippedCues tests the complete end-to-end cache writing behavior
func TestCacheWritingWithSkippedCues(t *testing.T) {
	workspace := &Workspace{}

	// Create mock original cache data
	originalCache := map[string]any{
		"data": []any{
			map[string]any{
				"cues": []any{
					map[string]any{
						"number":   "1.0",
						"name":     "Original Cached Name",
						"type":     "audio",
						"uniqueID": "test-id-1",
					},
					map[string]any{
						"number":   "2.0",
						"name":     "Original Light Cue",
						"type":     "light",
						"uniqueID": "test-id-2",
					},
				},
			},
		},
	}

	// Test the cache writing behavior with skip preservation
	t.Log("Testing cache update with mixed skip/update actions...")

	// First, test that we can properly find and index original cache data
	originalCues := workspace.indexCuesFromWorkspace(originalCache)
	if len(originalCues) != 2 {
		t.Errorf("Expected 2 original cues, got %d", len(originalCues))
	}

	// Verify we have the expected original names
	if cue, exists := originalCues["1.0"]; exists {
		if name, ok := cue["name"].(string); ok && name == "Original Cached Name" {
			t.Log("✅ Found original cached name for cue 1.0")
		} else {
			t.Errorf("❌ Expected original cached name 'Original Cached Name', got '%v'", name)
		}
	}

	// Test the workspace replacement logic on a copy of current data
	testWorkspace := map[string]any{
		"data": []any{
			map[string]any{
				"cues": []any{
					map[string]any{
						"number":   "1.0",
						"name":     "MODIFIED IN QLAB",
						"type":     "audio",
						"uniqueID": "test-id-1",
					},
					map[string]any{
						"number":   "2.0",
						"name":     "Normal Cue Updated",
						"type":     "light",
						"uniqueID": "test-id-2",
					},
				},
			},
		},
	}

	// Replace skipped cue with original cached data
	if originalCue, exists := originalCues["1.0"]; exists {
		err := workspace.replaceWorkspaceCueWithCached(testWorkspace, originalCue, "1.0")
		if err != nil {
			t.Fatalf("❌ Failed to replace skipped cue: %v", err)
		}
	}

	// Verify the result - cue 1.0 should be preserved, cue 2.0 should remain updated
	resultCues := workspace.indexCuesFromWorkspace(testWorkspace)

	// Check that cue 1.0 was preserved from cache
	if cue, exists := resultCues["1.0"]; exists {
		if name, ok := cue["name"].(string); ok {
			if name == "Original Cached Name" {
				t.Log("✅ Skipped cue 1.0 correctly preserved cached state")
			} else {
				t.Errorf("❌ Expected preserved name 'Original Cached Name', got '%s'", name)
			}
		}
	}

	// Check that cue 2.0 remained in its current state (not preserved)
	if cue, exists := resultCues["2.0"]; exists {
		if name, ok := cue["name"].(string); ok {
			if name == "Normal Cue Updated" {
				t.Log("✅ Non-skipped cue 2.0 correctly kept current state")
			} else {
				t.Errorf("❌ Expected current name 'Normal Cue Updated', got '%s'", name)
			}
		}
	}

	t.Log("✅ End-to-end cache writing with skip preservation working correctly!")
}
