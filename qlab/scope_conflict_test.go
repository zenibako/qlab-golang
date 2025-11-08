package qlab

import (
	"testing"
)

// TestScopeBasedComparison tests the scope-based comparison functionality
func TestScopeBasedComparison(t *testing.T) {
	workspace := &Workspace{}

	// Create test data with three versions: source, cache, and current QLab
	sourceCues := map[string]map[string]any{
		"1.0": {
			"uniqueID": "source-cue-1",
			"number":   "1.0",
			"name":     "Source Cue Modified",
			"type":     "audio",
			"notes":    "Source notes",
		},
	}

	cachedCues := map[string]map[string]any{
		"1.0": {
			"uniqueID": "cache-cue-1",
			"number":   "1.0",
			"name":     "Original Cue",
			"type":     "audio",
			"notes":    "Original notes",
		},
	}

	currentCues := map[string]map[string]any{
		"1.0": {
			"uniqueID": "qlab-cue-1",
			"number":   "1.0",
			"name":     "QLab Cue Modified",
			"type":     "audio",
			"notes":    "QLab notes",
		},
	}

	// Wrap in workspace structure
	sourceData := map[string]any{"cues": []any{sourceCues["1.0"]}}
	cachedData := map[string]any{"cues": []any{cachedCues["1.0"]}}
	currentData := map[string]any{"cues": []any{currentCues["1.0"]}}

	// Perform scope-based comparison
	scopeComparison, err := workspace.PerformScopeBasedComparison(sourceData, cachedData, currentData)
	if err != nil {
		t.Fatalf("PerformScopeBasedComparison failed: %v", err)
	}

	// Verify workspace scope
	if scopeComparison.Scope != ScopeWorkspace {
		t.Errorf("Expected workspace scope, got %s", scopeComparison.Scope)
	}

	if !scopeComparison.HasChanges {
		t.Error("Expected changes to be detected")
	}

	if !scopeComparison.ConflictExists {
		t.Error("Expected conflicts to be detected")
	}

	// Verify cue scope
	if len(scopeComparison.ChildScopes) != 1 {
		t.Fatalf("Expected 1 cue scope, got %d", len(scopeComparison.ChildScopes))
	}

	cueScope := scopeComparison.ChildScopes[0]
	if cueScope.Scope != ScopeCue {
		t.Errorf("Expected cue scope, got %s", cueScope.Scope)
	}

	if cueScope.Identifier != "1.0" {
		t.Errorf("Expected cue identifier '1.0', got '%s'", cueScope.Identifier)
	}

	// Verify field conflicts
	if len(cueScope.FieldChanges) == 0 {
		t.Error("Expected field changes to be detected")
	}

	// Check for name field conflict (modified in both source and QLab)
	nameConflict, hasNameConflict := cueScope.FieldChanges["name"]
	if !hasNameConflict {
		t.Error("Expected 'name' field conflict")
	} else {
		if nameConflict.FieldName != "name" {
			t.Errorf("Expected field name 'name', got '%s'", nameConflict.FieldName)
		}

		if nameConflict.SourceValue != "Source Cue Modified" {
			t.Errorf("Expected source name 'Source Cue Modified', got '%v'", nameConflict.SourceValue)
		}

		if nameConflict.CacheValue != "Original Cue" {
			t.Errorf("Expected cache name 'Original Cue', got '%v'", nameConflict.CacheValue)
		}

		if nameConflict.QLabValue != "QLab Cue Modified" {
			t.Errorf("Expected QLab name 'QLab Cue Modified', got '%v'", nameConflict.QLabValue)
		}
	}

	t.Logf("Scope comparison successful: %d field changes detected", len(cueScope.FieldChanges))
}

