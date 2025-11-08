package qlab

import (
	"fmt"
)

type ConflictResolutionChoice string

const (
	ChoiceUseSource ConflictResolutionChoice = "use_source"
	ChoiceKeepQLab  ConflictResolutionChoice = "keep_qlab"
	ChoiceSkip      ConflictResolutionChoice = "skip"
)

type ConflictResolutionRequest struct {
	Conflicts []CueConflict `json:"conflicts"`
	RequestID string        `json:"request_id"`
}

type ConflictResolutionResponse struct {
	RequestID   string                              `json:"request_id"`
	Resolutions map[string]ConflictResolutionChoice `json:"resolutions"`
}

type ConflictResolver interface {
	ResolveConflicts(conflicts []CueConflict) (map[string]ConflictResolutionChoice, error)
}

type InteractiveResolver struct {
	responseChannel chan ConflictResolutionResponse
	requestSender   func(ConflictResolutionRequest) error
}

func NewInteractiveResolver(requestSender func(ConflictResolutionRequest) error) *InteractiveResolver {
	return &InteractiveResolver{
		responseChannel: make(chan ConflictResolutionResponse, 1),
		requestSender:   requestSender,
	}
}

func (r *InteractiveResolver) ResolveConflicts(conflicts []CueConflict) (map[string]ConflictResolutionChoice, error) {
	requestID := fmt.Sprintf("conflict-req-%d", len(conflicts))

	request := ConflictResolutionRequest{
		Conflicts: conflicts,
		RequestID: requestID,
	}

	err := r.requestSender(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send conflict resolution request: %v", err)
	}

	response := <-r.responseChannel

	if response.RequestID != requestID {
		return nil, fmt.Errorf("request ID mismatch: expected %s, got %s", requestID, response.RequestID)
	}

	return response.Resolutions, nil
}

func (r *InteractiveResolver) SubmitResolution(response ConflictResolutionResponse) {
	r.responseChannel <- response
}

func ApplyResolutions(comparison *ThreeWayComparison, resolutions map[string]ConflictResolutionChoice) {
	for cueNumber, choice := range resolutions {
		result, exists := comparison.CueResults[cueNumber]
		if !exists {
			continue
		}

		switch choice {
		case ChoiceUseSource:
			result.Action = "update"
			result.Reason = "User chose to use source file version"

		case ChoiceKeepQLab:
			result.Action = "skip"
			result.Reason = "User chose to keep QLab version"
			if comparison.QLabChosenCues == nil {
				comparison.QLabChosenCues = make(map[string]bool)
			}
			comparison.QLabChosenCues[cueNumber] = true

		case ChoiceSkip:
			result.Action = "skip"
			result.Reason = "User chose to skip this cue"
		}

		comparison.CueResults[cueNumber] = result
	}
}
