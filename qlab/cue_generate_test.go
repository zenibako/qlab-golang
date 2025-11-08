package qlab

import (
	"testing"

	"github.com/zenibako/qlab-golang/templates"
)

// TestNewCueGenerator tests creating a new CueGenerator
func TestNewCueGenerator(t *testing.T) {
	workspace, _ := setupWorkspaceWithCleanup(t)

	generator := NewCueGenerator(workspace)

	if generator == nil {
		t.Fatal("Expected non-nil CueGenerator")
	}

	if generator.workspace != workspace {
		t.Error("CueGenerator should reference the workspace")
	}
}

// TestGenerateCuesBasic tests basic cue generation
func TestGenerateCuesBasic(t *testing.T) {
	workspace, mockServer := setupWorkspaceWithCleanup(t)
	generator := NewCueGenerator(workspace)

	t.Log("=== Testing basic cue generation ===")

	// Create a simple memo cue template
	template := templates.CueTemplate{
		Type: "memo",
		Name: "Test Memo Cue",
		Properties: map[string]any{
			"notes": "Generated from template",
		},
	}

	request := templates.CueGenerationRequest{
		CueNumber: "100.0",
		Template:  template,
	}

	// Generate the cue
	result := generator.GenerateCues(request)

	if !result.Success {
		t.Fatalf("Expected successful cue generation, got errors: %v", result.Errors)
	}

	if len(result.CuesCreated) != 1 {
		t.Fatalf("Expected 1 cue created, got %d", len(result.CuesCreated))
	}

	created := result.CuesCreated[0]
	t.Logf("Created cue: ID=%s, Number=%s, Name=%s, Type=%s",
		created.UniqueID, created.CueNumber, created.Name, created.Type)

	// Verify the cue was created in the mock server
	if mockServer.GetCueCount() != 1 {
		t.Errorf("Expected 1 cue in mock server, got %d", mockServer.GetCueCount())
	}

	// Verify cue properties
	if created.Type != "memo" {
		t.Errorf("Expected type 'memo', got '%s'", created.Type)
	}
	if created.Name != "Test Memo Cue" {
		t.Errorf("Expected name 'Test Memo Cue', got '%s'", created.Name)
	}
	if created.CueNumber != "100.0" {
		t.Errorf("Expected cue number '100.0', got '%s'", created.CueNumber)
	}
}

// TestGenerateCuesWithChildren tests generating a group cue with children
func TestGenerateCuesWithChildren(t *testing.T) {
	workspace, mockServer := setupWorkspaceWithCleanup(t)
	generator := NewCueGenerator(workspace)

	t.Log("=== Testing cue generation with children ===")

	// Create a group cue with child cues
	template := templates.CueTemplate{
		Type: "group",
		Name: "Test Scene",
		Properties: map[string]any{
			"mode": 3, // Timeline mode
		},
		Children: []templates.CueTemplate{
			{
				Type: "light",
				Name: "Lights up",
			},
			{
				Type: "audio",
				Name: "Background music",
			},
		},
	}

	request := templates.CueGenerationRequest{
		CueNumber: "200.0",
		Template:  template,
	}

	// Generate the cues
	result := generator.GenerateCues(request)

	if !result.Success {
		t.Fatalf("Expected successful cue generation, got errors: %v", result.Errors)
	}

	// Should create 3 cues: 1 parent + 2 children
	expectedCount := 3
	if len(result.CuesCreated) != expectedCount {
		t.Fatalf("Expected %d cues created, got %d", expectedCount, len(result.CuesCreated))
	}

	// Verify parent cue
	parent := result.CuesCreated[0]
	if parent.Type != "group" {
		t.Errorf("Expected parent type 'group', got '%s'", parent.Type)
	}
	if parent.CueNumber != "200.0" {
		t.Errorf("Expected parent cue number '200.0', got '%s'", parent.CueNumber)
	}

	// Verify child cues
	child1 := result.CuesCreated[1]
	if child1.Type != "light" {
		t.Errorf("Expected first child type 'light', got '%s'", child1.Type)
	}
	if child1.ParentID != parent.UniqueID {
		t.Errorf("Expected child1 parent ID to match parent unique ID")
	}

	child2 := result.CuesCreated[2]
	if child2.Type != "audio" {
		t.Errorf("Expected second child type 'audio', got '%s'", child2.Type)
	}
	if child2.ParentID != parent.UniqueID {
		t.Errorf("Expected child2 parent ID to match parent unique ID")
	}

	t.Logf("Successfully created parent cue with %d children", len(template.Children))
	t.Logf("Mock server now has %d cues", mockServer.GetCueCount())
}

