package qlab

// CueChangeResult represents the result of comparing a cue across three sources
type CueChangeResult struct {
	HasChanged     bool                      // Whether the cue needs to be updated
	Reason         string                    // Explanation of why it changed or didn't change
	ExistingID     string                    // ID of existing cue in QLab (if unchanged)
	Action         string                    // What action to take: "create", "update", "skip"
	ModifiedFields map[string]string         // Fields that differ: field_name -> "old_value -> new_value"
	CueID          string                    // QLab cue ID for traceability
	FieldConflicts map[string]*FieldConflict // Detailed field-level conflict information
	ScopeData      *ScopeComparison          // Scope-based comparison data
}

// ThreeWayComparison contains the results of comparing QLab workspace, cache, and source
type ThreeWayComparison struct {
	CueResults       map[string]*CueChangeResult // Map of cue number -> comparison result
	HasCache         bool                        // Whether cache was available
	HasQLabData      bool                        // Whether QLab data was available
	CacheMatchesQLab bool                        // Whether cache matches current QLab state
	QLabChosenCues   map[string]bool             // Cues where user chose "Keep QLab version"
	QLabChosenFields map[string]map[string]bool  // Fields where user chose "Keep QLab version": cue -> field -> bool
	CurrentQLabData  map[string]any              // Current QLab workspace data for source file updates
	WorkspaceScope   *ScopeComparison            // Workspace-level scope comparison
	MergedResult     *MergedScope                // Final merged result after conflict resolution
}
