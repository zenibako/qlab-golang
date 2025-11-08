package qlab

import (
	"encoding/json"
	"testing"
)

// TestTextCueGeometryFields verifies that all geometry fields are properly defined
func TestTextCueGeometryFields(t *testing.T) {
	// Create a text cue with all geometry parameters
	cue := Cue{
		Type:          CueTypeText,
		Name:          "Geometry Test",
		Number:        "1.0",
		Text:          "Hello World",
		TextColor:     []float64{1.0, 1.0, 1.0, 1.0}, // White
		TextBgColor:   []float64{0.0, 0.0, 0.0, 0.5}, // Semi-transparent black
		TextFontSize:  48.0,
		TextAlignment: TextAlignCenter,
		StageID:       "test-stage-id",
		StageName:     "Test Stage",
		Translation:   []float64{100.0, 200.0},
		Scale:         []float64{1.5, 2.0},
		Rotation:      45.0,
		RotationType:  RotationTypeZ,
		Quaternion:    []float64{1.0, 0.0, 0.0, 0.0},
		Opacity:       0.8,
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(cue, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal cue: %v", err)
	}

	t.Logf("Marshaled text cue:\n%s", string(jsonData))

	// Unmarshal back
	var unmarshaled Cue
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal cue: %v", err)
	}

	// Verify all fields
	tests := []struct {
		name     string
		got      any
		expected any
	}{
		{"Type", unmarshaled.Type, CueTypeText},
		{"Name", unmarshaled.Name, "Geometry Test"},
		{"Text", unmarshaled.Text, "Hello World"},
		{"TextFontSize", unmarshaled.TextFontSize, 48.0},
		{"TextAlignment", unmarshaled.TextAlignment, TextAlignCenter},
		{"Translation length", len(unmarshaled.Translation), 2},
		{"Translation[0]", unmarshaled.Translation[0], 100.0},
		{"Translation[1]", unmarshaled.Translation[1], 200.0},
		{"Scale length", len(unmarshaled.Scale), 2},
		{"Scale[0]", unmarshaled.Scale[0], 1.5},
		{"Scale[1]", unmarshaled.Scale[1], 2.0},
		{"Rotation", unmarshaled.Rotation, 45.0},
		{"RotationType", unmarshaled.RotationType, RotationTypeZ},
		{"Quaternion length", len(unmarshaled.Quaternion), 4},
		{"Opacity", unmarshaled.Opacity, 0.8},
		{"TextColor length", len(unmarshaled.TextColor), 4},
		{"TextBgColor length", len(unmarshaled.TextBgColor), 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.got)
			}
		})
	}
}

// TestTextCueJSONTags verifies JSON tag names match QLab OSC API
func TestTextCueJSONTags(t *testing.T) {
	cue := Cue{
		Translation:   []float64{10, 20},
		Scale:         []float64{2, 3},
		Rotation:      90,
		RotationType:  RotationTypeX,
		Quaternion:    []float64{0, 1, 0, 0},
		Opacity:       0.5,
		TextColor:     []float64{1, 0, 0, 1},
		TextBgColor:   []float64{0, 0, 1, 0.5},
		TextFontSize:  72,
		TextAlignment: TextAlignRight,
	}

	jsonData, err := json.Marshal(cue)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Convert to map to check JSON keys
	var data map[string]any
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// Verify key names match QLab OSC API
	expectedKeys := []string{
		"translation",
		"scale",
		"rotation",
		"rotationType",
		"quaternion",
		"opacity",
		"text/format/color",
		"text/format/backgroundColor",
		"text/format/fontSize",
		"text/format/alignment",
	}

	for _, key := range expectedKeys {
		if _, exists := data[key]; !exists {
			t.Errorf("Expected JSON key '%s' not found in output", key)
		} else {
			t.Logf("✓ Found key: %s = %v", key, data[key])
		}
	}
}

// TestRotationTypeConstants verifies rotation type constants
func TestRotationTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{"3D Orientation", RotationType3D, 0},
		{"X Rotation", RotationTypeX, 1},
		{"Y Rotation", RotationTypeY, 2},
		{"Z Rotation", RotationTypeZ, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, tt.value)
			}
		})
	}
}

// TestTextAlignmentConstants verifies text alignment constants
func TestTextAlignmentConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"Left", TextAlignLeft, "left"},
		{"Center", TextAlignCenter, "center"},
		{"Right", TextAlignRight, "right"},
		{"Justify", TextAlignJustify, "justify"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.value)
			}
		})
	}
}

// TestCueTypeConstants verifies cue type constants
func TestCueTypeConstants(t *testing.T) {
	// Test that common cue types are defined
	cueTypes := []string{
		CueTypeAudio,
		CueTypeVideo,
		CueTypeText,
		CueTypeFade,
		CueTypeGroup,
		CueTypeMemo,
	}

	for _, cueType := range cueTypes {
		if cueType == "" {
			t.Errorf("Cue type constant is empty")
		} else {
			t.Logf("✓ Cue type defined: %s", cueType)
		}
	}
}