// TestGenerateCuesAudioWithProperties tests generating an audio cue with file target
func TestGenerateCuesAudioWithProperties(t *testing.T) {
	workspace, _ := setupWorkspaceWithCleanup(t)
	generator := NewCueGenerator(workspace)

	t.Log("=== Testing audio cue generation with properties ===")

	template := templates.CueTemplate{
		Type: "audio",
		Name: "Background Music",
		Properties: map[string]any{
			"fileTarget": "music/background.mp3",
		},
	}

	request := templates.CueGenerationRequest{
		CueNumber: "300.0",
		Template:  template,
	}

	result := generator.GenerateCues(request)

	if !result.Success {
		t.Fatalf("Expected successful cue generation, got errors: %v", result.Errors)
	}

	if len(result.CuesCreated) != 1 {
		t.Fatalf("Expected 1 cue created, got %d", len(result.CuesCreated))
	}

	created := result.CuesCreated[0]
	if created.Type != "audio" {
		t.Errorf("Expected type 'audio', got '%s'", created.Type)
	}

	t.Logf("Created audio cue: %s", created.Name)
}

// TestGenerateCuesWithParent tests generating cues with a specific parent ID
func TestGenerateCuesWithParent(t *testing.T) {
	workspace, _ := setupWorkspaceWithCleanup(t)
	generator := NewCueGenerator(workspace)

	t.Log("=== Testing cue generation with parent ID ===")

	// First create a parent group cue
	parentTemplate := templates.CueTemplate{
		Type: "group",
		Name: "Parent Group",
		Properties: map[string]any{
			"mode": 0, // Cue list mode
		},
	}

	parentRequest := templates.CueGenerationRequest{
		CueNumber: "400.0",
		Template:  parentTemplate,
	}

	parentResult := generator.GenerateCues(parentRequest)
	if !parentResult.Success {
		t.Fatalf("Failed to create parent cue: %v", parentResult.Errors)
	}

	parentID := parentResult.CuesCreated[0].UniqueID
	t.Logf("Created parent cue with ID: %s", parentID)

	// Now create a child cue with explicit parent ID
	childTemplate := templates.CueTemplate{
		Type: "memo",
		Name: "Child Memo",
	}

	childRequest := templates.CueGenerationRequest{
		CueNumber: "401.0",
		Template:  childTemplate,
		ParentID:  parentID,
	}

	childResult := generator.GenerateCues(childRequest)
	if !childResult.Success {
		t.Fatalf("Failed to create child cue: %v", childResult.Errors)
	}

	if len(childResult.CuesCreated) != 1 {
		t.Fatalf("Expected 1 child cue created, got %d", len(childResult.CuesCreated))
	}

	child := childResult.CuesCreated[0]
	if child.ParentID != parentID {
		t.Errorf("Expected child parent ID '%s', got '%s'", parentID, child.ParentID)
	}

	t.Logf("Successfully created child cue under parent")
}

