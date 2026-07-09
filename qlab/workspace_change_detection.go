package qlab

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/zenibako/qlab-golang/messages"
)

// createCue creates a new cue of the given type in the workspace and sets its
// properties from cueData. It is a pure-data OSC primitive called by
// ApplyCueChanges (sync_api.go) via the cli/qlabsync orchestrator. The legacy
// three-way diff/conflict/merge orchestration that previously shared this file
// has moved to cli/cuesync (TASK-273 Phase D); only the transport primitives the
// pure-data API needs remain.
func (q *Workspace) createCue(cueData map[string]any, cueNumber string) (string, error) {
	cueType, _ := cueData["type"].(string)
	cueName, _ := cueData["name"].(string)

	// Create new cue with type - workspace ID is required
	if q.workspace_id == "" {
		return "", fmt.Errorf("workspace ID is required for cue creation but not available")
	}

	address := q.addressBuilder.BuildAddress(messages.MsgWorkspaceNew, nil)
	log.Debug("Creating cue with OSC", "address", address, "type", cueType)
	reply := q.Send(address, cueType)

	if len(reply) == 0 {
		return "", fmt.Errorf("no reply received when creating cue")
	}

	// Extract the new cue's unique ID from reply
	replyStr, ok := reply[0].(string)
	if !ok {
		return "", fmt.Errorf("invalid reply format")
	}

	var newCueData map[string]any
	err := json.Unmarshal([]byte(replyStr), &newCueData)
	if err != nil {
		return "", fmt.Errorf("failed to parse new cue reply: %v", err)
	}

	// Check for error status in reply
	if status, ok := newCueData["status"].(string); ok && status == "error" {
		return "", formatErrorWithJSON("QLab rejected cue creation", replyStr)
	}

	uniqueID, ok := newCueData["data"].(string)
	if !ok {
		// Check if data contains "badpass" indicating connection issues
		if data, hasData := newCueData["data"]; hasData {
			if dataStr, isStr := data.(string); isStr && dataStr == "badpass" {
				return "", fmt.Errorf("QLab authentication failed - check passcode and ensure workspace is connected")
			}
		}
		return "", formatErrorWithJSON("no uniqueID in new cue reply", replyStr)
	}

	log.Infof("Created cue with ID: %s", uniqueID)

	// Track this cue for potential rollback
	q.trackCreatedCue(uniqueID)

	// Set cue properties
	if cueName != "" {
		if err := q.setCueProperty(uniqueID, "name", cueName); err != nil {
			return "", fmt.Errorf("failed to set cue name: %v", err)
		}
	}

	if cueNumber != "" {
		if err := q.setCueProperty(uniqueID, "number", cueNumber); err != nil {
			// Check if this is a cue number conflict error
			if _, isConflict := err.(*CueNumberConflictError); isConflict {
				log.Warnf("Skipping cue number assignment due to conflict: %v", err)
			} else {
				return "", fmt.Errorf("failed to set cue number: %v", err)
			}
		}
	}

	// Handle fileTarget for any cue type (audio, video, etc.)
	if fileTarget, ok := cueData["fileTarget"].(string); ok && fileTarget != "" {
		// Resolve relative paths to absolute paths
		absoluteFilePath, err := q.resolveFilePath(fileTarget)
		if err != nil {
			return "", fmt.Errorf("failed to resolve file path '%s': %v", fileTarget, err)
		}

		if err := q.setCueProperty(uniqueID, "file", absoluteFilePath); err != nil {
			return "", fmt.Errorf("failed to set file: %v", err)
		}
	}

	// Set type-specific properties
	switch cueType {
	case "text":
		if text, ok := cueData["text"].(string); ok && text != "" {
			if err := q.setCueProperty(uniqueID, "text", text); err != nil {
				return "", fmt.Errorf("failed to set text: %v", err)
			}
		}
		// Set text format color (text/format/color) - requires 4 separate numeric arguments
		if textColor, ok := cueData["text/format/color"].([]any); ok && len(textColor) == 4 {
			if err := q.setCuePropertyWithArgs(uniqueID, "text/format/color", textColor[0], textColor[1], textColor[2], textColor[3]); err != nil {
				return "", fmt.Errorf("failed to set text color: %v", err)
			}
		}
		// Set text background color (text/format/backgroundColor) - requires 4 separate numeric arguments
		if textBgColor, ok := cueData["text/format/backgroundColor"].([]any); ok && len(textBgColor) == 4 {
			if err := q.setCuePropertyWithArgs(uniqueID, "text/format/backgroundColor", textBgColor[0], textBgColor[1], textBgColor[2], textBgColor[3]); err != nil {
				return "", fmt.Errorf("failed to set text background color: %v", err)
			}
		}
	case "audio":
		if infiniteLoop, ok := cueData["infiniteLoop"].(bool); ok && infiniteLoop {
			if err := q.setCueProperty(uniqueID, "infiniteLoop", "1"); err != nil {
				return "", fmt.Errorf("failed to set infinite loop: %v", err)
			}
		}
	case "group":
		if mode, ok := cueData["mode"].(float64); ok {
			if err := q.setCueProperty(uniqueID, "mode", fmt.Sprintf("%.0f", mode)); err != nil {
				return "", fmt.Errorf("failed to set group mode: %v", err)
			}
		}
	case "fade":
		// Set fade cue target
		if targetNumber, ok := cueData["cueTargetNumber"].(string); ok && targetNumber != "" {
			if err := q.setCueProperty(uniqueID, "cueTargetNumber", targetNumber); err != nil {
				log.Warnf("Failed to set cueTargetNumber %s, trying cueTargetID fallback: %v", targetNumber, err)
				// Fallback to cueTargetID if we have it
				if targetID, ok := cueData["cueTargetID"].(string); ok && targetID != "" {
					if err := q.setCueProperty(uniqueID, "cueTargetID", targetID); err != nil {
						return "", fmt.Errorf("failed to set cue target: %v", err)
					}
				}
			}
		} else if targetID, ok := cueData["cueTargetID"].(string); ok && targetID != "" {
			// Only cueTargetID is available
			if err := q.setCueProperty(uniqueID, "cueTargetID", targetID); err != nil {
				return "", fmt.Errorf("failed to set cue target: %v", err)
			}
		}
		// Set fade geometry parameter enables
		if doOpacity, ok := cueData["doOpacity"].(bool); ok && doOpacity {
			if err := q.setCueProperty(uniqueID, "doOpacity", "1"); err != nil {
				log.Warnf("Failed to set doOpacity for fade cue %s: %v", uniqueID, err)
			}
		}
		if doTranslation, ok := cueData["doTranslation"].(bool); ok && doTranslation {
			if err := q.setCueProperty(uniqueID, "doTranslation", "1"); err != nil {
				log.Warnf("Failed to set doTranslation for fade cue %s: %v", uniqueID, err)
			}
		}
		if doScale, ok := cueData["doScale"].(bool); ok && doScale {
			if err := q.setCueProperty(uniqueID, "doScale", "1"); err != nil {
				log.Warnf("Failed to set doScale for fade cue %s: %v", uniqueID, err)
			}
		}
		if doRotation, ok := cueData["doRotation"].(bool); ok && doRotation {
			if err := q.setCueProperty(uniqueID, "doRotation", "1"); err != nil {
				log.Warnf("Failed to set doRotation for fade cue %s: %v", uniqueID, err)
			}
		}
		// Set geometry properties for fade cues
		if opacity, ok := cueData["opacity"].(float64); ok && opacity > 0 {
			if err := q.setCueProperty(uniqueID, "opacity", fmt.Sprintf("%g", opacity)); err != nil {
				log.Warnf("Failed to set opacity for fade cue %s: %v", uniqueID, err)
			}
		}
		if translation, ok := cueData["translation"].([]any); ok && len(translation) == 2 {
			x, _ := translation[0].(float64)
			y, _ := translation[1].(float64)
			if err := q.setCuePropertyWithArgs(uniqueID, "translation", float32(x), float32(y)); err != nil {
				log.Warnf("Failed to set translation for fade cue %s: %v", uniqueID, err)
			}
		}
		if scale, ok := cueData["scale"].([]any); ok && len(scale) == 2 {
			x, _ := scale[0].(float64)
			y, _ := scale[1].(float64)
			if err := q.setCuePropertyWithArgs(uniqueID, "scale", float32(x), float32(y)); err != nil {
				log.Warnf("Failed to set scale for fade cue %s: %v", uniqueID, err)
			}
		}
		if rotation, ok := cueData["rotation"].(float64); ok && rotation != 0 {
			if err := q.setCueProperty(uniqueID, "rotation", fmt.Sprintf("%g", rotation)); err != nil {
				log.Warnf("Failed to set rotation for fade cue %s: %v", uniqueID, err)
			}
		}
	case "list", "cart":
		// List and Cart cues have read-only mode properties, skip mode setting
	case "start", "stop":
		// First try cueTargetNumber (preferred approach)
		if targetNumber, ok := cueData["cueTargetNumber"].(string); ok && targetNumber != "" {
			if err := q.setCueProperty(uniqueID, "cueTargetNumber", targetNumber); err != nil {
				log.Warnf("Failed to set cueTargetNumber %s, trying cueTargetID fallback: %v", targetNumber, err)
				// Fallback to cueTargetID if we have it
				if targetID, ok := cueData["cueTargetID"].(string); ok && targetID != "" {
					if err := q.setCueProperty(uniqueID, "cueTargetID", targetID); err != nil {
						return "", fmt.Errorf("failed to set cue target: %v", err)
					}
				}
			}
		} else if targetID, ok := cueData["cueTargetID"].(string); ok && targetID != "" {
			// Only cueTargetID is available
			if err := q.setCueProperty(uniqueID, "cueTargetID", targetID); err != nil {
				return "", fmt.Errorf("failed to set cue target: %v", err)
			}
		}
	}

	return uniqueID, nil
}