// TestMergedScopeGeneration tests merged result generation after conflict resolution
func TestMergedScopeGeneration(t *testing.T) {
	workspace := &Workspace{}

	// Create a simple scope with field conflicts
	scopeComparison := &ScopeComparison{
		Scope:      ScopeCue,
		Identifier: "1.0",
		FieldChanges: map[string]*FieldConflict{
			"name": {
				FieldName:   "name",
				SourceValue: "Source Name",
				CacheValue:  "Cache Name",
				QLabValue:   "QLab Name",
			},
			"notes": {
				FieldName:   "notes",
				SourceValue: "Source Notes",
				CacheValue:  "Cache Notes",
				QLabValue:   "QLab Notes",
			},
		},
		HasChanges:     true,
		ConflictExists: true,
	}

	// Create comparison with user choices
	comparison := &ThreeWayComparison{
		QLabChosenCues:   make(map[string]bool),
		QLabChosenFields: make(map[string]map[string]bool),
	}

	// User chose to keep QLab's name but use source notes
	comparison.QLabChosenFields["1.0"] = map[string]bool{
		"name": true, // Keep QLab
	}

	// Generate merged scope
	mergedScope, err := workspace.GenerateMergedScope(scopeComparison, comparison)
	if err != nil {
		t.Fatalf("GenerateMergedScope failed: %v", err)
	}

	// Verify merged data
	if mergedScope.Scope != ScopeCue {
		t.Errorf("Expected cue scope, got %s", mergedScope.Scope)
	}

	if mergedScope.Identifier != "1.0" {
		t.Errorf("Expected identifier '1.0', got '%s'", mergedScope.Identifier)
	}

	// Check name field (should be from QLab)
	if mergedScope.MergedData["name"] != "QLab Name" {
		t.Errorf("Expected merged name to be 'QLab Name', got '%v'", mergedScope.MergedData["name"])
	}

	if mergedScope.SourceFields["name"] != "qlab" {
		t.Errorf("Expected name source to be 'qlab', got '%s'", mergedScope.SourceFields["name"])
	}

	// Check notes field (should be from source by default)
	if mergedScope.MergedData["notes"] != "Source Notes" {
		t.Errorf("Expected merged notes to be 'Source Notes', got '%v'", mergedScope.MergedData["notes"])
	}

	if mergedScope.SourceFields["notes"] != "source" {
		t.Errorf("Expected notes source to be 'source', got '%s'", mergedScope.SourceFields["notes"])
	}

	t.Logf("Merged scope successful: %d fields merged from %d sources",
		len(mergedScope.MergedData), len(mergedScope.SourceFields))
}

// TestFieldLevelConflictIdentification tests identifying conflicts at field level
func TestFieldLevelConflictIdentification(t *testing.T) {
	workspace := &Workspace{}

	// Create scope with various conflict scenarios
	scopeComparison := &ScopeComparison{
		Scope:      ScopeWorkspace,
		Identifier: "workspace",
		ChildScopes: []*ScopeComparison{
			{
				Scope:      ScopeCue,
				Identifier: "1.0",
				FieldChanges: map[string]*FieldConflict{
					"name": {
						FieldName:   "name",
						SourceValue: "Source Name",
						CacheValue:  "Cache Name",
						QLabValue:   "Different Name",
					},
				},
				HasChanges:     true,
				ConflictExists: true,
			},
			{
				Scope:      ScopeCue,
				Identifier: "2.0",
				FieldChanges: map[string]*FieldConflict{
					"notes": {
						FieldName:   "notes",
						SourceValue: "Updated Notes",
						CacheValue:  "Old Notes",
						QLabValue:   "Old Notes",
					},
				},
				HasChanges:     true,
				ConflictExists: false, // Only source changed
			},
		},
		HasChanges:     true,
		ConflictExists: true,
	}

	// Identify conflicts
	conflicts := workspace.identifyConflictsFromScope(scopeComparison)

	// Should find 1 conflict (cue 1.0 has three-way divergence)
	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflict, got %d", len(conflicts))
	}

	if len(conflicts) > 0 {
		conflict := conflicts[0]

		if conflict.CueIdentifier != "1.0" {
			t.Errorf("Expected conflict on cue '1.0', got '%s'", conflict.CueIdentifier)
		}

		if conflict.Scope != ScopeCue {
			t.Errorf("Expected cue scope, got %s", conflict.Scope)
		}

		if len(conflict.FieldConflicts) == 0 {
			t.Error("Expected field conflicts to be populated")
		}

		if len(conflict.Properties) == 0 {
			t.Error("Expected conflicting properties list")
		}

		t.Logf("Conflict identified: %s - %d fields, type: %s",
			conflict.CueIdentifier, len(conflict.FieldConflicts), conflict.ConflictType)
	}
}

