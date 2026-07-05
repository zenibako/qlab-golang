package qlab

import "github.com/zenibako/qlab-golang/threeway"

// Aliased to the destination-agnostic threeway package (TASK-261) so
// existing callers of the qlab package keep compiling unchanged.
type (
	CueChangeResult    = threeway.CueChangeResult
	ThreeWayComparison = threeway.ThreeWayComparison
)
