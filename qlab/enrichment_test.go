package qlab

import (
	"testing"
)

func TestPropertyEnrichment(t *testing.T) {
	mockServer := NewMockOSCServer("localhost", 55004)
	err := mockServer.Start()
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer func() {
		if err := mockServer.Stop(); err != nil {
			t.Logf("Failed to stop mock server: %v", err)
		}
	}()

	workspace := NewWorkspace("localhost", 55004)
	_, err = workspace.Init("test-passcode")
	if err != nil {
		t.Fatalf("Failed to initialize workspace: %v", err)
	}

	t.Run("fileTarget is populated", func(t *testing.T) {
		audioCueData := map[string]any{
			"type":       "audio",
			"number":     "1.0",
			"name":       "Test Audio",
			"fileTarget": "music/test.mp3",
		}
		_, err := workspace.createCue(audioCueData, "1.0")
		if err != nil {
			t.Fatalf("Failed to create audio cue: %v", err)
		}

		currentWorkspace, err := workspace.queryCurrentWorkspaceState()
		if err != nil {
			t.Fatalf("Failed to query workspace state: %v", err)
		}

		found := false
		data, ok := currentWorkspace["data"].([]any)
		if !ok {
			t.Fatalf("No data array in workspace response")
		}

		for _, cueListData := range data {
			if cueList, ok := cueListData.(map[string]any); ok {
				if cues, ok := cueList["cues"].([]any); ok {
					for _, cueData := range cues {
						if cue, ok := cueData.(map[string]any); ok {
							if fileTarget, ok := cue["fileTarget"].(string); ok {
								found = true
								t.Logf("✓ Found fileTarget: %s", fileTarget)
								if fileTarget != "music/test.mp3" {
									t.Errorf("Expected fileTarget 'music/test.mp3', got '%s'", fileTarget)
								}
							}
						}
					}
				}
			}
		}

		if !found {
			t.Error("fileTarget property was not populated in queryCurrentWorkspaceState")
		}
	})

	t.Run("cueTargetNumber is populated", func(t *testing.T) {
		startCueData := map[string]any{
			"type":            "start",
			"number":          "2.0",
			"name":            "Start Audio",
			"cueTargetNumber": "1.0",
		}
		uniqueID, err := workspace.createCue(startCueData, "2.0")
		if err != nil {
			t.Fatalf("Failed to create start cue: %v", err)
		}
		t.Logf("Created start cue with ID: %s", uniqueID)

		currentWorkspace, err := workspace.queryCurrentWorkspaceState()
		if err != nil {
			t.Fatalf("Failed to query workspace state: %v", err)
		}

		found := false
		data, ok := currentWorkspace["data"].([]any)
		if !ok {
			t.Fatalf("No data array in workspace response")
		}

		for _, cueListData := range data {
			if cueList, ok := cueListData.(map[string]any); ok {
				t.Logf("Cue list name: %v, type: %v", cueList["name"], cueList["type"])
				if cues, ok := cueList["cues"].([]any); ok {
					t.Logf("Found %d cues in list", len(cues))
					for _, cueData := range cues {
						if cue, ok := cueData.(map[string]any); ok {
							cueNum := cue["number"]
							cueName := cue["name"]
							cueType := cue["type"]
							t.Logf("Cue: number=%v, name=%v, type=%v", cueNum, cueName, cueType)

							if cueNumber, ok := cue["number"].(string); ok && cueNumber == "2.0" {
								keys := make([]string, 0, len(cue))
								for k := range cue {
									keys = append(keys, k)
								}
								t.Logf("Found target cue 2.0, keys: %v", keys)
								if cueTargetNumber, ok := cue["cueTargetNumber"].(string); ok {
									found = true
									t.Logf("✓ Found cueTargetNumber: %s", cueTargetNumber)
									if cueTargetNumber != "1.0" {
										t.Errorf("Expected cueTargetNumber '1.0', got '%s'", cueTargetNumber)
									}
								} else {
									t.Logf("cueTargetNumber not found in cue")
								}
							}
						}
					}
				}
			}
		}

		if !found {
			t.Error("cueTargetNumber property was not populated in queryCurrentWorkspaceState")
		}
	})
}