// updateCueProperties updates the properties of an existing cue identified by uniqueID.
// It is a pure-data OSC primitive called by ApplyCueChanges.
func (q *Workspace) updateCueProperties(uniqueID string, cueData map[string]any) error {
	cueType, _ := cueData["type"].(string)
	cueName, _ := cueData["name"].(string)

	log.Debug("Updating cue properties", "uniqueID", uniqueID, "type", cueType, "name", cueName)

	// Set cue properties that may have changed
	if cueName != "" {
		if err := q.setCueProperty(uniqueID, "name", cueName); err != nil {
			return fmt.Errorf("failed to update cue name: %v", err)
		}
	}

	// Handle fileTarget for any cue type (audio, video, etc.)
	if fileTarget, ok := cueData["fileTarget"].(string); ok && fileTarget != "" {
		// Resolve relative paths to absolute paths
		absoluteFilePath, err := q.resolveFilePath(fileTarget)
		if err != nil {
			return fmt.Errorf("failed to resolve file path '%s': %v", fileTarget, err)
		}

		if err := q.setCueProperty(uniqueID, "file", absoluteFilePath); err != nil {
			return fmt.Errorf("failed to update file: %v", err)
		}
	}

	// Set type-specific properties
	switch cueType {
	case "text":
		if text, ok := cueData["text"].(string); ok && text != "" {
			if err := q.setCueProperty(uniqueID, "text", text); err != nil {
				return fmt.Errorf("failed to update text: %v", err)
			}
		}
		// Set text format color (text/format/color) - requires 4 separate numeric arguments as float32
		if textColor, ok := cueData["text/format/color"].([]any); ok && len(textColor) == 4 {
			// Convert to float32 for OSC
			r, _ := textColor[0].(float64)
			g, _ := textColor[1].(float64)
			b, _ := textColor[2].(float64)
			a, _ := textColor[3].(float64)
			if err := q.setCuePropertyWithArgs(uniqueID, "text/format/color", float32(r), float32(g), float32(b), float32(a)); err != nil {
				return fmt.Errorf("failed to update text color: %v", err)
			}
		}
		// Set text background color (text/format/backgroundColor) - requires 4 separate numeric arguments as float32
		if textBgColor, ok := cueData["text/format/backgroundColor"].([]any); ok && len(textBgColor) == 4 {
			// Convert to float32 for OSC
			r, _ := textBgColor[0].(float64)
			g, _ := textBgColor[1].(float64)
			b, _ := textBgColor[2].(float64)
			a, _ := textBgColor[3].(float64)
			if err := q.setCuePropertyWithArgs(uniqueID, "text/format/backgroundColor", float32(r), float32(g), float32(b), float32(a)); err != nil {
				return fmt.Errorf("failed to update text background color: %v", err)
			}
		}
		// Set text format properties
		if fontSize, ok := cueData["text/format/fontSize"].(float64); ok && fontSize > 0 {
			if err := q.setCueProperty(uniqueID, "text/format/fontSize", fmt.Sprintf("%g", fontSize)); err != nil {
				return fmt.Errorf("failed to update font size: %v", err)
			}
		}
		if alignment, ok := cueData["text/format/alignment"].(string); ok && alignment != "" {
			if err := q.setCueProperty(uniqueID, "text/format/alignment", alignment); err != nil {
				return fmt.Errorf("failed to update text alignment: %v", err)
			}
		}
		// Set geometry properties
		if stageName, ok := cueData["stageName"].(string); ok && stageName != "" {
			if err := q.setCueProperty(uniqueID, "stageName", stageName); err != nil {
				return fmt.Errorf("failed to update stage name: %v", err)
			}
		}
		if stageID, ok := cueData["stageID"].(string); ok && stageID != "" {
			if err := q.setCueProperty(uniqueID, "stageID", stageID); err != nil {
				return fmt.Errorf("failed to update stage ID: %v", err)
			}
		}
		if translation, ok := cueData["translation"].([]any); ok && len(translation) == 2 {
			x, _ := translation[0].(float64)
			y, _ := translation[1].(float64)
			if err := q.setCuePropertyWithArgs(uniqueID, "translation", float32(x), float32(y)); err != nil {
				return fmt.Errorf("failed to update translation: %v", err)
			}
		}
		if opacity, ok := cueData["opacity"].(float64); ok && opacity > 0 {
			if err := q.setCueProperty(uniqueID, "opacity", fmt.Sprintf("%g", opacity)); err != nil {
				return fmt.Errorf("failed to update opacity: %v", err)
			}
		}
	case "audio":
		if infiniteLoop, ok := cueData["infiniteLoop"].(bool); ok && infiniteLoop {
			if err := q.setCueProperty(uniqueID, "infiniteLoop", "1"); err != nil {
				return fmt.Errorf("failed to update infinite loop: %v", err)
			}
		}
	case "group":
		if mode, ok := cueData["mode"].(float64); ok {
			if err := q.setCueProperty(uniqueID, "mode", fmt.Sprintf("%.0f", mode)); err != nil {
				return fmt.Errorf("failed to update group mode: %v", err)
			}
		}
	case "fade":
		// Set fade geometry parameter enables
		if doOpacity, ok := cueData["doOpacity"].(bool); ok && doOpacity {
			if err := q.setCueProperty(uniqueID, "doOpacity", "1"); err != nil {
				log.Warnf("Failed to set doOpacity for fade cue %s: %v", uniqueID, err)
			}
		}
		if doTranslation, ok := cueData["doTranslation"].(bool); ok && doTranslation {
			if err := q.setCueProperty(uniqueID, "doTranslation", "1"); err != nil {
				log.Warnf("Failed to set doTranslation for fade cue %s: %v", uniqueID, err)
			}
		}
		if doScale, ok := cueData["doScale"].(bool); ok && doScale {
			if err := q.setCueProperty(uniqueID, "doScale", "1"); err != nil {
				log.Warnf("Failed to set doScale for fade cue %s: %v", uniqueID, err)
			}
		}
		if doRotation, ok := cueData["doRotation"].(bool); ok && doRotation {
			if err := q.setCueProperty(uniqueID, "doRotation", "1"); err != nil {
				log.Warnf("Failed to set doRotation for fade cue %s: %v", uniqueID, err)
			}
		}
		// Set geometry properties for fade cues
		if opacity, ok := cueData["opacity"].(float64); ok && opacity > 0 {
			if err := q.setCueProperty(uniqueID, "opacity", fmt.Sprintf("%g", opacity)); err != nil {
				return fmt.Errorf("failed to update opacity: %v", err)
			}
		}
		if translation, ok := cueData["translation"].([]any); ok && len(translation) == 2 {
			x, _ := translation[0].(float64)
			y, _ := translation[1].(float64)
			if err := q.setCuePropertyWithArgs(uniqueID, "translation", float32(x), float32(y)); err != nil {
				return fmt.Errorf("failed to update translation: %v", err)
			}
		}
		if scale, ok := cueData["scale"].([]any); ok && len(scale) == 2 {
			x, _ := scale[0].(float64)
			y, _ := scale[1].(float64)
			if err := q.setCuePropertyWithArgs(uniqueID, "scale", float32(x), float32(y)); err != nil {
				return fmt.Errorf("failed to update scale: %v", err)
			}
		}
		if rotation, ok := cueData["rotation"].(float64); ok && rotation != 0 {
			if err := q.setCueProperty(uniqueID, "rotation", fmt.Sprintf("%g", rotation)); err != nil {
				return fmt.Errorf("failed to update rotation: %v", err)
			}
		}
	case "list", "cart":
		// List and Cart cues have read-only mode properties, skip mode setting
	case "start", "stop":
		// Skip cue target setting - this will be handled elsewhere if needed
	}

	// Handle cueTargetNumber if present
	if cueTargetNumber, ok := cueData["cueTargetNumber"].(string); ok && cueTargetNumber != "" {
		if err := q.setCueProperty(uniqueID, "cueTargetNumber", cueTargetNumber); err != nil {
			return fmt.Errorf("failed to update cue target number: %v", err)
		}
	}

	return nil
}

