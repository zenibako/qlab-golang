package qlab_test

import (
	"encoding/json"
	"fmt"

	"github.com/zenibako/qlab-golang/qlab"
)

// Example demonstrating how to export workspace data to JSON
func ExampleToJSON() {
	cues := []qlab.Cue{
		{
			Type:   "text",
			Number: "1",
			Name:   "Opening Text",
			Text:   "Welcome to the show!",
		},
		{
			Type:       "audio",
			Number:     "2",
			Name:       "Background Music",
			FileTarget: "audio/music.mp3",
		},
	}

	// Export to JSON with indentation
	jsonOutput, err := qlab.ToJSON("My Show", cues, true)
	if err != nil {
		panic(err)
	}

	fmt.Println("JSON export successful, length:", len(jsonOutput))
	// Output: JSON export successful, length: 291
}

// Example demonstrating how to get structured data for custom serialization
func ExampleToWorkspaceData() {
	cues := []qlab.Cue{
		{
			Type:   "group",
			Number: "1",
			Name:   "Scene 1",
			Mode:   qlab.GroupModeTimeline,
			Cues: []qlab.Cue{
				{
					Type:   "text",
					Number: "1.1",
					Name:   "Title Card",
					Text:   "Act I",
				},
			},
		},
	}

	// Get structured data
	workspaceData := qlab.ToWorkspaceData("My Production", cues)

	fmt.Printf("Workspace: %s\n", workspaceData.Name)
	fmt.Printf("Top-level cues: %d\n", len(workspaceData.Cues))
	fmt.Printf("Nested cues: %d\n", len(workspaceData.Cues[0].Cues))

	// Output:
	// Workspace: My Production
	// Top-level cues: 1
	// Nested cues: 1
}

// Example demonstrating how to serialize to custom formats
func ExampleToWorkspaceData_customFormat() {
	cues := []qlab.Cue{
		{
			Type:     "text",
			Number:   "1",
			Name:     "Test Cue",
			Duration: 5.0,
		},
	}

	// Get structured data
	workspaceData := qlab.ToWorkspaceData("Test Show", cues)

	// Serialize to JSON (or YAML, TOML, XML, etc.)
	jsonBytes, _ := json.Marshal(workspaceData)

	fmt.Println("Can serialize to any format")
	fmt.Println("JSON length:", len(jsonBytes))

	// Output:
	// Can serialize to any format
	// JSON length: 89
}