// TestGenerateCuesMultipleLevels tests generating nested cues (group with children with children)
func TestGenerateCuesMultipleLevels(t *testing.T) {
	workspace, mockServer := setupWorkspaceWithCleanup(t)
	generator := NewCueGenerator(workspace)

	t.Log("=== Testing multi-level cue generation ===")

	// Create a deeply nested structure
	template := templates.CueTemplate{
		Type: "group",
		Name: "Act I",
		Properties: map[string]any{
			"mode": 0,
		},
		Children: []templates.CueTemplate{
			{
				Type: "group",
				Name: "Scene 1",
				Properties: map[string]any{
					"mode": 3, // Timeline
				},
				Children: []templates.CueTemplate{
					{
						Type: "light",
						Name: "Scene 1 Lights",
					},
					{
						Type: "audio",
						Name: "Scene 1 Music",
					},
				},
			},
			{
				Type: "memo",
				Name: "Intermission",
			},
		},
	}

	request := templates.CueGenerationRequest{
		CueNumber: "500.0",
		Template:  template,
	}

	result := generator.GenerateCues(request)

	if !result.Success {
		t.Fatalf("Expected successful cue generation, got errors: %v", result.Errors)
	}

	// Total: 1 parent + 2 children (1 group + 1 memo) + 2 grandchildren = 5 cues
	expectedCount := 5
	if len(result.CuesCreated) != expectedCount {
		t.Fatalf("Expected %d cues created, got %d", expectedCount, len(result.CuesCreated))
	}

	// Verify the hierarchy
	actI := result.CuesCreated[0]
	scene1 := result.CuesCreated[1]
	lights := result.CuesCreated[2]
	music := result.CuesCreated[3]
	intermission := result.CuesCreated[4]

	if actI.Type != "group" || actI.Name != "Act I" {
		t.Errorf("First cue should be 'Act I' group")
	}

	if scene1.Type != "group" || scene1.Name != "Scene 1" {
		t.Errorf("Second cue should be 'Scene 1' group")
	}
	if scene1.ParentID != actI.UniqueID {
		t.Errorf("Scene 1 should be child of Act I")
	}

	if lights.ParentID != scene1.UniqueID {
		t.Errorf("Lights should be child of Scene 1")
	}
	if music.ParentID != scene1.UniqueID {
		t.Errorf("Music should be child of Scene 1")
	}

	if intermission.ParentID != actI.UniqueID {
		t.Errorf("Intermission should be child of Act I")
	}

	t.Logf("Successfully created %d cues in nested hierarchy", len(result.CuesCreated))
	t.Logf("Mock server has %d cues", mockServer.GetCueCount())
}

// TestGenerateCuesErrorHandling tests error handling in cue generation
func TestGenerateCuesErrorHandling(t *testing.T) {
	workspace, _ := setupWorkspaceWithCleanup(t)
	generator := NewCueGenerator(workspace)

	t.Log("=== Testing error handling in cue generation ===")

	// Test with invalid cue type (mock server should still handle this gracefully)
	template := templates.CueTemplate{
		Type: "invalid_type",
		Name: "Invalid Cue",
	}

	request := templates.CueGenerationRequest{
		CueNumber: "999.0",
		Template:  template,
	}

	result := generator.GenerateCues(request)

	// The mock server will accept any type, so this should succeed
	// In a real QLab instance, this might fail
	t.Logf("Result success: %v", result.Success)
	t.Logf("Errors: %v", result.Errors)
	t.Logf("Cues created: %d", len(result.CuesCreated))
}

// TestExtractUniqueIDFromResult tests the unique ID extraction helper
func TestExtractUniqueIDFromResult(t *testing.T) {
	tests := []struct {
		name     string
		result   []any
		expected string
	}{
		{
			name:     "Valid JSON response with UUID",
			result:   []any{`{"data":"550e8400-e29b-41d4-a716-446655440000","status":"ok"}`},
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "Valid JSON with short UUID",
			result:   []any{`{"data":"MOCK-CUE-1","status":"ok"}`},
			expected: "MOCK-CUE-1",
		},
		{
			name:     "Empty result",
			result:   []any{},
			expected: "",
		},
		{
			name:     "Non-JSON string",
			result:   []any{"ok"},
			expected: "",
		},
		{
			name:     "JSON without data field",
			result:   []any{`{"status":"ok"}`},
			expected: "",
		},
		{
			name:     "JSON with null data",
			result:   []any{`{"data":null,"status":"ok"}`},
			expected: "",
		},
		{
			name:     "Invalid JSON",
			result:   []any{`{invalid json}`},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := extractUniqueIDFromResult(tt.result)
			if actual != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, actual)
			}
		})
	}
}