// setCueProperty sets a property on a cue
func (q *Workspace) setCueProperty(uniqueID, property, value string) error {
	if q.workspace_id == "" {
		return fmt.Errorf("workspace ID is required for cue property setting but not available")
	}

	// Check for cue number conflicts
	if property == "number" && value != "" {
		if err := q.handleCueNumberConflict(uniqueID, value); err != nil {
			// If it's a conflict error and we're not forcing, skip setting the property
			if _, isConflict := err.(*CueNumberConflictError); isConflict {
				log.Infof("Skipping cue number assignment due to conflict")
				return err
			}
			return err
		}
	}

	address := q.addressBuilder.BuildCuePropertyAddress(uniqueID, property)
	log.Debug("Setting cue property - sending OSC", "address", address, "value", value)
	reply := q.Send(address, value)

	// Check for error in reply
	if len(reply) > 0 {
		if replyStr, ok := reply[0].(string); ok {
			log.Debug("Received OSC reply for property setting", "reply", replyStr)
			var replyData map[string]any
			if err := json.Unmarshal([]byte(replyStr), &replyData); err == nil {
				if status, ok := replyData["status"].(string); ok && status == "error" {
					log.Debug("ERROR - QLab returned error status for property setting")
					return formatErrorWithJSON(fmt.Sprintf("failed to set %s=%s for cue %s", property, value, uniqueID), replyStr)
				}
			}
		}
	} else {
		log.Debug("WARNING - No reply received for property setting", "property", property, "value", value)
	}

	// Update tracking for cue numbers
	if property == "number" {
		if value != "" {
			q.cueNumbers[value] = uniqueID
			log.Debug("Tracked new cue number", "cue_number", value, "id", uniqueID)
		}
	}

	log.Debug("Set cue property", "property", property, "value", value, "cue_id", uniqueID)
	return nil
}

