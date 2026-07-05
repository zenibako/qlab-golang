package qlab

// ConflictType represents the type of conflict detected
type ConflictType string

const (
	ConflictThreeWayDivergence ConflictType = "three_way_divergence" // Source ≠ Cache ≠ QLab
	ConflictCacheStale         ConflictType = "cache_stale"          // Cache ≠ QLab but Source = Cache
	ConflictSourceModified     ConflictType = "source_modified"      // Source ≠ Cache but Cache = QLab
)

// ConflictScope represents the level at which a conflict occurs
type ConflictScope string

const (
	ScopeWorkspace ConflictScope = "workspace" // Workspace-level changes (structure, cue lists)
	ScopeCueList   ConflictScope = "cuelist"   // Cue list-level changes
	ScopeCue       ConflictScope = "cue"       // Cue-level changes
	ScopeField     ConflictScope = "field"     // Field-level changes
)

// FieldConflict represents a conflict at the field level
type FieldConflict struct {
	FieldName    string // Name of the conflicting field
	SourceValue  any    // Value in source
	CacheValue   any    // Value in cache
	QLabValue    any    // Value in QLab
	ChosenValue  any    // Value chosen after resolution (nil if not resolved)
	ChosenSource string // Which source was chosen: "source", "qlab", "cache", or "custom"
}

// CueConflict represents a conflict that needs user resolution
type CueConflict struct {
	CueNumber      string                    // Cue number that has the conflict
	CueIdentifier  string                    // Full identifier (may be position-based)
	ConflictType   ConflictType              // Type of conflict
	Scope          ConflictScope             // Scope of the conflict
	SourceData     map[string]any            // Source cue data
	CacheData      map[string]any            // Cached cue data (may be nil)
	QLabData       map[string]any            // Current QLab cue data (may be nil)
	Properties     []string                  // List of conflicting properties
	FieldConflicts map[string]*FieldConflict // Detailed field-level conflicts
	Description    string                    // Human-readable description
	Resolved       bool                      // Whether conflict has been resolved
}

// ScopeComparison represents changes detected within a specific scope
type ScopeComparison struct {
	Scope          ConflictScope             // The scope being compared
	Identifier     string                    // Identifier for this scope (cue number, cue list name, etc.)
	HasChanges     bool                      // Whether changes were detected
	ChangeType     string                    // Type of change: "create", "update", "delete", "none"
	FieldChanges   map[string]*FieldConflict // Field-level changes
	ChildScopes    []*ScopeComparison        // Nested scopes (e.g., cues within a cue list)
	ConflictExists bool                      // Whether unresolved conflicts exist
	Resolved       bool                      // Whether all conflicts resolved
}

// MergedScope represents the final merged state after conflict resolution
type MergedScope struct {
	Scope        ConflictScope     // The scope of this merged result
	Identifier   string            // Identifier for this scope
	MergedData   map[string]any    // The final merged data
	ChildScopes  []*MergedScope    // Nested merged scopes
	SourceFields map[string]string // Tracking: field -> source ("source", "qlab", "cache")
	AppliedAt    string            // Timestamp when merge was applied
}
