package qlab

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/zenibako/qlab-golang/messages"
)

// compareCacheWithCurrentState compares cached workspace with current QLab state
func (q *Workspace) compareCacheWithCurrentState(cachedWorkspace, currentWorkspace map[string]any) bool {
	// For a basic comparison, we'll check if the structure and main properties match
	// A more sophisticated comparison could check individual cue properties

	cachedCues := q.indexCuesFromWorkspace(cachedWorkspace)
	currentCues := q.indexCuesFromWorkspace(currentWorkspace)

	// Check if the number of cues matches
	if len(cachedCues) != len(currentCues) {
		return false
	}

	// Check if each cached cue matches the current one
	for cueNumber, cachedCue := range cachedCues {
		currentCue, exists := currentCues[cueNumber]
		if !exists {
			return false
		}

		if !q.compareCueProperties(cachedCue, currentCue) {
			return false
		}
	}

	return true
}

// compareCueProperties compares the important properties of two cues
func (q *Workspace) compareCueProperties(cue1, cue2 map[string]any) bool {
	differences := q.compareCuePropertiesDetailed(cue1, cue2)
	return len(differences) == 0
}

// compareCuePropertiesDetailed compares properties and returns detailed differences
func (q *Workspace) compareCuePropertiesDetailed(cue1, cue2 map[string]any) map[string]string {
	// List of all properties we might want to compare
	allProperties := []string{
		"name", "type", "fileTarget", "duration", "cueTargetNumber",
		"armed", "colorName", "flagged", "notes",
	}

	differences := make(map[string]string)

	for _, prop := range allProperties {
		// Only compare properties that exist in both cues or where one has a meaningful value
		val1 := q.normalizeProperty(cue1[prop])
		val2 := q.normalizeProperty(cue2[prop])

		// Skip comparison if both values are empty/missing
		if val1 == "" && val2 == "" {
			continue
		}

		// For properties that may not exist in QLab data (like fileTarget, cueTargetNumber),
		// only compare if BOTH cues have the property defined
		if prop == "fileTarget" || prop == "cueTargetNumber" {
			// Check if both cues actually have this property key
			_, has1 := cue1[prop]
			_, has2 := cue2[prop]

			// Only compare if BOTH cues have this property
			// If one cue lacks the property entirely, skip comparison to avoid false positives
			if !has1 || !has2 {
				continue
			}
		}

		// Apply smart comparison for properties that might have default value differences
		if !q.comparePropertyValues(prop, val1, val2) {
			differences[prop] = fmt.Sprintf("'%s' -> '%s'", val1, val2)
		}
	}

	return differences
}

// comparePropertyValues applies smart comparison logic for specific properties
func (q *Workspace) comparePropertyValues(property, val1, val2 string) bool {
	if val1 == val2 {
		return true
	}

	// Handle boolean properties: treat "false", "" and "true" as equivalent for armed/flagged
	// These are operational states, not content that should trigger updates
	if property == "armed" || property == "flagged" {
		// All boolean states should be considered equivalent for cue matching
		// Armed/flagged states are user-controlled and shouldn't prevent cue recognition
		return true
	}

	// Handle numeric properties: treat "0" and "" as equivalent (both are zero values)
	if property == "duration" {
		if (val1 == "0" && val2 == "") || (val1 == "" && val2 == "0") {
			return true
		}
	}

	// Handle type property: QLab capitalizes cue types
	if property == "type" {
		// Normalize both values to lowercase for comparison
		if strings.EqualFold(val1, val2) {
			return true
		}
	}

	// Handle fileTarget property: compare basename only since paths may differ
	if property == "fileTarget" {
		// If both have values, compare the basename (filename)
		if val1 != "" && val2 != "" {
			base1 := filepath.Base(val1)
			base2 := filepath.Base(val2)
			return base1 == base2
		}
		// If one is empty and the other isn't, they're different
		return false
	}

	// Handle colorName: treat "" and "none" as equivalent (both mean no color)
	if property == "colorName" {
		if (val1 == "" && val2 == "none") || (val1 == "none" && val2 == "") {
			return true
		}
	}

	// Handle cueTargetNumber: treat "" and actual values as different unless both empty
	if property == "cueTargetNumber" {
		// Only consider equal if both are empty or both have the same value
		if val1 == "" && val2 == "" {
			return true
		}
		// If one is empty and other isn't, they're different
		return val1 == val2
	}

	return false
}

