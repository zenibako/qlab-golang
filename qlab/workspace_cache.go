package qlab

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
)

// queryCurrentWorkspaceState queries the current QLab workspace state for
// caching/comparison. It is the pure-data read the orchestrator (cli/qlabsync)
// drives the neutral engine against via ReadWorkspaceState. The three-way
// comparison that previously lived here has moved to cli/cuesync (TASK-273
// Phase D).
func (q *Workspace) queryCurrentWorkspaceState() (map[string]any, error) {
	// Try multiple approaches to get all cues in the workspace

	// Approach 1: Try /cueLists (should work if cue lists are Group cues with children)
	log.Info("Attempting to fetch cues using /cueLists")
	address := fmt.Sprintf("/workspace/%s/cueLists", q.workspace_id)
	reply := q.Send(address, "")

	if len(reply) == 0 {
		log.Warn("No reply received from /cueLists - QLab may be busy or disconnected")
		return nil, fmt.Errorf("no reply received from QLab when querying workspace state")
	}

	replyStr, ok := reply[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid reply format from QLab workspace query")
	}

	var replyData map[string]any
	err := json.Unmarshal([]byte(replyStr), &replyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse QLab workspace reply: %v", err)
	}

	// Check for error status - including timeout errors
	if status, ok := replyData["status"].(string); ok && status == "error" {
		// Check if this is a timeout error
		if errorMsg, hasError := replyData["error"].(string); hasError {
			if strings.Contains(errorMsg, "timeout") {
				log.Warn("QLab query timed out - workspace may be too large or QLab is busy")
				log.Info("Consider increasing timeout with SetTimeout() or reducing workspace size")
			}
		}
		return nil, formatErrorWithJSON("QLab error querying workspace state", replyStr)
	}

	// Check if we have cue lists with actual cue children
	data, ok := replyData["data"].([]any)
	if !ok {
		return replyData, nil // Return as-is if no data array
	}

	log.Info("Received cue lists data", "count", len(data))

	// Count total cues across all lists to see if we have actual cue data
	totalCues := 0
	for _, cueListInterface := range data {
		if cueList, ok := cueListInterface.(map[string]any); ok {
			if cuesArray, exists := cueList["cues"]; exists {
				if cues, ok := cuesArray.([]any); ok {
					totalCues += len(cues)
				}
			}
		}
	}

	log.Info("Total cues found in /cueLists", "count", totalCues)

	// If we found actual cues, enrich and return the data
	if totalCues > 0 {
		log.Info("Successfully retrieved cues using /cueLists")
		q.enrichCuesWithProperties(replyData)
		return replyData, nil
	}

	// Approach 2: DISABLED - /selectedCues approach has timeout issues
	// This approach doesn't work reliably with QLab and causes 20+ second delays
	log.Info("Skipping /selectedCues approach (disabled due to timeout issues)")

	// Approach 3: Try individual cue list traversal
	log.Info("Trying individual cue list traversal")

	for i, cueListInterface := range data {
		cueList, ok := cueListInterface.(map[string]any)
		if !ok {
			continue
		}

		// Check if this cue list already has cues
		if cuesArray, exists := cueList["cues"]; exists {
			if cues, ok := cuesArray.([]any); ok && len(cues) > 0 {
				log.Info("Cue list already has cues", "index", i, "count", len(cues))
				continue // This cue list already has cue data
			}
		}

		// Try to get either uniqueID or number for the cue list
		var cueIdentifier string
		var childrenAddress string

		// First, try uniqueID (preferred method)
		if uniqueID, exists := cueList["uniqueID"]; exists {
			if uniqueIDStr, ok := uniqueID.(string); ok && uniqueIDStr != "" {
				cueIdentifier = uniqueIDStr
				childrenAddress = fmt.Sprintf("/workspace/%s/cue_id/%s/children", q.workspace_id, uniqueIDStr)
				log.Info("Fetching cues for cue list", "index", i, "uniqueID", uniqueIDStr)
			}
		}

		// Fallback to number if uniqueID not available
		if cueIdentifier == "" {
			if listNumber, exists := cueList["number"]; exists {
				if listNumberStr, ok := listNumber.(string); ok && listNumberStr != "" {
					cueIdentifier = listNumberStr
					childrenAddress = fmt.Sprintf("/workspace/%s/cue/%s/children", q.workspace_id, listNumberStr)
					log.Info("Fetching cues for cue list", "index", i, "number", listNumberStr)
				}
			}
		}

		// Skip if no identifier found
		if cueIdentifier == "" {
			log.Warn("Cue list has no number or uniqueID", "index", i)
			continue
		}
		childrenReply := q.Send(childrenAddress, "")

		if len(childrenReply) == 0 {
			log.Warn("No reply received for cue list children", "identifier", cueIdentifier)
			continue
		}

		childrenStr, ok := childrenReply[0].(string)
		if !ok {
			log.Warn("Invalid reply format for cue list children", "identifier", cueIdentifier)
			continue
		}

		var childrenData map[string]any
		err := json.Unmarshal([]byte(childrenStr), &childrenData)
		if err != nil {
			log.Error("Failed to parse cue list children", "identifier", cueIdentifier, "error", err)
			continue
		}

		// Check for error status
		if status, ok := childrenData["status"].(string); ok && status == "error" {
			log.Error("QLab error fetching children for cue list", "identifier", cueIdentifier, "response", childrenStr)
			continue
		}

		// Extract the cues and add them to the cue list
		if childrenCues, ok := childrenData["data"].([]any); ok {
			cueList["cues"] = childrenCues
			log.Info("Successfully fetched cues for cue list", "identifier", cueIdentifier, "count", len(childrenCues))
		} else {
			log.Warn("No cues data found in children response for cue list", "identifier", cueIdentifier)
		}
	}

	// Enrich cues with additional properties not included in /cueLists
	q.enrichCuesWithProperties(replyData)

	// Return the enhanced workspace data
	return replyData, nil
}

