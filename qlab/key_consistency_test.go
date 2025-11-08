package qlab

import (
	"testing"
)

// TestCueNumberFormatting tests that cue numbers are formatted consistently
func TestCueNumberFormatting(t *testing.T) {
	workspace := &Workspace{}

	// Test workspace data with various number formats
	workspaceData := map[string]any{
		"cues": []any{
			map[string]any{
				"type":     "audio",
				"number":   1.0, // float64 should become "1.0"
				"name":     "Float Cue",
				"uniqueID": "test-id-1",
			},
			map[string]any{
				"type":     "audio",
				"number":   "2.0", // string should stay "2.0"
				"name":     "String Cue",
				"uniqueID": "test-id-2",
			},
			map[string]any{
				"type":     "group",
				"number":   3, // int should become "3"
				"name":     "Int Cue",
				"uniqueID": "test-id-3",
			},
		},
	}

	// Test that indexing formats numbers consistently
	cueIndex := workspace.indexCuesFromWorkspace(workspaceData)

	expectedKeys := []string{"1.0", "2.0", "3"}
	for _, key := range expectedKeys {
		if _, exists := cueIndex[key]; !exists {
			t.Errorf("Expected formatted cue number '%s' not found in index", key)
		}
	}

	t.Logf("Successfully formatted cue numbers: %v", func() []string {
		keys := make([]string, 0, len(cueIndex))
		for k := range cueIndex {
			keys = append(keys, k)
		}
		return keys
	}())
}

// TestCueComparisonLogic tests the cue property comparison logic
func TestCueComparisonLogic(t *testing.T) {
	workspace := &Workspace{}

	testCases := []struct {
		name     string
		cue1     map[string]any
		cue2     map[string]any
		expected bool
	}{
		{
			name:     "identical cues",
			cue1:     map[string]any{"number": "1.0", "name": "Test", "type": "audio"},
			cue2:     map[string]any{"number": "1.0", "name": "Test", "type": "audio"},
			expected: true,
		},
		{
			name:     "different names",
			cue1:     map[string]any{"number": "1.0", "name": "Test1", "type": "audio"},
			cue2:     map[string]any{"number": "1.0", "name": "Test2", "type": "audio"},
			expected: false,
		},
		{
			name:     "different types",
			cue1:     map[string]any{"number": "1.0", "name": "Test", "type": "audio"},
			cue2:     map[string]any{"number": "1.0", "name": "Test", "type": "memo"},
			expected: false,
		},
		{
			name:     "with extra uniqueID in second",
			cue1:     map[string]any{"number": "1.0", "name": "Test", "type": "audio"},
			cue2:     map[string]any{"number": "1.0", "name": "Test", "type": "audio", "uniqueID": "abc123"},
			expected: true, // uniqueID should be ignored in comparison
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := workspace.compareCueProperties(tc.cue1, tc.cue2)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for cues: %v vs %v", tc.expected, result, tc.cue1, tc.cue2)
			}
		})
	}
}

// TestChangeDetectionResultMapping tests that change detection results are properly mapped
func TestChangeDetectionResultMapping(t *testing.T) {
	// Create mock change results
	changeResults := map[string]*CueChangeResult{
		"1.0": {
			HasChanged: false,
			Action:     "skip",
			Reason:     "unchanged",
			ExistingID: "existing-123",
		},
		"Test Memo@1": {
			HasChanged: false,
			Action:     "skip",
			Reason:     "unchanged",
			ExistingID: "existing-456",
		},
	}

	// Test lookup scenarios
	testCases := []struct {
		fullNumber     string
		indexKey       string
		shouldFind     bool
		expectedID     string
		expectedAction string
	}{
		{
			fullNumber:     "1.0",
			indexKey:       "1.0",
			shouldFind:     true,
			expectedID:     "existing-123",
			expectedAction: "skip",
		},
		{
			fullNumber:     "",
			indexKey:       "Test Memo@1",
			shouldFind:     true,
			expectedID:     "existing-456",
			expectedAction: "skip",
		},
		{
			fullNumber:     "2.0",
			indexKey:       "2.0",
			shouldFind:     false,
			expectedID:     "",
			expectedAction: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.fullNumber+"_"+tc.indexKey, func(t *testing.T) {
			// Simulate the lookup logic from processCueListWithParentMappingAndChangeDetection
			lookupKey := tc.indexKey
			if lookupKey == "" {
				lookupKey = tc.fullNumber
			}

			if changeResult, exists := changeResults[lookupKey]; exists && lookupKey != "" {
				if !tc.shouldFind {
					t.Errorf("Found result when shouldn't have: key=%s", lookupKey)
					return
				}
				if changeResult.ExistingID != tc.expectedID {
					t.Errorf("Expected ID %s, got %s", tc.expectedID, changeResult.ExistingID)
				}
				if changeResult.Action != tc.expectedAction {
					t.Errorf("Expected action %s, got %s", tc.expectedAction, changeResult.Action)
				}
			} else {
				if tc.shouldFind {
					t.Errorf("Should have found result for key: %s", lookupKey)
				}
			}
		})
	}
}