// setCuePropertyWithArgs sets a property on a cue with multiple OSC arguments
func (q *Workspace) setCuePropertyWithArgs(uniqueID, property string, args ...any) error {
	if q.workspace_id == "" {
		return fmt.Errorf("workspace ID is required for cue property setting but not available")
	}

	address := q.addressBuilder.BuildCuePropertyAddress(uniqueID, property)
	log.Debug("Setting cue property with args - sending OSC", "address", address, "args", args)
	reply := q.SendWithArgs(address, args...)

	// Check for error in reply
	if len(reply) > 0 {
		if replyStr, ok := reply[0].(string); ok {
			log.Debug("Received OSC reply for property setting", "reply", replyStr)
			var replyData map[string]any
			if err := json.Unmarshal([]byte(replyStr), &replyData); err == nil {
				if status, ok := replyData["status"].(string); ok && status == "error" {
					log.Debug("ERROR - QLab returned error status for property setting")
					return formatErrorWithJSON(fmt.Sprintf("failed to set %s for cue %s", property, uniqueID), replyStr)
				}
			}
		}
	} else {
		log.Debug("WARNING - No reply received for property setting", "property", property, "args", args)
	}

	log.Debug("Set cue property with args", "property", property, "args", args, "cue_id", uniqueID)
	return nil
}

