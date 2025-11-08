package qlab

import (
	"github.com/charmbracelet/log"
)

// GetRunningCueNumbers extracts cue numbers from running cues
func GetRunningCueNumbers(runningCues []map[string]any) []string {
	numbers := make([]string, 0, len(runningCues))
	for _, cue := range runningCues {
		if cueNumber, ok := cue["number"].(string); ok {
			numbers = append(numbers, cueNumber)
		}
	}
	return numbers
}

// GetSelectedCueNumbers extracts cue numbers from selected cues
func GetSelectedCueNumbers(selectedCues []map[string]any) []string {
	numbers := make([]string, 0, len(selectedCues))
	for _, cue := range selectedCues {
		if cueNumber, ok := cue["number"].(string); ok {
			numbers = append(numbers, cueNumber)
		}
	}
	return numbers
}

// SetupUpdateListener sets up a listener that calls the handler when QLab sends updates
func SetupUpdateListener(workspace *Workspace, handler func()) error {
	return workspace.StartUpdateListener(func(address string, args []any) {
		log.Debug("QLab update received", "address", address)
		handler()
	})
}