// normalizeProperty normalizes a property value for comparison
func (q *Workspace) normalizeProperty(value any) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%g", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case int:
		return fmt.Sprintf("%d", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// processCueList recursively processes cues and their sub-cues
func (q *Workspace) processCueList(cueData map[string]any, parentNumber string) error {
	_, err := q.processCueListWithParent(cueData, parentNumber, "")
	return err
}

// setCueTargets sets cue targets using the number-to-ID mapping
func (q *Workspace) setCueTargets(mapping *CueMapping) error {
	for _, cueTarget := range mapping.CuesWithTargets {
		// First try to use cueTargetNumber (preferred approach)
		if err := q.setCueProperty(cueTarget.UniqueID, "cueTargetNumber", cueTarget.TargetNumber); err != nil {
			log.Warnf("Failed to set cueTargetNumber %s for cue %s, trying cueTargetID fallback: %v",
				cueTarget.TargetNumber, cueTarget.UniqueID, err)

			// Fallback to cueTargetID if number approach failed
			if targetID, exists := mapping.NumberToID[cueTarget.TargetNumber]; exists {
				if err := q.setCueProperty(cueTarget.UniqueID, "cueTargetID", targetID); err != nil {
					return fmt.Errorf("failed to set cue target %s -> %s: %v", cueTarget.TargetNumber, targetID, err)
				}
				log.Infof("Set cue target via ID fallback: %s -> %s (%s)", cueTarget.UniqueID, cueTarget.TargetNumber, targetID)
			} else {
				log.Warnf("Target cue number %s not found for cue %s", cueTarget.TargetNumber, cueTarget.UniqueID)
			}
		} else {
			log.Infof("Set cue target via number: %s -> %s", cueTarget.UniqueID, cueTarget.TargetNumber)
		}
	}
	return nil
}

// processCueListWithParent recursively processes cues and their sub-cues with parent tracking
func (q *Workspace) processCueListWithParent(cueData map[string]any, parentNumber string, parentUniqueID string) (string, error) {
	cueType, _ := cueData["type"].(string)
	cueName, _ := cueData["name"].(string)
	var cueNumber string
	if num, ok := cueData["number"]; ok && num != nil {
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

	// Build full cue number with parent prefix
	fullNumber := cueNumber
	if parentNumber != "" && cueNumber != "" {
		// Check if the cue number appears to be absolute (contains decimal point)
		// or relative (simple integer/short string)
		// Absolute numbers like "1.0", "2.5", etc. should not be concatenated
		// Relative numbers like "1", "2", "a", etc. should be concatenated
		if strings.Contains(cueNumber, ".") {
			// Likely an absolute cue number, use as-is
			fullNumber = cueNumber
		} else {
			// Likely a relative cue number, concatenate with parent
			fullNumber = parentNumber + "." + cueNumber
		}
	}

	if cueName != "" {
		if fullNumber != "" {
			log.Infof("Processing cue: [%s] %s (%s)", fullNumber, cueName, cueType)
		} else {
			log.Infof("Processing cue: %s (%s)", cueName, cueType)
		}
	}

	// Create the cue in QLab
	uniqueID, err := q.createCue(cueData, fullNumber)
	if err != nil {
		return "", fmt.Errorf("failed to create cue %s: %v", fullNumber, err)
	}

	// Move cue into parent group if we have a parent
	if parentUniqueID != "" && uniqueID != "" {
		err = q.moveCueToParent(uniqueID, parentUniqueID)
		if err != nil {
			return "", fmt.Errorf("failed to move cue %s into parent %s: %v", uniqueID, parentUniqueID, err)
		}
	}

	// Process sub-cues if they exist
	if subCues, ok := cueData["cues"].([]any); ok {
		for childIndex, subCueData := range subCues {
			if subCue, ok := subCueData.(map[string]any); ok {
				childUniqueID, err := q.processCueListWithParent(subCue, fullNumber, "")
				if err != nil {
					return "", fmt.Errorf("error processing sub-cue %d: %v", childIndex, err)
				}

				// Move the child cue into this parent group at the correct index
				if childUniqueID != "" && uniqueID != "" {
					err = q.moveCueToParentWithIndex(childUniqueID, uniqueID, childIndex)
					if err != nil {
						return "", fmt.Errorf("failed to move child cue %s into parent %s at index %d: %v", childUniqueID, uniqueID, childIndex, err)
					}
				}
			}
		}
	}

	return uniqueID, nil
}

// createCue sends OSC messages to create a cue in QLab and returns the uniqueID
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

// createCueWithoutTarget creates a cue without setting any cue targets (used in two-pass approach)
func (q *Workspace) createCueWithoutTarget(cueData map[string]any, cueNumber string) (string, error) {
	cueType, _ := cueData["type"].(string)
	cueName, _ := cueData["name"].(string)

	// Create new cue with type - workspace ID is required
	if q.workspace_id == "" {
		return "", fmt.Errorf("workspace ID is required for cue creation but not available")
	}

	address := q.addressBuilder.BuildAddress(messages.MsgWorkspaceNew, nil)
	log.Debug("Creating cue - sending OSC", "address", address, "type", cueType)
	reply := q.Send(address, cueType)

	if len(reply) == 0 {
		log.Debug("ERROR - No reply received when creating cue", "type", cueType)
		return "", fmt.Errorf("no reply received when creating cue")
	}

	// Extract the new cue's unique ID from reply
	replyStr, ok := reply[0].(string)
	if !ok {
		log.Debug("ERROR - Invalid reply format for cue creation", "reply", reply)
		return "", fmt.Errorf("invalid reply format")
	}
	log.Debug("Received OSC reply for cue creation", "reply", replyStr)

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

	// Handle common cue properties
	if notes, ok := cueData["notes"].(string); ok && notes != "" {
		if err := q.setCueProperty(uniqueID, "notes", notes); err != nil {
			return "", fmt.Errorf("failed to set notes: %v", err)
		}
	}

	if duration, ok := cueData["duration"].(string); ok && duration != "" && duration != "0" {
		if err := q.setCueProperty(uniqueID, "duration", duration); err != nil {
			return "", fmt.Errorf("failed to set duration: %v", err)
		}
	}

	if preWait, ok := cueData["preWait"].(string); ok && preWait != "" && preWait != "0" {
		if err := q.setCueProperty(uniqueID, "preWait", preWait); err != nil {
			return "", fmt.Errorf("failed to set preWait: %v", err)
		}
	}

	if armed, ok := cueData["armed"].(string); ok && armed == "true" {
		if err := q.setCueProperty(uniqueID, "armed", "1"); err != nil {
			return "", fmt.Errorf("failed to set armed: %v", err)
		}
	}

	if colorName, ok := cueData["colorName"].(string); ok && colorName != "" && colorName != "none" {
		if err := q.setCueProperty(uniqueID, "colorName", colorName); err != nil {
			return "", fmt.Errorf("failed to set colorName: %v", err)
		}
	}

	// Set type-specific properties (excluding cue targets)
	switch cueType {
	case "text":
		// Set basic text property first
		if text, ok := cueData["text"].(string); ok && text != "" {
			if err := q.setCueProperty(uniqueID, "text", text); err != nil {
				return "", fmt.Errorf("failed to set text: %v", err)
			}
		}
		// Set stage assignment BEFORE format properties (required for format props to work)
		if stageName, ok := cueData["stageName"].(string); ok && stageName != "" {
			if err := q.setCueProperty(uniqueID, "stageName", stageName); err != nil {
				log.Warnf("Failed to set stage name (may not exist): %v", err)
			}
		} else if stageID, ok := cueData["stageID"].(string); ok && stageID != "" {
			if err := q.setCueProperty(uniqueID, "stageID", stageID); err != nil {
				log.Warnf("Failed to set stage ID (may not exist): %v", err)
			}
		} else {
			// No stage specified - try to get first available stage
			stages, err := q.getVideoStages()
			if err == nil && len(stages) > 0 {
				firstStageID := stages[0]["uniqueID"].(string)
				log.Debugf("Auto-assigning text cue to first video stage: %s", firstStageID)
				if err := q.setCueProperty(uniqueID, "stageID", firstStageID); err != nil {
					log.Warnf("Failed to auto-assign to video stage: %v", err)
				}
			} else {
				log.Warnf("No video stage available for text cue - format properties may not work")
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
				// Log warning but don't fail - text cue may not be patched to stage yet
				log.Warnf("Failed to set text color for cue %s (may need stage assignment): %v", uniqueID, err)
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
				// Log warning but don't fail - text cue may not be patched to stage yet
				log.Warnf("Failed to set text background color for cue %s (may need stage assignment): %v", uniqueID, err)
			}
		}
		// Set text format properties
		if fontSize, ok := cueData["text/format/fontSize"].(float64); ok && fontSize > 0 {
			if err := q.setCueProperty(uniqueID, "text/format/fontSize", fmt.Sprintf("%g", fontSize)); err != nil {
				log.Warnf("Failed to set font size for cue %s: %v", uniqueID, err)
			}
		}
		if alignment, ok := cueData["text/format/alignment"].(string); ok && alignment != "" {
			if err := q.setCueProperty(uniqueID, "text/format/alignment", alignment); err != nil {
				log.Warnf("Failed to set text alignment for cue %s: %v", uniqueID, err)
			}
		}
		// Set geometry properties
		if stageName, ok := cueData["stageName"].(string); ok && stageName != "" {
			if err := q.setCueProperty(uniqueID, "stageName", stageName); err != nil {
				return "", fmt.Errorf("failed to set stage name: %v", err)
			}
		}
		if stageID, ok := cueData["stageID"].(string); ok && stageID != "" {
			if err := q.setCueProperty(uniqueID, "stageID", stageID); err != nil {
				return "", fmt.Errorf("failed to set stage ID: %v", err)
			}
		}
		if translation, ok := cueData["translation"].([]any); ok && len(translation) == 2 {
			x, _ := translation[0].(float64)
			y, _ := translation[1].(float64)
			if err := q.setCuePropertyWithArgs(uniqueID, "translation", float32(x), float32(y)); err != nil {
				log.Warnf("Failed to set translation for cue %s: %v", uniqueID, err)
			}
		}
		if opacity, ok := cueData["opacity"].(float64); ok && opacity > 0 {
			if err := q.setCueProperty(uniqueID, "opacity", fmt.Sprintf("%g", opacity)); err != nil {
				log.Warnf("Failed to set opacity for cue %s: %v", uniqueID, err)
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
	case "list", "cart":
		// List and Cart cues have read-only mode properties, skip mode setting
	case "start", "stop":
		// Skip cue target setting - this will be handled in the second pass
	}

	return uniqueID, nil
}

// updateCueProperties updates an existing cue with changed properties from cueData
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

	log.Infof("Successfully moved cue %s into parent %s", cueID, parentCueID)
	return nil
}

// moveCueToParentWithIndex moves a cue into a parent group cue at a specific index
func (q *Workspace) moveCueToParentWithIndex(cueID, parentCueID string, index int) error {
	if q.workspace_id == "" {
		return fmt.Errorf("workspace ID is required for cue movement but not available")
	}

	// Build the move address: /workspace/{id}/move/{cue_id} {new_index} {new_parent_cue_id}
	address := fmt.Sprintf("/workspace/%s/move/%s", q.workspace_id, cueID)

	log.Debug("Moving cue into parent at index", "cue_id", cueID, "parent_id", parentCueID, "index", index)
	reply := q.SendWithArgs(address, int32(index), parentCueID)

	// Check for error in reply
	if len(reply) > 0 {
		if replyStr, ok := reply[0].(string); ok {
			var replyData map[string]any
			if err := json.Unmarshal([]byte(replyStr), &replyData); err == nil {
				if status, ok := replyData["status"].(string); ok && status == "error" {
					return formatErrorWithJSON(fmt.Sprintf("failed to move cue %s into parent %s at index %d", cueID, parentCueID, index), replyStr)
				}
			}
		}
	}

	log.Infof("Successfully moved cue %s into parent %s at index %d", cueID, parentCueID, index)
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

		// Recursively process children if this is a group cue
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
		log.Debug("No cue lists found, workspace is empty")
		return nil
	}

	// Process each cue list
	totalCues := 0
	totalCueLists := 0
	for _, cueListData := range data {
		cueList, ok := cueListData.(map[string]any)
		if !ok {
			continue
		}

		// Index this cue list by name
		if name, hasName := cueList["name"].(string); hasName && name != "" {
			if uniqueID, hasID := cueList["uniqueID"].(string); hasID {
				q.cueListNames[name] = uniqueID
				totalCueLists++
			}
		}

		// Extract cues from this cue list
		if cues, ok := cueList["cues"].([]any); ok {
			count := q.indexCueNumbers(cues)
			totalCues += count
		}
	}

	log.Infof("Indexed %d existing cues with numbers and %d cue lists", totalCues, totalCueLists)
	return nil
}

// CueNumberConflictError represents a cue number conflict
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

// IdentifyConflicts analyzes the three-way comparison to find conflicts that need user resolution
// Enhanced version with scope-based and field-level conflict detection
func (q *Workspace) IdentifyConflicts(comparison *ThreeWayComparison) ([]CueConflict, error) {
	var conflicts []CueConflict

	// Handle case where QLab query failed
	if !comparison.HasQLabData {
		if comparison.HasCache {
			log.Warn("QLab data unavailable - using cache-only comparison")
			log.Info("Conflicts cannot be detected without current QLab state")
			log.Info("Recommendation: Increase timeout or check QLab connection")
		}
		return conflicts, nil
	}

	// Only identify conflicts if we have cache (need common ancestor)
	if !comparison.HasCache {
		log.Debug("No cache available - three-way conflict detection unavailable")
		return conflicts, nil
	}

	// If cache matches QLab, then only simple source vs cache conflicts are possible
	// These are typically handled automatically, so we don't need user input
	if comparison.CacheMatchesQLab {
		log.Debug("Cache matches QLab state, no complex conflicts detected")
		return conflicts, nil
	}

	// Use scope-based conflict identification if available
	if comparison.WorkspaceScope != nil {
		return q.identifyConflictsFromScope(comparison.WorkspaceScope), nil
	}

	// Fallback to legacy cue-level conflict detection
	for cueNumber, result := range comparison.CueResults {
		if result == nil {
			continue
		}

		// Look for cases where manual intervention might be needed
		// This occurs when QLab was modified externally or when both source and QLab differ
		if result.Action == "update" && (strings.Contains(result.Reason, "QLab modified externally") || strings.Contains(result.Reason, "both source and QLab modified")) {
			var conflictType ConflictType
			var description string

			if strings.Contains(result.Reason, "QLab modified externally") {
				conflictType = ConflictCacheStale
				description = fmt.Sprintf("Cue %s has been modified in QLab since last sync", cueNumber)
			} else {
				conflictType = ConflictThreeWayDivergence
				description = fmt.Sprintf("Cue %s has been modified in both the source file and QLab since last sync", cueNumber)
			}

			conflict := CueConflict{
				CueNumber:      cueNumber,
				CueIdentifier:  cueNumber,
				ConflictType:   conflictType,
				Scope:          ScopeCue,
				Description:    description,
				FieldConflicts: result.FieldConflicts,
				Resolved:       false,
			}
			conflicts = append(conflicts, conflict)
			log.Debug("Identified conflict for cue", "cue_number", cueNumber, "type", conflictType)
		}
	}

	return conflicts, nil
}

// identifyConflictsFromScope recursively identifies conflicts from scope comparison
func (q *Workspace) identifyConflictsFromScope(scope *ScopeComparison) []CueConflict {
	var conflicts []CueConflict

	if scope == nil {
		return conflicts
	}

	// Check if this scope has conflicts
	if scope.ConflictExists {
		// Build list of conflicting properties
		properties := make([]string, 0, len(scope.FieldChanges))
		fieldConflicts := make(map[string]*FieldConflict)

		for fieldName, fieldConflict := range scope.FieldChanges {
			if q.isFieldConflict(fieldConflict) {
				properties = append(properties, fieldName)
				fieldConflicts[fieldName] = fieldConflict
			}
		}

		if len(properties) > 0 {
			var conflictType ConflictType
			var description string

			// Determine conflict type based on field changes
			hasSourceChanges := false
			hasQLabChanges := false

			for _, fc := range fieldConflicts {
				sourceNorm := q.normalizeProperty(fc.SourceValue)
				cacheNorm := q.normalizeProperty(fc.CacheValue)
				qlabNorm := q.normalizeProperty(fc.QLabValue)

				if !q.comparePropertyValues(fc.FieldName, sourceNorm, cacheNorm) {
					hasSourceChanges = true
				}
				if !q.comparePropertyValues(fc.FieldName, qlabNorm, cacheNorm) {
					hasQLabChanges = true
				}
			}

			if hasSourceChanges && hasQLabChanges {
				conflictType = ConflictThreeWayDivergence
				description = fmt.Sprintf("%s '%s' has conflicting changes in source and QLab (fields: %v)",
					scope.Scope, scope.Identifier, properties)
			} else if hasQLabChanges {
				conflictType = ConflictCacheStale
				description = fmt.Sprintf("%s '%s' modified in QLab (fields: %v)",
					scope.Scope, scope.Identifier, properties)
			} else if hasSourceChanges {
				conflictType = ConflictSourceModified
				description = fmt.Sprintf("%s '%s' modified in source (fields: %v)",
					scope.Scope, scope.Identifier, properties)
			}

			conflict := CueConflict{
				CueNumber:      scope.Identifier,
				CueIdentifier:  scope.Identifier,
				ConflictType:   conflictType,
				Scope:          scope.Scope,
				Properties:     properties,
				FieldConflicts: fieldConflicts,
				Description:    description,
				Resolved:       false,
			}

			conflicts = append(conflicts, conflict)
			log.Debugf("Identified %s-level conflict: %s (%d fields)", scope.Scope, scope.Identifier, len(properties))
		}
	}

	// Recursively check child scopes
	for _, childScope := range scope.ChildScopes {
		childConflicts := q.identifyConflictsFromScope(childScope)
		conflicts = append(conflicts, childConflicts...)
	}

	return conflicts
}

// PromptUserForConflictResolution uses huh to prompt the user for conflict resolution choices
func (q *Workspace) PromptUserForConflictResolution(conflicts []CueConflict, comparison *ThreeWayComparison) error {
	if len(conflicts) == 0 {
		return nil
	}

	log.Infof("Found %d conflicts that require your attention", len(conflicts))

	for i, conflict := range conflicts {
		log.Infof("Conflict %d/%d: %s", i+1, len(conflicts), conflict.Description)

		var choice string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(fmt.Sprintf("How would you like to resolve the conflict for cue %s?", conflict.CueNumber)).
					Description(conflict.Description).
					Options(
						huh.NewOption("Use source file version (overwrite QLab)", "source"),
						huh.NewOption("Keep QLab version (overwrite source)", "qlab"),
						huh.NewOption("Skip this cue (no changes)", "skip"),
					).
					Value(&choice),
			),
		)

		err := form.Run()
		if err != nil {
			return fmt.Errorf("failed to get user input for conflict resolution: %v", err)
		}

		// Apply the user's choice by modifying the comparison results
		if result, exists := comparison.CueResults[conflict.CueNumber]; exists {
			switch choice {
			case "source":
				result.Action = "update"
				result.Reason = "User chose to use source file version"
				log.Infof("User chose to use source version for cue %s", conflict.CueNumber)
			case "qlab":
				result.Action = "skip"
				result.Reason = "User chose to keep QLab version"
				comparison.QLabChosenCues[conflict.CueNumber] = true
				log.Infof("User chose to keep QLab version for cue %s", conflict.CueNumber)
			case "skip":
				result.Action = "skip"
				result.Reason = "User chose to skip this cue"
				log.Infof("User chose to skip cue %s", conflict.CueNumber)
			default:
				return fmt.Errorf("unexpected choice: %s", choice)
			}
		}
	}

	log.Info("All conflicts resolved by user")
	return nil
}

