package threeway

import "testing"

func cueWorkspace(number, name string) map[string]any {
	return map[string]any{
		"cues": []any{
			map[string]any{"number": number, "name": name, "type": "audio"},
		},
	}
}

// TestThreeWayOutcomes exercises the four outcomes a three-way comparison
// between source, cache, and a live destination (e.g. QLab) can produce:
// in-sync, source-modified, remote-modified, and a genuine conflict where
// both sides changed differently since the last sync.
func TestThreeWayOutcomes(t *testing.T) {
	tests := []struct {
		name                  string
		source, cache, remote map[string]any
		wantHasChanges        bool
		wantConflictCount     int
	}{
		{
			name:              "in sync: source, cache, and remote all match",
			source:            cueWorkspace("1", "Original Name"),
			cache:             cueWorkspace("1", "Original Name"),
			remote:            cueWorkspace("1", "Original Name"),
			wantHasChanges:    false,
			wantConflictCount: 0,
		},
		{
			name:              "source modified: cache still matches remote",
			source:            cueWorkspace("1", "Edited Locally"),
			cache:             cueWorkspace("1", "Original Name"),
			remote:            cueWorkspace("1", "Original Name"),
			wantHasChanges:    true,
			wantConflictCount: 0,
		},
		{
			name:              "remote modified: source still matches cache",
			source:            cueWorkspace("1", "Original Name"),
			cache:             cueWorkspace("1", "Original Name"),
			remote:            cueWorkspace("1", "Edited Remotely"),
			wantHasChanges:    true,
			wantConflictCount: 0,
		},
		{
			name:              "conflict: source and remote diverge from cache differently",
			source:            cueWorkspace("1", "Edited Locally"),
			cache:             cueWorkspace("1", "Original Name"),
			remote:            cueWorkspace("1", "Edited Remotely"),
			wantHasChanges:    true,
			wantConflictCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheMatchesRemote := CompareCacheWithCurrentState(tt.cache, tt.remote)

			scope, err := PerformScopeBasedComparison(tt.source, tt.cache, tt.remote)
			if err != nil {
				t.Fatalf("PerformScopeBasedComparison returned error: %v", err)
			}

			if scope.HasChanges != tt.wantHasChanges {
				t.Errorf("scope.HasChanges = %v, want %v", scope.HasChanges, tt.wantHasChanges)
			}

			comparison := &ThreeWayComparison{
				CueResults:       make(map[string]*CueChangeResult),
				HasCache:         true,
				HasQLabData:      true,
				CacheMatchesQLab: cacheMatchesRemote,
				WorkspaceScope:   scope,
			}

			conflicts, err := IdentifyConflicts(comparison)
			if err != nil {
				t.Fatalf("IdentifyConflicts returned error: %v", err)
			}

			if len(conflicts) != tt.wantConflictCount {
				t.Errorf("len(conflicts) = %d, want %d (conflicts: %+v)", len(conflicts), tt.wantConflictCount, conflicts)
			}
		})
	}
}

// TestApplyResolutionsAndMerge verifies that once a conflict is resolved,
// GenerateMergedScope/ExtractMergedWorkspaceData produce the chosen value.
func TestApplyResolutionsAndMerge(t *testing.T) {
	source := cueWorkspace("1", "Edited Locally")
	cache := cueWorkspace("1", "Original Name")
	remote := cueWorkspace("1", "Edited Remotely")

	scope, err := PerformScopeBasedComparison(source, cache, remote)
	if err != nil {
		t.Fatalf("PerformScopeBasedComparison returned error: %v", err)
	}

	comparison := &ThreeWayComparison{
		CueResults:       make(map[string]*CueChangeResult),
		HasCache:         true,
		HasQLabData:      true,
		CacheMatchesQLab: false,
		WorkspaceScope:   scope,
		QLabChosenCues:   map[string]bool{"1": true}, // user chose to keep the remote (QLab) version
	}

	merged, err := GenerateMergedScope(scope, comparison)
	if err != nil {
		t.Fatalf("GenerateMergedScope returned error: %v", err)
	}

	workspaceData, err := ExtractMergedWorkspaceData(merged)
	if err != nil {
		t.Fatalf("ExtractMergedWorkspaceData returned error: %v", err)
	}

	cues, ok := workspaceData["cues"].([]any)
	if !ok || len(cues) != 1 {
		t.Fatalf("expected 1 merged cue, got %+v", workspaceData["cues"])
	}

	mergedCue, ok := cues[0].(map[string]any)
	if !ok {
		t.Fatalf("expected merged cue to be a map, got %T", cues[0])
	}

	if got := mergedCue["name"]; got != "Edited Remotely" {
		t.Errorf("merged name = %v, want %q (user chose to keep remote version)", got, "Edited Remotely")
	}
}