// moveCueToParent moves a cue into a parent group cue
func (q *Workspace) moveCueToParent(cueID, parentCueID string) error {
	if q.workspace_id == "" {
		return fmt.Errorf("workspace ID is required for cue movement but not available")
	}

	// Build the move address: /workspace/{id}/move/{cue_id} {new_index} {new_parent_cue_id}
	address := fmt.Sprintf("/workspace/%s/move/%s", q.workspace_id, cueID)

	// Use index 0 to place the cue at the beginning of the parent group
	log.Debug("Moving cue into parent at index 0", "cue_id", cueID, "parent_id", parentCueID)
	reply := q.SendWithArgs(address, int32(0), parentCueID)

	// Check for error in reply
	if len(reply) > 0 {
		if replyStr, ok := reply[0].(string); ok {
			var replyData map[string]any
			if err := json.Unmarshal([]byte(replyStr), &replyData); err == nil {
				if status, ok := replyData["status"].(string); ok && status == "error" {
					return formatErrorWithJSON(fmt.Sprintf("failed to move cue %s into parent %s", cueID, parentCueID), replyStr)
				}
			}
		}
	}

	log.Debug("Moved cue into parent", "cue_id", cueID, "parent_id", parentCueID)
	return nil
}

// getCueChildren queries QLab for the children of a specific cue
func (q *Workspace) getCueChildren(cueID string) ([]map[string]any, error) {
	if q.workspace_id == "" {
		return nil, fmt.Errorf("workspace ID is required for cue queries but not available")
	}

	// Build the children query address: /workspace/{id}/cue_id/{cue_id}/children
	address := fmt.Sprintf("/workspace/%s/cue_id/%s/children", q.workspace_id, cueID)

	log.Debug("Querying children for cue", "cue_id", cueID)
	reply := q.Send(address, "")

	if len(reply) == 0 {
		return nil, fmt.Errorf("no reply received when querying cue children")
	}

	// Parse the reply
	replyStr, ok := reply[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid reply format from children query")
	}

	var replyData map[string]any
	err := json.Unmarshal([]byte(replyStr), &replyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse children query reply: %v", err)
	}

	// Check for error status
	if status, ok := replyData["status"].(string); ok && status == "error" {
		return nil, formatErrorWithJSON("QLab error querying children", replyStr)
	}

	// Extract the children data
	data, ok := replyData["data"].([]any)
	if !ok {
		return nil, fmt.Errorf("no children data in reply")
	}

	// Convert to map slice
	var children []map[string]any
	for _, child := range data {
		if childMap, ok := child.(map[string]any); ok {
			children = append(children, childMap)
		}
	}

	log.Debug("Found children for cue", "child_count", len(children), "cue_id", cueID)
	return children, nil
}

// getAllCueIDs queries QLab for all cue IDs in the workspace
func (q *Workspace) getAllCueIDs() ([]string, error) {
	if q.workspace_id == "" {
		return nil, fmt.Errorf("workspace ID is required for cue queries but not available")
	}

	// Build the cueLists query address: /workspace/{id}/cueLists/uniqueIDs
	address := fmt.Sprintf("/workspace/%s/cueLists/uniqueIDs", q.workspace_id)

	log.Debug("Querying all cue IDs in workspace", "workspace_id", q.workspace_id)
	reply := q.Send(address, "")

	if len(reply) == 0 {
		return nil, fmt.Errorf("no reply received when querying all cue IDs")
	}

	// Parse the reply
	replyStr, ok := reply[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid reply format from all cue IDs query")
	}

	var replyData map[string]any
	err := json.Unmarshal([]byte(replyStr), &replyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse all cue IDs query reply: %v", err)
	}

	// Check for error status
	if status, ok := replyData["status"].(string); ok && status == "error" {
		return nil, formatErrorWithJSON("QLab error querying all cue IDs", replyStr)
	}

	// Extract the data
	data, ok := replyData["data"].([]any)
	if !ok {
		return []string{}, nil // Empty workspace is OK
	}

	// Process each cue list
	var allIDs []string
	for _, cueListData := range data {
		cueList, ok := cueListData.(map[string]any)
		if !ok {
			continue
		}

		// Extract cues from this cue list
		if cues, ok := cueList["cues"].([]any); ok {
			ids := extractCueIDs(cues)
			allIDs = append(allIDs, ids...)
		}
	}

	log.Infof("Found %d total cues in workspace", len(allIDs))
	return allIDs, nil
}

