package qlab

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestToJSONIncludesDuration(t *testing.T) {
	// Test that all cues include duration field when serialized to JSON
	cues := []Cue{
		{
			Type:   "group",
			Number: "1",
			Name:   "Test Group",
			Mode:   3,
			Cues: []Cue{
				{
					Type:     "text",
					Number:   "1.1",
					Name:     "With Duration",
					Text:     "This has a duration",
					Duration: 5.0,
					PreWait:  10.0,
				},
				{
					Type:    "text",
					Number:  "1.2",
					Name:    "Without Duration",
					Text:    "This has no duration",
					PreWait: 15.0,
				},
				{
					Type:        "text",
					Number:      "1.3",
					Name:        "With Colors",
					Text:        "Colored text",
					TextColor:   []float64{1.0, 1.0, 1.0, 1.0},
					TextBgColor: []float64{0, 0, 0, 0},
					Duration:    0, // Zero duration
				},
			},
		},
	}

	// Use the new structured data API
	workspaceData := ToWorkspaceData("Test Workspace", cues)

	// Serialize to JSON
	jsonBytes, err := json.MarshalIndent(workspaceData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	result := string(jsonBytes)

	// Verify that cues with non-zero duration have it set
	if !strings.Contains(result, `"duration": 5`) {
		t.Errorf("Expected to find duration: 5 for cue with 5.0 duration")
		t.Logf("Generated JSON:\n%s", result)
	}

	// Verify the JSON is valid and parseable
	var parsed WorkspaceData
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to parse generated JSON: %v", err)
	}

	// Verify structure
	if len(parsed.Cues) != 1 {
		t.Errorf("Expected 1 top-level cue, got %d", len(parsed.Cues))
	}
	if len(parsed.Cues[0].Cues) != 3 {
		t.Errorf("Expected 3 nested cues, got %d", len(parsed.Cues[0].Cues))
	}
}

func TestNormalizeCue(t *testing.T) {
	// Test that NormalizeCue fixes invalid cues
	cue := &Cue{
		Type:        "text",                // Missing type would be defaulted
		TextColor:   []float64{1.0, 0.5},   // Invalid - should be 4 elements
		TextBgColor: []float64{0, 0, 0, 0}, // Valid
		Cues: []Cue{
			{
				Type:      "",             // Empty type should be defaulted
				TextColor: []float64{1.0}, // Invalid
			},
		},
	}

	NormalizeCue(cue)

	if cue.Type != "text" {
		t.Errorf("Expected type to remain 'text', got %q", cue.Type)
	}

	if cue.TextColor != nil {
		t.Errorf("Expected invalid TextColor to be cleared, got %v", cue.TextColor)
	}

	if len(cue.TextBgColor) != 4 {
		t.Errorf("Expected valid TextBgColor to remain, got %v", cue.TextBgColor)
	}

	if cue.Cues[0].Type != "group" {
		t.Errorf("Expected empty type to default to 'group', got %q", cue.Cues[0].Type)
	}

	if cue.Cues[0].TextColor != nil {
		t.Errorf("Expected nested invalid TextColor to be cleared, got %v", cue.Cues[0].TextColor)
	}
}

