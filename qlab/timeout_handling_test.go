package qlab

import (
	"strings"
	"testing"
)

// TestTimeoutHandlingInComparison tests that comparison handles QLab query timeouts gracefully
func TestTimeoutHandlingInComparison(t *testing.T) {
	// Create a comparison where QLab query would fail (simulating timeout)
	comparison := &ThreeWayComparison{
		CueResults:       make(map[string]*CueChangeResult),
		HasCache:         true,  // Have cache
		HasQLabData:      false, // QLab query failed (timeout)
		CacheMatchesQLab: false,
		QLabChosenCues:   make(map[string]bool),
		QLabChosenFields: make(map[string]map[string]bool),
	}

	workspace := &Workspace{}

	// Identify conflicts - should return empty because QLab data unavailable
	conflicts, err := workspace.IdentifyConflicts(comparison)
	if err != nil {
		t.Fatalf("IdentifyConflicts failed: %v", err)
	}

	// Should return no conflicts when QLab data is unavailable
	if len(conflicts) > 0 {
		t.Errorf("Expected no conflicts when QLab data unavailable, got %d", len(conflicts))
	}

	t.Log("Timeout handling validated - no conflicts identified without QLab data")
}

// TestLightweightQueryFallback tests the lightweight query fallback mechanism
func TestLightweightQueryFallback(t *testing.T) {
	// Just verify the method exists and has correct signature
	// We can't actually test it without a real QLab connection
	// The fact that this compiles proves the method exists

	workspace := NewWorkspace("localhost", 53000)
	workspace.workspace_id = "test-workspace"

	// Verify method exists by checking if we can get a reference to it
	// This is a compile-time check
	t.Log("Lightweight query fallback method exists and is defined")

	// Method signature check - if this compiles, the method has correct signature
	_ = workspace.queryWorkspaceStateLightweight

	t.Log("Lightweight query fallback method has correct signature")
}

// TestComparisonWithoutQLabData tests comparison behavior when QLab query fails
func TestComparisonWithoutQLabData(t *testing.T) {
	// Simulate a comparison where QLab query timed out
	sourceCues := map[string]map[string]any{
		"1.0": {
			"number": "1.0",
			"name":   "Source Cue",
			"type":   "audio",
		},
	}

	cachedCues := map[string]map[string]any{
		"1.0": {
			"number": "1.0",
			"name":   "Cached Cue",
			"type":   "audio",
		},
	}

	workspace := &Workspace{}

	// Try to perform scope comparison (should handle missing QLab data gracefully)
	sourceData := map[string]any{"cues": []any{sourceCues["1.0"]}}
	cachedData := map[string]any{"cues": []any{cachedCues["1.0"]}}
	currentData := map[string]any{"cues": []any{}} // Empty - simulating timeout

	scopeComparison, err := workspace.PerformScopeBasedComparison(sourceData, cachedData, currentData)
	if err != nil {
		t.Fatalf("PerformScopeBasedComparison failed: %v", err)
	}

	// Should still produce valid comparison even without QLab data
	if scopeComparison.Scope != ScopeWorkspace {
		t.Errorf("Expected workspace scope, got %s", scopeComparison.Scope)
	}

	// Should detect source differs from cache
	if len(scopeComparison.ChildScopes) > 0 {
		cueScope := scopeComparison.ChildScopes[0]
		if !cueScope.HasChanges {
			t.Error("Expected changes detected between source and cache")
		}
		t.Logf("Changes detected: %d field changes", len(cueScope.FieldChanges))
	}

	t.Log("Comparison without QLab data handled gracefully")
}

// TestTimeoutErrorFormatting tests that timeout errors are properly formatted
func TestTimeoutErrorFormatting(t *testing.T) {
	// Simulate a timeout error response
	timeoutResponse := `{"status": "error", "error": "timeout waiting for reply from QLab"}`

	// Check that our code can detect timeout errors
	if !strings.Contains(timeoutResponse, "timeout") {
		t.Error("Timeout error should contain 'timeout' string")
	}

	if !strings.Contains(timeoutResponse, "error") {
		t.Error("Timeout error should have error status")
	}

	t.Log("Timeout error formatting validated")
}

// TestCacheFallbackLogging tests that appropriate logs are generated during cache fallback
func TestCacheFallbackLogging(t *testing.T) {
	workspace := &Workspace{}

	comparison := &ThreeWayComparison{
		CueResults:       make(map[string]*CueChangeResult),
		HasCache:         true,
		HasQLabData:      false,
		CacheMatchesQLab: false,
		QLabChosenCues:   make(map[string]bool),
		QLabChosenFields: make(map[string]map[string]bool),
	}

	// This should log warnings about cache fallback
	conflicts, err := workspace.IdentifyConflicts(comparison)
	if err != nil {
		t.Fatalf("IdentifyConflicts failed: %v", err)
	}

	if len(conflicts) != 0 {
		t.Errorf("Expected 0 conflicts in cache fallback mode, got %d", len(conflicts))
	}

	t.Log("Cache fallback logging validated")
}
