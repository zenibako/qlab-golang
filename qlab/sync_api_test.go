package qlab

import (
	"os"
	"path/filepath"
	"testing"
)

// TestApplyCueChanges_DryRun exercises all three change actions through the
// dry-run path (writes are mocked), so it needs no live OSC server.
func TestApplyCueChanges_DryRun(t *testing.T) {
	w := NewTestWorkspace("localhost", 53535, "test-ws")
	w.SetDryRun(true)

	changes := []CueChange{
		{Action: CueCreate, Number: "1.0", Cue: Cue{
			Number: "1.0", Type: "audio", Name: "New Audio",
		}},
		{Action: CueUpdate, Number: "2.0", UniqueID: "cue-2", Cue: Cue{
			Name: "Renamed",
		}},
		{Action: CueDelete, Number: "3.0", UniqueID: "cue-3"},
	}

	applied, err := w.ApplyCueChanges(changes)
	if err != nil {
		t.Fatalf("ApplyCueChanges: %v", err)
	}
	if applied != 3 {
		t.Errorf("applied = %d, want 3", applied)
	}
}

// TestApplyCueChanges_ParentPlacement exercises non-recursive create plus
// placing a child under a parent created earlier in the same batch (the child's
// ParentNumber resolves to the parent's freshly created uniqueID). Writes are
// mocked via dry-run.
func TestApplyCueChanges_ParentPlacement(t *testing.T) {
	w := NewTestWorkspace("localhost", 53535, "test-ws")
	w.SetDryRun(true)

	changes := []CueChange{
		{Action: CueCreate, Number: "1", Cue: Cue{Number: "1", Type: "group", Name: "G"}},
		{Action: CueCreate, Number: "1.1", ParentNumber: "1", Cue: Cue{Number: "1.1", Type: "audio", Name: "child"}},
	}

	applied, err := w.ApplyCueChanges(changes)
	if err != nil {
		t.Fatalf("ApplyCueChanges: %v", err)
	}
	if applied != 2 {
		t.Errorf("applied = %d, want 2 (parent + nested child)", applied)
	}
}

// TestSaveLoadCacheRoundTrip verifies the merge-base snapshot survives a
// save/load cycle. It writes under the real per-user cache dir (matching the
// legacy convention) with a unique base name and cleans up after.
func TestSaveLoadCacheRoundTrip(t *testing.T) {
	w := NewTestWorkspace("localhost", 53535, "test-ws")

	cueFile := filepath.Join(t.TempDir(), "roundtrip-"+t.Name()+".cue")
	t.Cleanup(func() {
		if dir, base, err := cacheDirForCueFile(cueFile); err == nil {
			matches, _ := filepath.Glob(filepath.Join(dir, base+"_*.json"))
			for _, m := range matches {
				_ = os.Remove(m)
			}
		}
	})

	want := map[string]any{
		"cues": []any{
			map[string]any{"number": "1.0", "uniqueID": "U1", "name": "A"},
		},
	}
	if err := w.SaveCache(cueFile, want); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}

	got, err := w.LoadCache(cueFile)
	if err != nil {
		t.Fatalf("LoadCache: %v", err)
	}
	cues, ok := got["cues"].([]any)
	if !ok || len(cues) != 1 {
		t.Fatalf("LoadCache round-trip: expected 1 cue, got %#v", got)
	}
	cue, _ := cues[0].(map[string]any)
	if cue["uniqueID"] != "U1" || cue["number"] != "1.0" {
		t.Errorf("round-trip mismatch: %#v", cue)
	}
}

// TestLoadCacheMissing returns (nil, nil) when no snapshot exists yet.
func TestLoadCacheMissing(t *testing.T) {
	w := NewTestWorkspace("localhost", 53535, "test-ws")
	cueFile := filepath.Join(t.TempDir(), "never-saved-"+t.Name()+".cue")
	got, err := w.LoadCache(cueFile)
	if err != nil {
		t.Fatalf("LoadCache: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil cache for a file with no snapshot, got %#v", got)
	}
}

// TestApplyCueChanges_MissingUniqueID guards update/delete without a target.
func TestApplyCueChanges_MissingUniqueID(t *testing.T) {
	w := NewTestWorkspace("localhost", 53535, "test-ws")
	w.SetDryRun(true)

	if _, err := w.ApplyCueChanges([]CueChange{{Action: CueUpdate, Number: "1.0"}}); err == nil {
		t.Error("expected error updating a cue with no uniqueID")
	}
	if _, err := w.ApplyCueChanges([]CueChange{{Action: CueDelete, Number: "1.0"}}); err == nil {
		t.Error("expected error deleting a cue with no uniqueID")
	}
}

// TestApplyCueChanges_UnknownAction rejects an unrecognized action.
func TestApplyCueChanges_UnknownAction(t *testing.T) {
	w := NewTestWorkspace("localhost", 53535, "test-ws")
	w.SetDryRun(true)
	if _, err := w.ApplyCueChanges([]CueChange{{Action: "frobnicate", Number: "1.0"}}); err == nil {
		t.Error("expected error for unknown change action")
	}
}

// TestReadWorkspaceState queries the mock OSC server and returns a cues map.
func TestReadWorkspaceState(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("getFreePort: %v", err)
	}
	mockServer := NewMockOSCServer("localhost", port)
	if err := mockServer.Start(); err != nil {
		t.Fatalf("mock server start: %v", err)
	}
	defer func() { _ = mockServer.Stop() }()

	w := NewTestWorkspace("localhost", port, mockServer.GetWorkspaceID())
	if _, err := w.Init(""); err != nil {
		t.Fatalf("Init: %v", err)
	}

	state, err := w.ReadWorkspaceState()
	if err != nil {
		t.Fatalf("ReadWorkspaceState: %v", err)
	}
	if _, ok := state["cues"].([]any); !ok {
		t.Errorf("ReadWorkspaceState should return a {\"cues\": []any} map, got %#v", state)
	}
}