// getMapKeys helper function to get keys from a map for logging
func getMapKeys(m map[string]map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// processCueListWithMappingAndChangeDetection processes cues with change detection support
func (q *Workspace) processCueListWithMappingAndChangeDetection(cueData map[string]any, parentNumber string, mapping *CueMapping, changeResults map[string]*CueChangeResult) error {
	log.Debug("Wrapper function calling processCueListWithParentMappingAndChangeDetection")
	uniqueID, err := q.processCueListWithParentMappingAndChangeDetection(cueData, parentNumber, "", mapping, changeResults)
	log.Debug("Wrapper function returned", "unique_id", uniqueID, "error", err)
	return err
}

// processCueListWithParentMappingAndChangeDetection recursively processes cues with change detection
func (q *Workspace) processCueListWithParentMappingAndChangeDetection(cueData map[string]any, parentNumber string, parentUniqueID string, mapping *CueMapping, changeResults map[string]*CueChangeResult) (string, error) {
	return q.processCueListWithParentMappingAndChangeDetectionWithIndex(cueData, parentNumber, parentUniqueID, mapping, changeResults, -1)
}

// processCueListWithParentMappingAndChangeDetectionWithIndex recursively processes cues with change detection and position tracking
func (q *Workspace) processCueListWithParentMappingAndChangeDetectionWithIndex(cueData map[string]any, parentNumber string, parentUniqueID string, mapping *CueMapping, changeResults map[string]*CueChangeResult, cueIndex int) (string, error) {
	cueType, _ := cueData["type"].(string)
	cueName, _ := cueData["name"].(string)

	// Debug: Print cue data structure
	keys := make([]string, 0, len(cueData))
	for k := range cueData {
		keys = append(keys, k)
	}
	log.Debug("Processing cue", "type", cueType, "name", cueName, "parent", parentNumber, "keys", keys)

	// Check if this cue list already exists (for duplicate prevention)
	var existingCueListID string
	if cueType == "list" && cueName != "" {
		log.Debug("Checking for existing cue list", "name", cueName)
		if existingID, exists := q.cueListNames[cueName]; exists {
			log.Debug("Found existing cue list, will use existing and process sub-cues", "name", cueName, "type", cueType, "id", existingID)
			existingCueListID = existingID
		} else {
			log.Debug("Cue list does not exist yet, will create new one", "name", cueName)
		}
	}

	log.Debug("Past duplicate check, extracting cue number")

	var cueNumber string
	if num, ok := cueData["number"]; ok && num != nil {
		// Handle different number types while preserving decimal format
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

	log.Debug("Extracted cue number from cue data", "cue_number", cueNumber)

	// Build full cue number with parent prefix
	fullNumber := cueNumber
	if parentNumber != "" && cueNumber != "" {
		if strings.Contains(cueNumber, ".") {
			fullNumber = cueNumber
		} else {
			fullNumber = parentNumber + "." + cueNumber
		}
	}

	// Check change detection results for this cue
	var uniqueID string
	var err error

	log.Debug("About to check change detection for cue", "full_number", fullNumber, "cue_name", cueName)

	// Generate position-based key for cues without numbers (same logic as indexing)
	var positionKey string
	if fullNumber == "" && cueIndex >= 0 {
		if parentNumber != "" {
			positionKey = fmt.Sprintf("%s@%d[%s:%s]", parentNumber, cueIndex, strings.ToLower(cueType), cueName)
		} else {
			positionKey = fmt.Sprintf("@%d[%s:%s]", cueIndex, strings.ToLower(cueType), cueName)
		}
		log.Debug("Generated position key for numberless cue", "position_key", positionKey, "parent", parentNumber, "index", cueIndex, "type", cueType, "name", cueName)
	}

	// Check change detection results using number first, then position key as fallback
	var changeResult *CueChangeResult
	var lookupKey string
	if fullNumber != "" {
		if result, exists := changeResults[fullNumber]; exists {
			changeResult = result
			lookupKey = fullNumber
		}
	} else if positionKey != "" {
		if result, exists := changeResults[positionKey]; exists {
			changeResult = result
			lookupKey = positionKey
		}
	}

	if changeResult != nil {
		log.Debug("Found change result for cue", "lookup_key", lookupKey, "action", changeResult.Action)

		switch changeResult.Action {
		case "skip":
			// Cue hasn't changed, skip creation and hierarchy processing
			log.Infof("Skipping unchanged cue: [%s] %s (%s) - %s", lookupKey, cueName, cueType, changeResult.Reason)
			uniqueID = changeResult.ExistingID
			if fullNumber != "" && uniqueID != "" {
				mapping.NumberToID[fullNumber] = uniqueID
			}
			// Early return to avoid move operations and sub-cue processing
			return uniqueID, nil

		case "update":
			// Update existing cue with changed properties
			log.Infof("Updating changed cue: [%s] %s (%s) - %s", lookupKey, cueName, cueType, changeResult.Reason)
			uniqueID = changeResult.ExistingID
			if uniqueID == "" {
				return "", fmt.Errorf("cannot update cue %s: no existing ID provided", lookupKey)
			}

			// Update the cue properties
			err = q.updateCueProperties(uniqueID, cueData)
			if err != nil {
				log.Debug("ERROR - Failed to update cue", "lookup_key", lookupKey, "uniqueID", uniqueID, "error", err)
				return "", fmt.Errorf("failed to update cue %s: %v", lookupKey, err)
			}
			log.Debug("Successfully updated cue", "lookup_key", lookupKey, "uniqueID", uniqueID)

			if fullNumber != "" && uniqueID != "" {
				mapping.NumberToID[fullNumber] = uniqueID
			}

		case "create":
			// Create new cue
			log.Debug("PROCESSING CREATE ACTION for cue", "lookup_key", lookupKey, "name", cueName, "type", cueType, "reason", changeResult.Reason)
			uniqueID, err = q.createCueWithoutTarget(cueData, fullNumber)
			if err != nil {
				log.Debug("ERROR - Failed to create cue", "lookup_key", lookupKey, "error", err)
				return "", fmt.Errorf("failed to create cue %s: %v", lookupKey, err)
			}
			log.Debug("Successfully created cue", "lookup_key", lookupKey, "uniqueID", uniqueID)
		default:
			// Create new cue
			log.Infof("Creating new cue: [%s] %s (%s) - %s", lookupKey, cueName, cueType, changeResult.Reason)
			uniqueID, err = q.createCueWithoutTarget(cueData, fullNumber)
			if err != nil {
				return "", fmt.Errorf("failed to create cue %s: %v", lookupKey, err)
			}
		}
	} else {
		// No change detection data available
		log.Debug("No change detection data found for cue, checking if cue already exists", "number", fullNumber)

		// Check if we already found this cue list exists
		if existingCueListID != "" {
			log.Infof("Using existing cue list: %s (%s) - ID %s", cueName, cueType, existingCueListID)
			uniqueID = existingCueListID

			// Return early - don't process sub-cues or move operations for existing cue lists
			if fullNumber != "" && uniqueID != "" {
				mapping.NumberToID[fullNumber] = uniqueID
			}
			return uniqueID, nil
		} else {
			// Create new cue
			if cueName != "" {
				if fullNumber != "" {
					log.Infof("Creating new cue (no change data): [%s] %s (%s)", fullNumber, cueName, cueType)
				} else {
					log.Infof("Creating new cue (no change data): %s (%s)", cueName, cueType)
				}
			}
			uniqueID, err = q.createCueWithoutTarget(cueData, fullNumber)
			if err != nil {
				log.Debug("ERROR - Failed to create cue in no-change-data path", "error", err)
				return "", fmt.Errorf("failed to create cue %s: %v", fullNumber, err)
			}
			log.Debug("Successfully created cue (no change data)", "number", fullNumber, "uniqueID", uniqueID)
		}
	}

	// Index newly created cue lists by name for duplicate prevention
	if cueType == "list" && cueName != "" && uniqueID != "" {
		q.cueListNames[cueName] = uniqueID
	}

	// Add to mapping if we have a cue number and ID
	if fullNumber != "" && uniqueID != "" {
		mapping.NumberToID[fullNumber] = uniqueID
	}

	// Check if this cue has a target that needs to be set later
	if targetNumber, ok := cueData["cueTargetNumber"].(string); ok && targetNumber != "" && uniqueID != "" {
		mapping.CuesWithTargets = append(mapping.CuesWithTargets, CueTarget{
			UniqueID:     uniqueID,
			TargetNumber: targetNumber,
		})
	}

	// Move cue into parent group if we have a parent
	if parentUniqueID != "" && uniqueID != "" {
		// Check if parent is an existing cue list - if so, skip move operation
		isExistingCueList := false
		for _, existingID := range q.cueListNames {
			if existingID == parentUniqueID {
				isExistingCueList = true
				break
			}
		}

		if isExistingCueList {
			log.Debug("Skipping move operation - parent is an existing cue list that cannot accept new cues", "parentUniqueID", parentUniqueID)
		} else {
			err = q.moveCueToParent(uniqueID, parentUniqueID)
			if err != nil {
				return "", fmt.Errorf("failed to move cue %s into parent %s: %v", uniqueID, parentUniqueID, err)
			}
		}
	}

	// Process sub-cues if they exist
	if cuesValue, exists := cueData["cues"]; exists {
		log.Debug("Found 'cues' field in cue data", "number", fullNumber)
		if subCues, ok := cuesValue.([]any); ok {
			log.Debug("Processing sub-cues for parent cue", "count", len(subCues), "parentNumber", fullNumber)
			if uniqueID != "" {
				for childIndex, subCueData := range subCues {
					if subCue, ok := subCueData.(map[string]any); ok {
						log.Debug("Processing sub-cue for parent", "childIndex", childIndex+1, "totalSubCues", len(subCues), "parentNumber", fullNumber)
						childUniqueID, err := q.processCueListWithParentMappingAndChangeDetectionWithIndex(subCue, fullNumber, "", mapping, changeResults, childIndex)
						if err != nil {
							log.Debug("ERROR - Failed to process sub-cue", "childIndex", childIndex, "error", err)
							return "", fmt.Errorf("error processing sub-cue %d: %v", childIndex, err)
						}

						// Move the child cue into this parent group at the correct index
						if childUniqueID != "" {
							// Check if this child cue was skipped (unchanged) - if so, don't move it
							shouldSkipMove := false

							// Generate the same lookup keys that would be used for this child cue
							childCueType, _ := subCue["type"].(string)
							childCueName, _ := subCue["name"].(string)
							childFullNumber, _ := subCue["number"].(string)

							var childLookupKey string
							if childFullNumber != "" {
								childLookupKey = childFullNumber
							} else {
								// Generate position key for numberless child cue
								if fullNumber != "" {
									childLookupKey = fmt.Sprintf("%s@%d[%s:%s]", fullNumber, childIndex, strings.ToLower(childCueType), childCueName)
								} else {
									childLookupKey = fmt.Sprintf("@%d[%s:%s]", childIndex, strings.ToLower(childCueType), childCueName)
								}
							}

							// Check if this child was skipped
							if childChangeResult, exists := changeResults[childLookupKey]; exists && childChangeResult.Action == "skip" {
								shouldSkipMove = true
								log.Debug("Skipping move for unchanged child cue", "childLookupKey", childLookupKey, "childUniqueID", childUniqueID)
							}

							if shouldSkipMove {
								// Child cue is already in correct position, don't move it
							} else {
								// Check if parent is an existing cue list - if so, skip move operation
								isExistingCueList := false
								for _, existingID := range q.cueListNames {
									if existingID == uniqueID {
										isExistingCueList = true
										break
									}
								}

								if isExistingCueList {
									log.Debug("Skipping child move operation - parent is an existing cue list that cannot accept moved cues", "parentUniqueID", uniqueID)
								} else {
									log.Debug("Moving child cue into parent", "childUniqueID", childUniqueID, "parentUniqueID", uniqueID, "index", childIndex)
									err = q.moveCueToParentWithIndex(childUniqueID, uniqueID, childIndex)
									if err != nil {
										log.Debug("ERROR - Failed to move child cue", "error", err)
										return "", fmt.Errorf("failed to move child cue %s into parent %s at index %d: %v", childUniqueID, uniqueID, childIndex, err)
									}
								}
							}
						}
					} else {
						log.Debug("WARNING - Sub-cue is not a valid map", "childIndex", childIndex)
					}
				}
			} else {
				log.Debug("WARNING - Parent cue has no uniqueID, cannot process sub-cues")
			}
		} else {
			log.Debug("WARNING - 'cues' field exists but is not an array", "number", fullNumber)
		}
	} else {
		log.Debug("No 'cues' field found in cue data", "number", fullNumber)
	}

	return uniqueID, nil
}