// extractCueIDs recursively extracts all cue IDs from a cues array
func extractCueIDs(cues []any) []string {
	var ids []string
	for _, cueData := range cues {
		cue, ok := cueData.(map[string]any)
		if !ok {
			continue
		}

		// Add this cue's unique ID
		if uniqueID, ok := cue["uniqueID"].(string); ok {
			ids = append(ids, uniqueID)
		}

		// Recursively process children
		if children, ok := cue["cues"].([]any); ok {
			childIDs := extractCueIDs(children)
			ids = append(ids, childIDs...)
		}
	}
	return ids
}

// getWorkspaceBasePath queries QLab for the workspace base path with fallback to workingDirectory
func (q *Workspace) getWorkspaceBasePath() (string, error) {
	if q.workspace_id == "" {
		return "", fmt.Errorf("workspace ID is required for basePath query but not available")
	}

	// Try workspace-specific basePath first
	basePath, err := q.queryWorkspaceBasePath()
	if err != nil {
		log.Debug("Failed to get workspace basePath, trying workingDirectory fallback", "error", err)
	} else if basePath != "" {
		return basePath, nil
	}

	// Fallback to /workingDirectory if basePath is empty or failed
	log.Debugf("BasePath empty or unavailable, falling back to /workingDirectory")
	workingDir, err := q.queryWorkingDirectory()
	if err != nil {
		return "", fmt.Errorf("failed to get workingDirectory fallback: %v", err)
	}

	return workingDir, nil
}

// queryWorkspaceBasePath queries /workspace/{id}/basePath
func (q *Workspace) queryWorkspaceBasePath() (string, error) {
	// Build the basePath query address: /workspace/{id}/basePath
	address := fmt.Sprintf("/workspace/%s/basePath", q.workspace_id)

	log.Debug("Querying basePath for workspace", "workspace_id", q.workspace_id)
	reply := q.Send(address, "")

	if len(reply) == 0 {
		return "", fmt.Errorf("no reply received when querying workspace basePath")
	}

	// Parse the reply
	replyStr, ok := reply[0].(string)
	if !ok {
		return "", fmt.Errorf("invalid reply format from basePath query")
	}

	var replyData map[string]any
	if err := json.Unmarshal([]byte(replyStr), &replyData); err != nil {
		return "", fmt.Errorf("failed to parse basePath reply: %v", err)
	}

	// Check for error in response
	if status, ok := replyData["status"].(string); ok && status != "ok" {
		return "", fmt.Errorf("QLab error getting basePath: %s", replyData["error"])
	}

	// Extract the basePath from the data field
	if data, ok := replyData["data"].(string); ok {
		log.Debug("Workspace basePath retrieved", "base_path", data)
		return data, nil
	}

	return "", fmt.Errorf("basePath not found in response data")
}

// queryWorkingDirectory queries /workingDirectory as fallback
func (q *Workspace) queryWorkingDirectory() (string, error) {
	address := "/workingDirectory"

	log.Debug("Querying /workingDirectory as fallback")
	reply := q.Send(address, "")

	if len(reply) == 0 {
		return "", fmt.Errorf("no reply received when querying /workingDirectory")
	}

	// Parse the reply
	replyStr, ok := reply[0].(string)
	if !ok {
		return "", fmt.Errorf("invalid reply format from /workingDirectory query")
	}

	var replyData map[string]any
	if err := json.Unmarshal([]byte(replyStr), &replyData); err != nil {
		return "", fmt.Errorf("failed to parse /workingDirectory reply: %v", err)
	}

	// Check for error in response
	if status, ok := replyData["status"].(string); ok && status != "ok" {
		return "", fmt.Errorf("QLab error getting /workingDirectory: %s", replyData["error"])
	}

	// Extract the working directory from the data field
	if data, ok := replyData["data"].(string); ok {
		log.Debug("Working directory retrieved", "working_directory", data)
		return data, nil
	}

	return "", fmt.Errorf("workingDirectory not found in response data")
}

// resolveFilePath converts relative file paths to absolute paths using workspace basePath
func (q *Workspace) resolveFilePath(filePath string) (string, error) {
	// Check if path is already absolute
	if filepath.IsAbs(filePath) {
		return filePath, nil
	}

	// First try to resolve relative to CUE file directory (if available)
	if q.cueFileDirectory != "" {
		absolutePath := filepath.Join(q.cueFileDirectory, filePath)
		log.Debug("Resolved relative path to absolute path (via CUE file directory)", "relative_path", filePath, "absolute_path", absolutePath)
		return absolutePath, nil
	}

	// Fallback to workspace base path
	basePath, err := q.getWorkspaceBasePath()
	if err != nil {
		return "", fmt.Errorf("failed to get workspace basePath: %v", err)
	}

	// Join the base path with the relative file path
	absolutePath := filepath.Join(basePath, filePath)
	log.Debug("Resolved relative path to absolute path (via workspace basePath)", "relative_path", filePath, "absolute_path", absolutePath)

	return absolutePath, nil
}