// TestExtractMergedWorkspaceData tests extracting final merged workspace structure
func TestExtractMergedWorkspaceData(t *testing.T) {
	workspace := &Workspace{}

	// Create merged scope with cues
	mergedScope := &MergedScope{
		Scope:      ScopeWorkspace,
		Identifier: "workspace",
		ChildScopes: []*MergedScope{
			{
				Scope:      ScopeCue,
				Identifier: "1.0",
				MergedData: map[string]any{
					"number": "1.0",
					"name":   "Merged Cue 1",
					"type":   "audio",
				},
				SourceFields: map[string]string{
					"number": "source",
					"name":   "qlab",
					"type":   "source",
				},
			},
			{
				Scope:      ScopeCue,
				Identifier: "2.0",
				MergedData: map[string]any{
					"number": "2.0",
					"name":   "Merged Cue 2",
					"type":   "group",
				},
				SourceFields: map[string]string{
					"number": "source",
					"name":   "source",
					"type":   "source",
				},
			},
		},
	}

	// Extract workspace data
	workspaceData, err := workspace.ExtractMergedWorkspaceData(mergedScope)
	if err != nil {
		t.Fatalf("ExtractMergedWorkspaceData failed: %v", err)
	}

	// Verify structure
	cues, ok := workspaceData["cues"].([]any)
	if !ok {
		t.Fatal("Expected 'cues' array in workspace data")
	}

	if len(cues) != 2 {
		t.Errorf("Expected 2 cues, got %d", len(cues))
	}

	// Verify first cue
	cue1, ok := cues[0].(map[string]any)
	if !ok {
		t.Fatal("Expected cue to be a map")
	}

	if cue1["name"] != "Merged Cue 1" {
		t.Errorf("Expected cue name 'Merged Cue 1', got '%v'", cue1["name"])
	}

	if cue1["type"] != "audio" {
		t.Errorf("Expected cue type 'audio', got '%v'", cue1["type"])
	}

	t.Logf("Extracted merged workspace: %d cues", len(cues))
}

// TestNoConflictScenario tests that no conflicts are detected when all sources agree
func TestNoConflictScenario(t *testing.T) {
	workspace := &Workspace{}

	// All sources have identical data
	identicalCue := map[string]any{
		"uniqueID": "cue-1",
		"number":   "1.0",
		"name":     "Same Cue",
		"type":     "audio",
	}

	sourceData := map[string]any{"cues": []any{identicalCue}}
	cachedData := map[string]any{"cues": []any{identicalCue}}
	currentData := map[string]any{"cues": []any{identicalCue}}

	// Perform scope-based comparison
	scopeComparison, err := workspace.PerformScopeBasedComparison(sourceData, cachedData, currentData)
	if err != nil {
		t.Fatalf("PerformScopeBasedComparison failed: %v", err)
	}

	// Should have no changes
	if scopeComparison.HasChanges {
		t.Error("Expected no changes when all sources are identical")
	}

	if scopeComparison.ConflictExists {
		t.Error("Expected no conflicts when all sources are identical")
	}

	// Verify cue scope
	if len(scopeComparison.ChildScopes) > 0 {
		cueScope := scopeComparison.ChildScopes[0]
		if cueScope.ConflictExists {
			t.Error("Expected no conflicts in cue scope")
		}
	}

	t.Log("No conflict scenario validated successfully")
}