// enrichCuesWithProperties queries additional cue properties not included in /cueLists response
// According to QLab OSC docs, /cueLists only returns: uniqueID, number, name, listName, type,
// colorName, flagged, armed. We need to query fileTarget and other properties separately.
func (q *Workspace) enrichCuesWithProperties(workspace map[string]any) {
	data, ok := workspace["data"].([]any)
	if !ok {
		return
	}

	for _, cueListData := range data {
		if cueList, ok := cueListData.(map[string]any); ok {
			if cues, ok := cueList["cues"].([]any); ok {
				q.enrichCueArrayWithProperties(cues)
			}
		}
	}
}

// enrichCueArrayWithProperties recursively enriches an array of cues with additional properties
func (q *Workspace) enrichCueArrayWithProperties(cues []any) {
	for _, cueData := range cues {
		if cue, ok := cueData.(map[string]any); ok {
			// Get uniqueID for property queries
			uniqueID, ok := cue["uniqueID"].(string)
			if !ok || uniqueID == "" {
				continue
			}

			// Query fileTarget property
			q.queryCueProperty(cue, uniqueID, "fileTarget")

			// Query cueTargetNumber property
			q.queryCueProperty(cue, uniqueID, "cueTargetNumber")

			// Recursively enrich child cues
			if children, ok := cue["cues"].([]any); ok {
				q.enrichCueArrayWithProperties(children)
			}
		}
	}
}

// queryCueProperty queries a single property from QLab and adds it to the cue map if not empty
func (q *Workspace) queryCueProperty(cue map[string]any, uniqueID, property string) {
	address := fmt.Sprintf("/workspace/%s/cue_id/%s/%s", q.workspace_id, uniqueID, property)
	reply := q.Send(address, "")
	log.Debug("Querying cue property", "uniqueID", uniqueID, "property", property, "reply_count", len(reply))
	if len(reply) > 0 {
		if replyStr, ok := reply[0].(string); ok {
			log.Debug("Got reply for property", "property", property, "reply", replyStr)
			var replyData map[string]any
			if err := json.Unmarshal([]byte(replyStr), &replyData); err == nil {
				if status, ok := replyData["status"].(string); ok && status == "ok" {
					if value, ok := replyData["data"].(string); ok && value != "" {
						cue[property] = value
						log.Debug("Enriched cue with property", "uniqueID", uniqueID, "property", property, "value", value)
					} else {
						log.Debug("Property value is empty or not a string", "property", property, "data", replyData["data"])
					}
				} else {
					log.Debug("Property query status not ok", "property", property, "status", status)
				}
			}
		}
	}
}
