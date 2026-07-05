package qlab

import "github.com/zenibako/qlab-golang/threeway"

// PerformScopeBasedComparison performs a hierarchical scope-based comparison.
// Delegates to the destination-agnostic threeway package (TASK-261).
func (q *Workspace) PerformScopeBasedComparison(sourceCueData, cachedCueData, currentQLabData map[string]any) (*ScopeComparison, error) {
	return threeway.PerformScopeBasedComparison(sourceCueData, cachedCueData, currentQLabData)
}

// GenerateMergedScope creates a merged result after conflict resolution.
// Delegates to the destination-agnostic threeway package (TASK-261).
func (q *Workspace) GenerateMergedScope(scopeComparison *ScopeComparison, comparison *ThreeWayComparison) (*MergedScope, error) {
	return threeway.GenerateMergedScope(scopeComparison, comparison)
}

// ExtractMergedWorkspaceData extracts the final merged workspace data structure.
// Delegates to the destination-agnostic threeway package (TASK-261).
func (q *Workspace) ExtractMergedWorkspaceData(mergedScope *MergedScope) (map[string]any, error) {
	return threeway.ExtractMergedWorkspaceData(mergedScope)
}