// deleteCue deletes a specific cue by ID
func (q *Workspace) deleteCue(cueID string) error {
	if q.workspace_id == "" {
		return fmt.Errorf("workspace ID is required for cue deletion but not available")
	}

	// Build the delete address: /workspace/{id}/delete_id/{cue_id}
	address := fmt.Sprintf("/workspace/%s/delete_id/%s", q.workspace_id, cueID)

	log.Debug("Deleting cue", "cue_id", cueID)
	reply := q.Send(address, "")

	if len(reply) == 0 {
		return fmt.Errorf("no reply received when deleting cue")
	}

	// Parse the reply
	replyStr, ok := reply[0].(string)
	if !ok {
		return fmt.Errorf("invalid reply format from delete cue")
	}

	var replyData map[string]any
	err := json.Unmarshal([]byte(replyStr), &replyData)
	if err != nil {
		return fmt.Errorf("failed to parse delete cue reply: %v", err)
	}

	// Check for error status
	if status, ok := replyData["status"].(string); ok && status == "error" {
		return formatErrorWithJSON("QLab error deleting cue", replyStr)
	}

	log.Debug("Successfully deleted cue", "cue_id", cueID)
	return nil
}

// getCueLists queries QLab for all cue lists, using cached data if available
func (q *Workspace) getCueLists() ([]any, error) {
	// Return cached data if available
	if q.cueListsCache != nil {
		log.Debug("Using cached cue lists data")
		return q.cueListsCache, nil
	}

	if q.workspace_id == "" {
		return nil, fmt.Errorf("workspace ID is required but not available")
	}

	log.Debug("Querying cue lists from QLab")
	address := fmt.Sprintf("/workspace/%s/cueLists", q.workspace_id)
	reply := q.Send(address, "")

	if len(reply) == 0 {
		log.Debug("No reply received when querying cue lists")
		return nil, nil
	}

	replyStr, ok := reply[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid reply format from cue lists query")
	}

	var replyData map[string]any
	err := json.Unmarshal([]byte(replyStr), &replyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cue lists reply: %v", err)
	}

	// Check for error status
	if status, ok := replyData["status"].(string); ok && status == "error" {
		return nil, fmt.Errorf("QLab error querying cue lists: %v", replyData["error"])
	}

	// Extract the cue lists data
	data, ok := replyData["data"].([]any)
	if !ok {
		log.Debug("No cue lists found in response")
		return nil, nil
	}

	// Cache the result for subsequent calls
	q.cueListsCache = data
	return data, nil
}

// indexExistingCues queries all existing cues and populates the cueNumbers map for conflict detection
func (q *Workspace) indexExistingCues() error {
	if q.workspace_id == "" {
		return fmt.Errorf("workspace ID is required for cue indexing but not available")
	}

	log.Debug("Indexing existing cues for conflict detection")

	// Use cached cue lists data
	data, err := q.getCueLists()
	if err != nil {
		return err
	}

	if data == nil {
		log.Debug("No cue lists found during indexing")
		return nil
	}

	for _, cueListData := range data {
		cueList, ok := cueListData.(map[string]any)
		if !ok {
			continue
		}

		if cues, ok := cueList["cues"].([]any); ok {
			count := q.indexCueNumbers(cues)
			log.Debugf("Indexed %d cues from cue list", count)
		}
	}

	return nil
}

// CueNumberConflictError represents a cue number conflict during cue creation/update
type CueNumberConflictError struct {
	CueNumber  string
	ExistingID string
	NewCueID   string
}

func (e *CueNumberConflictError) Error() string {
	return fmt.Sprintf("cue number conflict: '%s' is already assigned to cue %s", e.CueNumber, e.ExistingID)
}

// handleCueNumberConflict checks for conflicts and handles resolution based on force flag
func (q *Workspace) handleCueNumberConflict(newCueID, cueNumber string) error {
	// Check if this number is already in use
	existingID, exists := q.cueNumbers[cueNumber]
	if !exists {
		return nil // No conflict
	}

	// If the existing cue is the same as the new one, no conflict
	if existingID == newCueID {
		return nil
	}

	log.Warnf("Cue number conflict detected: '%s' is already assigned to cue %s", cueNumber, existingID)

	if q.forceCueNumbers {
		// Force cue number by clearing the existing cue's number
		log.Infof("Force mode enabled: clearing number from existing cue %s", existingID)

		err := q.clearCueNumber(existingID)
		if err != nil {
			return fmt.Errorf("failed to clear conflicting cue number: %v", err)
		}

		// Remove from tracking
		delete(q.cueNumbers, cueNumber)
		log.Infof("Cleared cue number '%s' from existing cue %s", cueNumber, existingID)
		return nil
	} else {
		// Return special error type for conflicts when not forcing
		return &CueNumberConflictError{
			CueNumber:  cueNumber,
			ExistingID: existingID,
			NewCueID:   newCueID,
		}
	}
}

// clearCueNumber removes the number from a cue
func (q *Workspace) clearCueNumber(cueID string) error {
	if q.workspace_id == "" {
		return fmt.Errorf("workspace ID is required for clearing cue number but not available")
	}

	address := q.addressBuilder.BuildCuePropertyAddress(cueID, "number")
	reply := q.Send(address, "") // Empty string clears the number

	// Check for error in reply
	if len(reply) > 0 {
		if replyStr, ok := reply[0].(string); ok {
			var replyData map[string]any
			if err := json.Unmarshal([]byte(replyStr), &replyData); err == nil {
				if status, ok := replyData["status"].(string); ok && status == "error" {
					return formatErrorWithJSON(fmt.Sprintf("failed to clear number for cue %s", cueID), replyStr)
				}
			}
		}
	}

	log.Debug("Cleared number for cue", "cue_id", cueID)
	return nil
}

