package threeway

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
)

// IndexCuesFromWorkspace extracts a flat map of cue number/identifier -> cue
// data from any of the workspace data shapes produced by the source CUE
// parser, a cache file, or a live destination read.
func IndexCuesFromWorkspace(workspace map[string]any) map[string]map[string]any {
	cueIndex := make(map[string]map[string]any)

	if workspace == nil {
		return cueIndex
	}

	// Temporary debug: print workspace structure to understand what we're receiving
	keys := make([]string, 0, len(workspace))
	for k := range workspace {
		keys = append(keys, k)
	}
	log.Debug("Workspace keys found", "keys", keys)

	// Extract cue lists from workspace data structure
	var cuesData []any

	// Handle different workspace data structures
	if cues, ok := workspace["cues"].([]any); ok {
		// Direct cues array (source CUE format)
		cuesData = cues
		log.Debug("Found cues via direct cues array", "cue_count", len(cuesData))
	} else if workspaceData, ok := workspace["workspace"].(map[string]any); ok {
		// Nested workspace structure (parsed CUE file format)
		if cues, ok := workspaceData["cues"].([]any); ok {
			cuesData = cues
			log.Debug("Found cues via nested workspace structure", "cue_count", len(cuesData))
		}
	} else if data, ok := workspace["data"].(map[string]any); ok {
		// QLab response format with data wrapper containing cueLists key
		log.Debug("Found data map, checking for cueLists")
		if cueLists, ok := data["cueLists"].([]any); ok {
			log.Debug("Found cueLists in data map", "cue_list_count", len(cueLists))
			// Extract cues from cue lists
			for _, cueListData := range cueLists {
				if cueList, ok := cueListData.(map[string]any); ok {
					if listCues, ok := cueList["cues"].([]any); ok {
						cuesData = append(cuesData, listCues...)
						log.Debug("Added cues from cueList", "cue_count", len(listCues))
					}
				}
			}
		}
		// Also check for direct cues array in data
		if directCues, ok := data["cues"].([]any); ok {
			cuesData = append(cuesData, directCues...)
			log.Debug("Added direct cues from data", "cue_count", len(directCues))
		}
	} else if cueLists, ok := workspace["data"].([]any); ok {
		// QLab response format where data is directly an array of cue lists
		log.Debug("Found data array with cueLists", "cue_list_count", len(cueLists))
		for i, cueListData := range cueLists {
			if cueList, ok := cueListData.(map[string]any); ok {
				// Debug: show keys in each cueList
				listKeys := make([]string, 0, len(cueList))
				for k := range cueList {
					listKeys = append(listKeys, k)
				}
				log.Debug("CueList keys found", "index", i, "keys", listKeys)

				if cuesValue, exists := cueList["cues"]; exists {
					log.Debug("CueList cues value found", "index", i, "type", fmt.Sprintf("%T", cuesValue))
					if listCues, ok := cuesValue.([]any); ok {
						cuesData = append(cuesData, listCues...)
						log.Debug("Added cues from cueList array", "cue_count", len(listCues))
					} else {
						log.Debug("CueList cues exists but wrong type", "index", i, "type", fmt.Sprintf("%T", cuesValue))
					}
				} else {
					log.Debug("CueList has no cues key", "index", i)
				}
			}
		}
	}

	// Recursively index all cues
	log.Debug("Processing total cues for indexing", "cue_count", len(cuesData))
	indexCuesRecursively(cuesData, "", cueIndex)
	log.Debug("Final cue index complete", "entry_count", len(cueIndex))

	return cueIndex
}

// indexCuesRecursively recursively indexes cues, handling nested cue structures
func indexCuesRecursively(cuesData []any, parentNumber string, cueIndex map[string]map[string]any) {
	for i, cueData := range cuesData {
		cue, ok := cueData.(map[string]any)
		if !ok {
			continue
		}

		// Extract cue number
		var cueNumber string
		if num, ok := cue["number"]; ok && num != nil {
			switch v := num.(type) {
			case string:
				cueNumber = v
			case float64:
				if v == float64(int64(v)) && v >= 0 && v <= 999 {
					cueNumber = fmt.Sprintf("%.1f", v)
				} else {
					cueNumber = fmt.Sprintf("%g", v)
				}
			case int64:
				cueNumber = fmt.Sprintf("%d", v)
			case int:
				cueNumber = fmt.Sprintf("%d", v)
			default:
				cueNumber = fmt.Sprintf("%v", v)
			}
		}

		// Build full cue number with parent prefix (same logic as processing)
		fullNumber := cueNumber
		if parentNumber != "" && cueNumber != "" {
			if strings.Contains(cueNumber, ".") {
				fullNumber = cueNumber
			} else {
				fullNumber = parentNumber + "." + cueNumber
			}
		}

		// Add to index if we have a number
		if fullNumber != "" {
			cueIndex[fullNumber] = cue
		} else {
			// Fallback: use position-based identification for cues without numbers
			// Include parent context, cue name, and position to create unique identifier
			cueName, _ := cue["name"].(string)
			cueType, _ := cue["type"].(string)

			// Create composite key: parent@position[type:name]
			// Normalize type to lowercase for consistent matching between source and QLab data
			normalizedType := strings.ToLower(cueType)
			var positionKey string
			if parentNumber != "" {
				positionKey = fmt.Sprintf("%s@%d[%s:%s]", parentNumber, i, normalizedType, cueName)
			} else {
				positionKey = fmt.Sprintf("@%d[%s:%s]", i, normalizedType, cueName)
			}

			// Only index if we have enough identifying information
			if cueType != "" || cueName != "" {
				cueIndex[positionKey] = cue
				log.Debug("Indexed cue by position", "position_key", positionKey, "parent", parentNumber, "index", i, "type", cueType, "name", cueName)
			}
		}

		// Process sub-cues recursively
		if subCues, ok := cue["cues"].([]any); ok {
			indexCuesRecursively(subCues, fullNumber, cueIndex)
		}
	}
}
