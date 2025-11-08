package qlab

import (
	"encoding/json"
	"fmt"
)

// ToWorkspaceData converts workspace name and cues to structured data.
// The caller can serialize this to JSON, YAML, TOML, or any other format.
// This returns a WorkspaceData struct with Name and Cues fields.
func ToWorkspaceData(workspaceName string, cues []Cue) WorkspaceData {
	// Normalize all cues to ensure proper defaults
	normalizedCues := make([]Cue, len(cues))
	for i := range cues {
		normalizedCues[i] = cues[i]
		NormalizeCue(&normalizedCues[i])
	}

	return WorkspaceData{
		Name: workspaceName,
		Cues: normalizedCues,
	}
}

// ToJSON converts workspace name and cues to JSON format.
// The caller can write this to a .json file if needed.
func ToJSON(workspaceName string, cues []Cue, indent bool) (string, error) {
	data := ToWorkspaceData(workspaceName, cues)
	var result []byte
	var err error

	if indent {
		result, err = json.MarshalIndent(data, "", "  ")
	} else {
		result, err = json.Marshal(data)
	}

	if err != nil {
		return "", fmt.Errorf("failed to marshal workspace data: %w", err)
	}

	return string(result), nil
}
