package qlab

import (
	"testing"
)

// TestThreeWayComparisonDemoWithOutput demonstrates the three-way comparison output
func TestThreeWayComparisonDemoWithOutput(t *testing.T) {
	// Create a workspace for the demo
	workspace := &Workspace{}

	// Create sample source data with multiple cues
	sourceData := map[string]any{
		"cues": []any{
			map[string]any{
				"type":   "audio",
				"number": "1.0",
				"name":   "Opening Music",
			},
			map[string]any{
				"type":   "light",
				"number": "2.0",
				"name":   "Stage Lights Up",
			},
			map[string]any{
				"type":   "memo",
				"number": "3.0",
				"name":   "Intermission Note",
			},
		},
	}

	// Create cached data (representing previous state)
	cachedData := map[string]any{
		"data": map[string]any{
			"cueLists": []any{
				map[string]any{
					"cues": []any{
						map[string]any{
							"type":     "audio",
							"number":   "1.0",
							"name":     "Opening Music", // Same as source
							"uniqueID": "cached-audio-1",
						},
						map[string]any{
							"type":     "light",
							"number":   "2.0",
							"name":     "Old Light Name", // Different from source
							"uniqueID": "cached-light-2",
						},
					},
				},
			},
		},
	}

	// Create current QLab data (representing current QLab state)
	currentData := map[string]any{
		"data": map[string]any{
			"cueLists": []any{
				map[string]any{
					"cues": []any{
						map[string]any{
							"type":     "audio",
							"number":   "1.0",
							"name":     "Opening Music", // Same as cache and source
							"uniqueID": "qlab-audio-1",
						},
						map[string]any{
							"type":     "light",
							"number":   "2.0",
							"name":     "Modified Light Name", // Different from both source and cache
							"uniqueID": "qlab-light-2",
						},
					},
				},
			},
		},
	}

	// Create a comparison manually for demonstration
	comparison := &ThreeWayComparison{
		CueResults:       make(map[string]*CueChangeResult),
		HasCache:         true,
		HasQLabData:      true,
		CacheMatchesQLab: false, // Cache and QLab differ
	}

	// Index cues from each source
	sourceCues := workspace.indexCuesFromWorkspace(sourceData)
	cachedCues := workspace.indexCuesFromWorkspace(cachedData)
	currentCues := workspace.indexCuesFromWorkspace(currentData)

	// Build comparison results for each source cue
	for cueNumber, sourceCue := range sourceCues {
		result := &CueChangeResult{
			HasChanged: true,
			Action:     "create",
			Reason:     "new cue",
		}

		// Check if cue exists in current QLab state
		if currentCue, existsInQLab := currentCues[cueNumber]; existsInQLab {
			if id, ok := currentCue["uniqueID"].(string); ok {
				result.ExistingID = id
				result.CueID = id
			}

			// Check if cue exists in cache
			if cachedCue, existsInCache := cachedCues[cueNumber]; existsInCache {
				// Enhanced three-way comparison logic with detailed differences
				sourceCacheDiffs := workspace.compareCuePropertiesDetailed(sourceCue, cachedCue)
				cacheCurrentDiffs := workspace.compareCuePropertiesDetailed(cachedCue, currentCue)
				sourceMatchesCache := len(sourceCacheDiffs) == 0
				cacheMatchesCurrent := len(cacheCurrentDiffs) == 0

				if sourceMatchesCache && cacheMatchesCurrent {
					// Source == Cache == Current
					result.HasChanged = false
					result.Action = "skip"
					result.Reason = "unchanged since last transmission"
					result.ModifiedFields = make(map[string]string)
				} else if sourceMatchesCache && !cacheMatchesCurrent {
					// Source == Cache != Current
					result.HasChanged = true
					result.Action = "update"
					result.Reason = "QLab modified externally, reverting to source"
					result.ModifiedFields = cacheCurrentDiffs
				} else if !sourceMatchesCache && cacheMatchesCurrent {
					// Source != Cache == Current
					result.HasChanged = true
					result.Action = "update"
					result.Reason = "source file modified"
					result.ModifiedFields = sourceCacheDiffs
				} else {
					// Source != Cache != Current (three-way divergence)
					result.HasChanged = true
					result.Action = "update"
					result.Reason = "both source and QLab modified (three-way conflict)"
					// Merge both difference sets for complete visibility
					result.ModifiedFields = make(map[string]string)
					for field, diff := range sourceCacheDiffs {
						result.ModifiedFields["source_vs_cache_"+field] = diff
					}
					for field, diff := range cacheCurrentDiffs {
						result.ModifiedFields["cache_vs_current_"+field] = diff
					}
				}
			} else {
				// Exists in QLab but not in cache
				sourceCurrentDiffs := workspace.compareCuePropertiesDetailed(sourceCue, currentCue)
				if len(sourceCurrentDiffs) == 0 {
					result.HasChanged = false
					result.Action = "skip"
					result.Reason = "matches current QLab state"
					result.ModifiedFields = make(map[string]string)
				} else {
					result.HasChanged = true
					result.Action = "update"
					result.Reason = "differs from current QLab state"
					result.ModifiedFields = sourceCurrentDiffs
				}
			}
		} else {
			// New cue
			result.ModifiedFields = make(map[string]string)
		}

		comparison.CueResults[cueNumber] = result
	}

	// Add a cue that only exists in source (new cue scenario)
	comparison.CueResults["3.0"] = &CueChangeResult{
		HasChanged: true,
		Action:     "create",
		Reason:     "new cue",
	}

	// Print the detailed three-way comparison results
	workspace.PrintThreeWayComparisonResults(comparison)

	t.Logf("Demo completed successfully - check the log output above for three-way comparison details")
}
