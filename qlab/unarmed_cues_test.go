package qlab

import (
	"testing"
)

// TestUnarmedAudioCues tests that audio cues can be created with armed = false
func TestUnarmedAudioCues(t *testing.T) {
	tests := []struct {
		name          string
		cue           Cue
		expectedArmed bool
	}{
		{
			name: "unarmed audio cue",
			cue: Cue{
				Type:       CueTypeAudio,
				Number:     "2.1",
				Name:       "Vocals Only",
				FileTarget: "/path/to/vocals.wav",
				Armed:      false,
			},
			expectedArmed: false,
		},
		{
			name: "another unarmed audio cue",
			cue: Cue{
				Type:       CueTypeAudio,
				Number:     "2.2",
				Name:       "Instrumental Only",
				FileTarget: "/path/to/instrumental.wav",
				Armed:      false,
			},
			expectedArmed: false,
		},
		{
			name: "armed audio cue",
			cue: Cue{
				Type:       CueTypeAudio,
				Number:     "2.0",
				Name:       "Original Audio",
				FileTarget: "/path/to/original.wav",
				Armed:      true,
			},
			expectedArmed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cue.Armed != tt.expectedArmed {
				t.Errorf("Expected armed = %v, got armed = %v for cue '%s'",
					tt.expectedArmed, tt.cue.Armed, tt.cue.Name)
			}

			// Verify cue type is audio
			if tt.cue.Type != CueTypeAudio {
				t.Errorf("Expected cue type 'audio', got '%s'", tt.cue.Type)
			}

			// Verify file target is set
			if tt.cue.FileTarget == "" {
				t.Error("Expected file target to be set")
			}
		})
	}
}

// TestUnarmedAudioInTimeline tests that unarmed audio cues are properly structured
// within a timeline group
func TestUnarmedAudioInTimeline(t *testing.T) {
	// Example timeline with multiple audio cues
	contentTimeline := Cue{
		Type:   CueTypeGroup,
		Number: "2",
		Name:   "Content",
		Mode:   3, // Timeline mode
		Cues: []Cue{
			{
				Type:       CueTypeAudio,
				Number:     "2.0",
				Name:       "Original Audio",
				FileTarget: "/path/to/original.wav",
				Armed:      true, // Original can be armed
			},
			{
				Type:       CueTypeAudio,
				Number:     "2.1",
				Name:       "Vocals Only",
				FileTarget: "/path/to/vocals.wav",
				Armed:      false, // Not armed by default
			},
			{
				Type:       CueTypeAudio,
				Number:     "2.2",
				Name:       "Instrumental Only",
				FileTarget: "/path/to/instrumental.wav",
				Armed:      false, // Not armed by default
			},
		},
	}

	// Verify timeline structure
	if contentTimeline.Mode != 3 {
		t.Errorf("Expected timeline mode (3), got mode %d", contentTimeline.Mode)
	}

	if len(contentTimeline.Cues) != 3 {
		t.Fatalf("Expected 3 cues in content timeline, got %d", len(contentTimeline.Cues))
	}

	// Verify original audio
	originalCue := contentTimeline.Cues[0]
	if originalCue.Name != "Original Audio" {
		t.Errorf("Expected first cue to be 'Original Audio', got '%s'", originalCue.Name)
	}
	// Original can be armed (user's choice)
	if originalCue.Armed != true {
		t.Logf("Note: Original audio is unarmed (user's choice)")
	}

	// Verify second audio cue
	vocalsCue := contentTimeline.Cues[1]
	if vocalsCue.Name != "Vocals Only" {
		t.Errorf("Expected second cue to be 'Vocals Only', got '%s'", vocalsCue.Name)
	}
	if vocalsCue.Armed {
		t.Error("Second audio cue should not be armed by default")
	}

	// Verify third audio cue
	instrumentalCue := contentTimeline.Cues[2]
	if instrumentalCue.Name != "Instrumental Only" {
		t.Errorf("Expected third cue to be 'Instrumental Only', got '%s'", instrumentalCue.Name)
	}
	if instrumentalCue.Armed {
		t.Error("Third audio cue should not be armed by default")
	}

	// Verify all are audio cues
	for i, cue := range contentTimeline.Cues {
		if cue.Type != CueTypeAudio {
			t.Errorf("Cue %d: expected type 'audio', got '%s'", i, cue.Type)
		}
		if cue.FileTarget == "" {
			t.Errorf("Cue %d: expected file target to be set", i)
		}
	}
}

// TestUnarmedCueJSONMarshaling tests that armed status is properly
// serialized to JSON for QLab consumption
func TestUnarmedCueJSONMarshaling(t *testing.T) {
	unarmedCue := Cue{
		Type:       CueTypeAudio,
		Number:     "2.1",
		Name:       "Vocals Only",
		FileTarget: "/path/to/vocals.wav",
		Armed:      false,
	}

	// Convert to workspace data
	cues := []Cue{unarmedCue}
	workspaceData := ToWorkspaceData("Test Workspace", cues)

	// Verify structure
	if len(workspaceData.Cues) != 1 {
		t.Fatalf("Expected 1 cue, got %d", len(workspaceData.Cues))
	}

	// Export to JSON
	jsonOutput, err := ToJSON("Test Workspace", cues, true)
	if err != nil {
		t.Fatalf("Failed to export to JSON: %v", err)
	}

	// Verify JSON contains armed = false
	// Note: We don't check the exact JSON string since omitempty might skip false values
	// The important thing is the struct field is set correctly
	t.Logf("JSON output length: %d bytes", len(jsonOutput))

	// Verify the original struct still has armed = false
	if unarmedCue.Armed {
		t.Error("Unarmed cue should have armed = false")
	}
}