func TestToJSONGroupCues(t *testing.T) {
	// Test that group cues are properly serialized
	cues := []Cue{
		{
			Type:   "group",
			Number: "1",
			Name:   "Timeline Group",
			Mode:   3,
		},
	}

	workspaceData := ToWorkspaceData("Test Workspace", cues)
	jsonBytes, err := json.MarshalIndent(workspaceData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	result := string(jsonBytes)

	// Verify group cue properties are present
	if !strings.Contains(result, `"type": "group"`) {
		t.Errorf("Expected to find group cue type")
		t.Logf("Generated JSON:\n%s", result)
	}

	if !strings.Contains(result, `"mode": 3`) {
		t.Errorf("Expected to find mode: 3")
		t.Logf("Generated JSON:\n%s", result)
	}
}

func TestToWorkspaceData(t *testing.T) {
	// Test that ToWorkspaceData returns structured data that can be serialized
	cues := []Cue{
		{
			Type:     "text",
			Number:   "1",
			Name:     "Test Text Cue",
			Text:     "Hello World",
			Duration: 5.0,
		},
		{
			Type:       "audio",
			Number:     "2",
			Name:       "Test Audio Cue",
			FileTarget: "/path/to/audio.mp3",
		},
	}

	workspaceData := ToWorkspaceData("Test Workspace", cues)

	// Verify the structure
	if workspaceData.Name != "Test Workspace" {
		t.Errorf("Expected workspace name 'Test Workspace', got %q", workspaceData.Name)
	}

	if len(workspaceData.Cues) != 2 {
		t.Errorf("Expected 2 cues, got %d", len(workspaceData.Cues))
	}

	// Verify first cue
	if workspaceData.Cues[0].Type != "text" {
		t.Errorf("Expected first cue type 'text', got %q", workspaceData.Cues[0].Type)
	}
	if workspaceData.Cues[0].Text != "Hello World" {
		t.Errorf("Expected first cue text 'Hello World', got %q", workspaceData.Cues[0].Text)
	}

	// Verify second cue
	if workspaceData.Cues[1].Type != "audio" {
		t.Errorf("Expected second cue type 'audio', got %q", workspaceData.Cues[1].Type)
	}
	if workspaceData.Cues[1].FileTarget != "/path/to/audio.mp3" {
		t.Errorf("Expected second cue fileTarget '/path/to/audio.mp3', got %q", workspaceData.Cues[1].FileTarget)
	}
}

func TestToJSON(t *testing.T) {
	// Test that ToJSON produces valid, parseable JSON
	cues := []Cue{
		{
			Type:   "text",
			Number: "1",
			Name:   "Test Cue",
			Text:   "Sample text",
		},
	}

	// Test without indentation
	jsonStr, err := ToJSON("Test Workspace", cues, false)
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify it's valid JSON by unmarshaling
	var parsed WorkspaceData
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if parsed.Name != "Test Workspace" {
		t.Errorf("Expected workspace name 'Test Workspace', got %q", parsed.Name)
	}

	if len(parsed.Cues) != 1 {
		t.Fatalf("Expected 1 cue, got %d", len(parsed.Cues))
	}

	if parsed.Cues[0].Type != "text" {
		t.Errorf("Expected cue type 'text', got %q", parsed.Cues[0].Type)
	}

	// Test with indentation
	jsonIndented, err := ToJSON("Test Workspace", cues, true)
	if err != nil {
		t.Fatalf("ToJSON with indent failed: %v", err)
	}

	// Indented version should be longer and contain newlines
	if len(jsonIndented) <= len(jsonStr) {
		t.Errorf("Expected indented JSON to be longer than compact JSON")
	}

	if !strings.Contains(jsonIndented, "\n") {
		t.Errorf("Expected indented JSON to contain newlines")
	}

	// Verify indented JSON is also valid
	if err := json.Unmarshal([]byte(jsonIndented), &parsed); err != nil {
		t.Fatalf("Failed to parse indented JSON output: %v", err)
	}
}

func TestStructuredDataWithNestedCues(t *testing.T) {
	// Test that nested cues are properly represented in structured data
	cues := []Cue{
		{
			Type:   "group",
			Number: "1",
			Name:   "Parent Group",
			Mode:   GroupModeTimeline,
			Cues: []Cue{
				{
					Type:   "text",
					Number: "1.1",
					Name:   "Child Text",
					Text:   "Nested text",
				},
				{
					Type:   "audio",
					Number: "1.2",
					Name:   "Child Audio",
				},
			},
		},
	}

	workspaceData := ToWorkspaceData("Test Workspace", cues)

	// Verify parent cue
	if len(workspaceData.Cues) != 1 {
		t.Fatalf("Expected 1 parent cue, got %d", len(workspaceData.Cues))
	}

	parent := workspaceData.Cues[0]
	if parent.Type != "group" {
		t.Errorf("Expected parent type 'group', got %q", parent.Type)
	}

	// Verify nested cues
	if len(parent.Cues) != 2 {
		t.Fatalf("Expected 2 child cues, got %d", len(parent.Cues))
	}

	if parent.Cues[0].Type != "text" {
		t.Errorf("Expected first child type 'text', got %q", parent.Cues[0].Type)
	}

	if parent.Cues[1].Type != "audio" {
		t.Errorf("Expected second child type 'audio', got %q", parent.Cues[1].Type)
	}

	// Test JSON serialization preserves nesting
	jsonStr, err := ToJSON("Test Workspace", cues, true)
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var parsed WorkspaceData
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(parsed.Cues[0].Cues) != 2 {
		t.Errorf("Expected 2 nested cues after JSON round-trip, got %d", len(parsed.Cues[0].Cues))
	}
}
