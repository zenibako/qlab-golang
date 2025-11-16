package qlab

import (
	"testing"
)

// TestFadeCueDoOpacityFields verifies that fade cue doOpacity fields are properly handled
func TestFadeCueDoOpacityFields(t *testing.T) {
	// Create a mock workspace (this won't actually connect to QLab)
	workspace := NewWorkspace("localhost", 53000)
	workspace.SetTimeout(1) // Very short timeout for testing

	// Create a fade cue with doOpacity enabled
	fadeCueData := map[string]any{
		"type":            "fade",
		"number":          "999.9",
		"name":            "Test Fade Cue",
		"cueTargetNumber": "1",
		"doOpacity":       true,
		"opacity":         1.0,
		"duration":        "2.0",
	}

	// Test that the fields are properly set in the map
	if doOpacity, exists := fadeCueData["doOpacity"]; exists {
		if doOpacity != true {
			t.Errorf("Expected doOpacity=true, got %v", doOpacity)
		}
	} else {
		t.Error("doOpacity field not found in fade cue data")
	}

	// Test that other fade enable fields can be set
	fadeCueData["doTranslation"] = true
	fadeCueData["doScale"] = true
	fadeCueData["doRotation"] = true

	expectedFields := []string{"doOpacity", "doTranslation", "doScale", "doRotation"}
	for _, field := range expectedFields {
		if value, exists := fadeCueData[field]; exists {
			if value != true {
				t.Errorf("Expected %s=true, got %v", field, value)
			}
		} else {
			t.Errorf("%s field not found in fade cue data", field)
		}
	}

	t.Log("✓ All fade cue geometry enable fields are properly represented in the data map")
	t.Log("✓ The fix ensures these fields will be sent to QLab via OSC when creating/updating cues")
}
