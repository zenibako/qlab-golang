package threeway

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
)

// IdentifyConflicts inspects a completed ThreeWayComparison and returns the
// set of conflicts that require user resolution.
func IdentifyConflicts(comparison *ThreeWayComparison) ([]CueConflict, error) {
	var conflicts []CueConflict

	// Handle case where QLab query failed
	if !comparison.HasQLabData {
		if comparison.HasCache {
			log.Warn("QLab data unavailable - using cache-only comparison")
			log.Info("Conflicts cannot be detected without current QLab state")
			log.Info("Recommendation: Increase timeout or check QLab connection")
		}
		return conflicts, nil
	}

	// Only identify conflicts if we have cache (need common ancestor)
	if !comparison.HasCache {
		log.Debug("No cache available - three-way conflict detection unavailable")
		return conflicts, nil
	}

	// If cache matches QLab, then only simple source vs cache conflicts are possible
	// These are typically handled automatically, so we don't need user input
	if comparison.CacheMatchesQLab {
		log.Debug("Cache matches QLab state, no complex conflicts detected")
		return conflicts, nil
	}

	// Use scope-based conflict identification if available
	if comparison.WorkspaceScope != nil {
		return identifyConflictsFromScope(comparison.WorkspaceScope), nil
	}

	// Fallback to legacy cue-level conflict detection
	for cueNumber, result := range comparison.CueResults {
		if result == nil {
			continue
		}

		// Look for cases where manual intervention might be needed
		// This occurs when QLab was modified externally or when both source and QLab differ
		if result.Action == "update" && (strings.Contains(result.Reason, "QLab modified externally") || strings.Contains(result.Reason, "both source and QLab modified")) {
			var conflictType ConflictType
			var description string

			if strings.Contains(result.Reason, "QLab modified externally") {
				conflictType = ConflictCacheStale
				description = fmt.Sprintf("Cue %s has been modified in QLab since last sync", cueNumber)
			} else {
				conflictType = ConflictThreeWayDivergence
				description = fmt.Sprintf("Cue %s has been modified in both the source file and QLab since last sync", cueNumber)
			}

			conflict := CueConflict{
				CueNumber:      cueNumber,
				CueIdentifier:  cueNumber,
				ConflictType:   conflictType,
				Scope:          ScopeCue,
				Description:    description,
				FieldConflicts: result.FieldConflicts,
				Resolved:       false,
			}
			conflicts = append(conflicts, conflict)
			log.Debug("Identified conflict for cue", "cue_number", cueNumber, "type", conflictType)
		}
	}

	return conflicts, nil
}

// identifyConflictsFromScope recursively identifies conflicts from scope comparison
func identifyConflictsFromScope(scope *ScopeComparison) []CueConflict {
	var conflicts []CueConflict

	if scope == nil {
		return conflicts
	}

	// Check if this scope has conflicts
	if scope.ConflictExists {
		// Build list of conflicting properties
		properties := make([]string, 0, len(scope.FieldChanges))
		fieldConflicts := make(map[string]*FieldConflict)

		for fieldName, fieldConflict := range scope.FieldChanges {
			if isFieldConflict(fieldConflict) {
				properties = append(properties, fieldName)
				fieldConflicts[fieldName] = fieldConflict
			}
		}

		if len(properties) > 0 {
			var conflictType ConflictType
			var description string

			// Determine conflict type based on field changes
			hasSourceChanges := false
			hasQLabChanges := false

			for _, fc := range fieldConflicts {
				sourceNorm := normalizeProperty(fc.SourceValue)
				cacheNorm := normalizeProperty(fc.CacheValue)
				qlabNorm := normalizeProperty(fc.QLabValue)

				if !comparePropertyValues(fc.FieldName, sourceNorm, cacheNorm) {
					hasSourceChanges = true
				}
				if !comparePropertyValues(fc.FieldName, qlabNorm, cacheNorm) {
					hasQLabChanges = true
				}
			}

			if hasSourceChanges && hasQLabChanges {
				conflictType = ConflictThreeWayDivergence
				description = fmt.Sprintf("%s '%s' has conflicting changes in source and QLab (fields: %v)",
					scope.Scope, scope.Identifier, properties)
			} else if hasQLabChanges {
				conflictType = ConflictCacheStale
				description = fmt.Sprintf("%s '%s' modified in QLab (fields: %v)",
					scope.Scope, scope.Identifier, properties)
			} else if hasSourceChanges {
				conflictType = ConflictSourceModified
				description = fmt.Sprintf("%s '%s' modified in source (fields: %v)",
					scope.Scope, scope.Identifier, properties)
			}

			conflict := CueConflict{
				CueNumber:      scope.Identifier,
				CueIdentifier:  scope.Identifier,
				ConflictType:   conflictType,
				Scope:          scope.Scope,
				Properties:     properties,
				FieldConflicts: fieldConflicts,
				Description:    description,
				Resolved:       false,
			}

			conflicts = append(conflicts, conflict)
			log.Debugf("Identified %s-level conflict: %s (%d fields)", scope.Scope, scope.Identifier, len(properties))
		}
	}

	// Recursively check child scopes
	for _, childScope := range scope.ChildScopes {
		childConflicts := identifyConflictsFromScope(childScope)
		conflicts = append(conflicts, childConflicts...)
	}

	return conflicts
}

// IdentifyConflictsFromScope is the exported form of identifyConflictsFromScope,
// for callers outside this package (e.g. the qlab.Workspace delegating wrapper).
func IdentifyConflictsFromScope(scope *ScopeComparison) []CueConflict {
	return identifyConflictsFromScope(scope)
}
