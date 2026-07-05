package qlab

import "github.com/zenibako/qlab-golang/threeway"

// These types are defined in the destination-agnostic threeway package (see
// TASK-261 in the cuejitsu backlog) so that the three-way diff/conflict
// engine can be reused by non-QLab destinations. Aliased here so existing
// callers of the qlab package keep compiling unchanged.
type (
	ConflictType    = threeway.ConflictType
	ConflictScope   = threeway.ConflictScope
	FieldConflict   = threeway.FieldConflict
	CueConflict     = threeway.CueConflict
	ScopeComparison = threeway.ScopeComparison
	MergedScope     = threeway.MergedScope
)

const (
	ConflictThreeWayDivergence = threeway.ConflictThreeWayDivergence
	ConflictCacheStale         = threeway.ConflictCacheStale
	ConflictSourceModified     = threeway.ConflictSourceModified

	ScopeWorkspace = threeway.ScopeWorkspace
	ScopeCueList   = threeway.ScopeCueList
	ScopeCue       = threeway.ScopeCue
	ScopeField     = threeway.ScopeField
)
