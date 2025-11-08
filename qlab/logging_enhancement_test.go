package qlab

import (
	"testing"
)

func TestDetailedCueComparison(t *testing.T) {
	workspace := &Workspace{}

	// Create test cues with differences
	sourceCue := map[string]any{
		"number":   "1",
		"name":     "Test Cue Updated",
		"type":     "Audio",
		"duration": "5.0",
		"notes":    "Updated notes",
		"uniqueID": "source-id-123",
	}

	currentCue := map[string]any{
		"number":   "1",
		"name":     "Test Cue Original",
		"type":     "Audio",
		"duration": "3.0",
		"notes":    "Original notes",
		"uniqueID": "qlab-id-456",
	}

	// Test the detailed comparison function
	t.Run("DetailedComparison", func(t *testing.T) {
		differences := workspace.compareCuePropertiesDetailed(sourceCue, currentCue)

		// Should detect differences in name, duration, and notes
		if len(differences) == 0 {
			t.Error("Expected differences to be detected")
		}

		expectedDiffs := []string{"name", "duration", "notes"}
		for _, field := range expectedDiffs {
			if _, exists := differences[field]; !exists {
				t.Errorf("Expected difference in field '%s' to be detected", field)
			}
		}

		// Check that the difference format is correct
		if nameDiff, exists := differences["name"]; exists {
			expectedFormat := "'Test Cue Updated' -> 'Test Cue Original'"
			if nameDiff != expectedFormat {
				t.Errorf("Expected name difference format '%s', got '%s'", expectedFormat, nameDiff)
			}
		}
	})

	// Test that identical cues have no differences
	t.Run("IdenticalCues", func(t *testing.T) {
		identicalCue := map[string]any{
			"number": "1",
			"name":   "Test Cue",
			"type":   "Audio",
		}

		differences := workspace.compareCuePropertiesDetailed(identicalCue, identicalCue)
		if len(differences) != 0 {
			t.Errorf("Expected no differences for identical cues, got %d differences", len(differences))
		}
	})
}

func TestCompareCuePropertiesBackwardCompatibility(t *testing.T) {
	workspace := &Workspace{}

	// Test that the boolean version still works correctly
	cue1 := map[string]any{
		"number": "1",
		"name":   "Test Cue",
		"type":   "Audio",
	}

	cue2 := map[string]any{
		"number": "1",
		"name":   "Test Cue",
		"type":   "Audio",
	}

	cue3 := map[string]any{
		"number": "1",
		"name":   "Different Cue",
		"type":   "Audio",
	}

	// Identical cues should match
	if !workspace.compareCueProperties(cue1, cue2) {
		t.Error("Expected identical cues to match")
	}

	// Different cues should not match
	if workspace.compareCueProperties(cue1, cue3) {
		t.Error("Expected different cues to not match")
	}
}

func TestCueChangeResultStructure(t *testing.T) {
	// Test that the new CueChangeResult fields work correctly
	result := &CueChangeResult{
		HasChanged: true,
		Reason:     "test reason",
		ExistingID: "existing-123",
		Action:     "update",
		ModifiedFields: map[string]string{
			"name":     "'Old Name' -> 'New Name'",
			"duration": "'3.0' -> '5.0'",
		},
		CueID: "cue-id-456",
	}

	// Verify all fields are accessible
	if !result.HasChanged {
		t.Error("Expected HasChanged to be true")
	}

	if result.CueID != "cue-id-456" {
		t.Errorf("Expected CueID 'cue-id-456', got '%s'", result.CueID)
	}

	if len(result.ModifiedFields) != 2 {
		t.Errorf("Expected 2 modified fields, got %d", len(result.ModifiedFields))
	}

	if result.ModifiedFields["name"] != "'Old Name' -> 'New Name'" {
		t.Errorf("Expected correct name change format, got '%s'", result.ModifiedFields["name"])
	}
}