// indexCueNumbers recursively processes cues and indexes their numbers
func (q *Workspace) indexCueNumbers(cues []any) int {
	count := 0
	for _, cueData := range cues {
		cue, ok := cueData.(map[string]any)
		if !ok {
			continue
		}

		// Index this cue's number if it has one
		if uniqueID, hasID := cue["uniqueID"].(string); hasID {
			var cueNumber string
			if num, hasNumber := cue["number"]; hasNumber && num != nil {
				// Handle different number types while preserving decimal format
				switch v := num.(type) {
				case string:
					// Already a string, use as-is
					cueNumber = v
				case float64:
					// For float64, use %g to get natural representation,
					// but preserve at least one decimal place for whole numbers if they came from "X.0"
					if v == float64(int64(v)) && v >= 0 && v <= 999 {
						// It's a whole number that might have been "X.0" originally
						// Use %.1f to force one decimal place for common cue numbers
						cueNumber = fmt.Sprintf("%.1f", v)
					} else {
						// Use %g for non-whole numbers (preserves natural format)
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
			if cueNumber != "" {
				q.cueNumbers[cueNumber] = uniqueID
				count++
				log.Debug("Indexed cue number", "cue_number", cueNumber, "id", uniqueID)
			}
		}

		// Recursively process children if this is a group cue
		if children, ok := cue["cues"].([]any); ok {
			childCount := q.indexCueNumbers(children)
			count += childCount
		}
	}
	return count
}

// clearAllCues removes all cues from the workspace
func (q *Workspace) clearAllCues() error {
	cueIDs, err := q.getAllCueIDs()
	if err != nil {
		// Check if this is the specific API error we expect to handle gracefully
		if strings.Contains(err.Error(), "QLab error querying all cue IDs") {
			log.Warnf("cueLists/uniqueIDs endpoint not available, cleanup will be limited: %v", err)
			return nil // Don't fail the test for this known API limitation
		}
		return fmt.Errorf("failed to get cue IDs for cleanup: %v", err)
	}

	if len(cueIDs) == 0 {
		log.Info("No cues to clean up")
		return nil
	}

	log.Infof("Cleaning up %d cues from workspace", len(cueIDs))

	// Delete each cue - track if any deletions failed
	var deletionErrors []string
	for _, cueID := range cueIDs {
		err := q.deleteCue(cueID)
		if err != nil {
			deletionErrors = append(deletionErrors, fmt.Sprintf("cue %s: %v", cueID, err))
			log.Warnf("Failed to delete cue %s: %v", cueID, err)
		}
	}

	// If we had deletion errors, that indicates a real QLab communication problem
	if len(deletionErrors) > 0 {
		return fmt.Errorf("failed to delete %d cues: %s", len(deletionErrors), strings.Join(deletionErrors, "; "))
	}

	log.Info("Workspace cleanup completed")
	return nil
}

// ensureCuejitsuInbox detects or creates a "Cuejitsu Inbox" cue list for staging imported cues
func (q *Workspace) ensureCuejitsuInbox() (string, error) {
	if q.workspace_id == "" {
		return "", fmt.Errorf("workspace ID is required for inbox management but not available")
	}

	log.Debug("Ensuring Cuejitsu Inbox cue list exists")

	// First, try to find existing "Cuejitsu Inbox" cue list
	inboxID, err := q.findCuejitsuInbox()
	if err != nil {
		return "", fmt.Errorf("error searching for Cuejitsu Inbox: %v", err)
	}

	// If found, store and return its ID
	if inboxID != "" {
		log.Infof("Found existing Cuejitsu Inbox cue list: %s", inboxID)
		q.inboxID = inboxID
		return inboxID, nil
	}

	// If not found, create it
	log.Info("Cuejitsu Inbox not found, creating new cue list")
	inboxID, err = q.createCuejitsuInbox()
	if err != nil {
		return "", fmt.Errorf("error creating Cuejitsu Inbox: %v", err)
	}

	log.Infof("Created Cuejitsu Inbox cue list: %s", inboxID)
	q.inboxID = inboxID
	return inboxID, nil
}

// findCuejitsuInbox searches for an existing "Cuejitsu Inbox" cue list
func (q *Workspace) findCuejitsuInbox() (string, error) {
	// Use cached cue lists data
	data, err := q.getCueLists()
	if err != nil {
		return "", err
	}

	if data == nil {
		return "", nil // No cue lists exist
	}

	// Search for "Cuejitsu Inbox" in each cue list
	for _, cueListData := range data {
		cueList, ok := cueListData.(map[string]any)
		if !ok {
			continue
		}

		// Check if this cue list is named "Cuejitsu Inbox"
		if name, ok := cueList["name"].(string); ok && name == "Cuejitsu Inbox" {
			if uniqueID, ok := cueList["uniqueID"].(string); ok {
				return uniqueID, nil
			}
		}
	}

	// No "Cuejitsu Inbox" found
	return "", nil
}
