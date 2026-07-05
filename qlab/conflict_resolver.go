package qlab

import "github.com/zenibako/qlab-golang/threeway"

// Aliased to the destination-agnostic threeway package (TASK-261) so
// existing callers of the qlab package keep compiling unchanged.
type (
	ConflictResolutionChoice   = threeway.ConflictResolutionChoice
	ConflictResolutionRequest  = threeway.ConflictResolutionRequest
	ConflictResolutionResponse = threeway.ConflictResolutionResponse
	ConflictResolver           = threeway.ConflictResolver
	InteractiveResolver        = threeway.InteractiveResolver
)

const (
	ChoiceUseSource = threeway.ChoiceUseSource
	ChoiceKeepQLab  = threeway.ChoiceKeepQLab
	ChoiceSkip      = threeway.ChoiceSkip
)

func NewInteractiveResolver(requestSender func(ConflictResolutionRequest) error) *InteractiveResolver {
	return threeway.NewInteractiveResolver(requestSender)
}

func ApplyResolutions(comparison *ThreeWayComparison, resolutions map[string]ConflictResolutionChoice) {
	threeway.ApplyResolutions(comparison, resolutions)
}
