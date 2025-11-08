package qlab

import (
	"testing"
)

// Removed cuejitsu dependency

// TestUberNoirComparisonDebug debugs the specific issue with uber_noir.cue three-way comparison
func TestUberNoirComparisonDebug(t *testing.T) {
	// Skip test - requires CUE file handler to be injected
	t.Skip("Test requires CUE file handler - skipping in standalone qlab-golang package")

	// Test code commented out - requires sourceData from CUE file handler
	/*
		workspace := &Workspace{}

		t.Logf("=== PARSED SOURCE DATA ===")
		t.Logf("Source data keys: %v", getKeys(sourceData))
		if ws, ok := sourceData["workspace"].(map[string]any); ok {
			t.Logf("Workspace keys: %v", getKeys(ws))
			if cues, ok := ws["cues"].([]any); ok {
				t.Logf("Found %d top-level cues", len(cues))
			}
		}

		// Index the source cues
		sourceCues := workspace.indexCuesFromWorkspace(sourceData)
		t.Logf("=== INDEXED SOURCE CUES ===")
		t.Logf("Found %d indexed source cues", len(sourceCues))
		for number, cue := range sourceCues {
			t.Logf("Source cue [%s]: name='%v', type='%v'", number, cue["name"], cue["type"])
		}

		// Simulate what current QLab cues would look like (identical to source)
		currentData := map[string]any{
			"data": map[string]any{
				"cueLists": []any{
					map[string]any{
						"cues": []any{
							map[string]any{
								"type":     "group",
								"number":   "0",
								"name":     "INT. CAR - NIGHT / It's pouring rain.",
								"uniqueID": "qlab-cue-0",
							},
							map[string]any{
								"type":     "audio",
								"number":   "1.0",
								"name":     "Round About Midnight - Miles Davis",
								"uniqueID": "qlab-cue-1.0",
							},
							map[string]any{
								"type":     "group",
								"number":   "1",
								"name":     "*Cool jazz music* starts to play.",
								"uniqueID": "qlab-cue-1",
							},
							map[string]any{
								"type":            "stop",
								"number":          "2.0",
								"cueTargetNumber": "1.0",
								"uniqueID":        "qlab-cue-2.0",
							},
							map[string]any{
								"type":     "group",
								"number":   "2",
								"name":     "KATE approaches the car and knocks on the window. *Music suddenly stops.*",
								"uniqueID": "qlab-cue-2",
							},
							map[string]any{
								"type":            "start",
								"number":          "3.0",
								"cueTargetNumber": "1.0",
								"uniqueID":        "qlab-cue-3.0",
							},
							map[string]any{
								"type":     "group",
								"number":   "3",
								"name":     "Joe goes to the dash to *turn on the jazz station*.",
								"uniqueID": "qlab-cue-3",
							},
						},
					},
				},
			},
		}

		// Index the current cues
		currentCues := workspace.indexCuesFromWorkspace(currentData)
		t.Logf("=== INDEXED CURRENT CUES ===")
		t.Logf("Found %d indexed current cues", len(currentCues))
		for number, cue := range currentCues {
			t.Logf("Current cue [%s]: name='%v', type='%v'", number, cue["name"], cue["type"])
		}

		// Test individual comparisons with detailed output
		for number := range sourceCues {
			if currentCue, exists := currentCues[number]; exists {
				sourceCue := sourceCues[number]

				// Show detailed property values before comparison
				t.Logf("=== DETAILED COMPARISON FOR CUE [%s] ===", number)
				t.Logf("Source name: '%v' (type: %T)", sourceCue["name"], sourceCue["name"])
				t.Logf("Current name: '%v' (type: %T)", currentCue["name"], currentCue["name"])
				t.Logf("Source type: '%v' (type: %T)", sourceCue["type"], sourceCue["type"])
				t.Logf("Current type: '%v' (type: %T)", currentCue["type"], currentCue["type"])

				// Test ALL properties being compared
				propertiesToCompare := []string{
					"name", "type", "fileTarget", "duration", "cueTargetNumber",
					"armed", "colorName", "flagged", "notes",
				}

				for _, prop := range propertiesToCompare {
					sourceVal := workspace.normalizeProperty(sourceCue[prop])
					currentVal := workspace.normalizeProperty(currentCue[prop])
					match := sourceVal == currentVal
					if !match {
						t.Logf("  MISMATCH %s: source='%s' vs current='%s'", prop, sourceVal, currentVal)
					}
				}

				matches := workspace.compareCueProperties(sourceCue, currentCue)
				t.Logf("Overall comparison result: %t", matches)
			}
		}
	*/
}
