package qlab

import (
	"encoding/json"
	"fmt"

	"github.com/zenibako/qlab-golang/templates"

	"github.com/charmbracelet/log"
)

// CueGenerator handles the generation of QLab cues via OSC
type CueGenerator struct {
	workspace *Workspace
}

// NewCueGenerator creates a new cue generator
func NewCueGenerator(workspace *Workspace) *CueGenerator {
	return &CueGenerator{
		workspace: workspace,
	}
}

// GenerateCues creates cues in QLab based on a template
func (cg *CueGenerator) GenerateCues(request templates.CueGenerationRequest) templates.CueGenerationResult {
	result := templates.CueGenerationResult{
		Success:     true,
		CuesCreated: []templates.CreatedCue{},
		Errors:      []string{},
	}

	// Create the cue(s) from the template
	cuesCreated, err := cg.createCueFromTemplate(request.Template, request.CueNumber, request.ParentID)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, err.Error())
		return result
	}

	result.CuesCreated = cuesCreated
	return result
}

// createCueFromTemplate creates a cue and its children from a template
func (cg *CueGenerator) createCueFromTemplate(template templates.CueTemplate, cueNumber string, parentID string) ([]templates.CreatedCue, error) {
	var allCreated []templates.CreatedCue

	// Create the main cue
	uniqueID, err := cg.createCue(template.Type, cueNumber, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s cue: %w", template.Type, err)
	}

	log.Info("Created cue", "type", template.Type, "uniqueID", uniqueID, "cueNumber", cueNumber)

	created := templates.CreatedCue{
		UniqueID:  uniqueID,
		CueNumber: cueNumber,
		Name:      template.Name,
		Type:      template.Type,
		ParentID:  parentID,
	}

	// Set cue properties
	if err := cg.setCueProperties(uniqueID, template.Name, template.Properties); err != nil {
		return nil, fmt.Errorf("failed to set properties for cue %s: %w", uniqueID, err)
	}

	allCreated = append(allCreated, created)

	// Create child cues if this is a group
	if len(template.Children) > 0 {
		for i, childTemplate := range template.Children {
			childCueNumber := fmt.Sprintf("%s.%d", cueNumber, i+1)
			childCues, err := cg.createCueFromTemplate(childTemplate, childCueNumber, uniqueID)
			if err != nil {
				return allCreated, fmt.Errorf("failed to create child cue %d: %w", i, err)
			}
			allCreated = append(allCreated, childCues...)
		}
	}

	return allCreated, nil
}

// createCue creates a single cue in QLab and returns its unique ID
func (cg *CueGenerator) createCue(cueType string, cueNumber string, parentID string) (string, error) {
	// Build the OSC address for creating a new cue
	address := cg.workspace.GetAddress("/new")

	// Build the input string - parent ID if provided
	input := cueType
	if parentID != "" {
		input = fmt.Sprintf("%s %s", cueType, parentID)
	}

	// Send the create command
	result := cg.workspace.Send(address, input)

	// Extract the unique ID from the result
	uniqueID := extractUniqueIDFromResult(result)
	if uniqueID == "" {
		return "", fmt.Errorf("failed to extract unique ID from result: %v", result)
	}

	log.Info("Created cue via OSC", "type", cueType, "uniqueID", uniqueID)

	// Set the cue number if provided and different from default
	if cueNumber != "" {
		if err := cg.setCueNumber(uniqueID, cueNumber); err != nil {
			log.Warn("Failed to set cue number", "uniqueID", uniqueID, "cueNumber", cueNumber, "error", err)
			// Don't fail completely if we can't set the number
		}
	}

	return uniqueID, nil
}

// setCueProperties sets the name and other properties of a cue
func (cg *CueGenerator) setCueProperties(uniqueID string, name string, properties map[string]any) error {
	// Set the cue name
	if name != "" {
		if err := cg.setCueName(uniqueID, name); err != nil {
			return fmt.Errorf("failed to set name: %w", err)
		}
	}

	// Set other properties
	for key, value := range properties {
		if key == "mode" && value != nil {
			// Special handling for group mode
			if err := cg.setCueProperty(uniqueID, "mode", value); err != nil {
				log.Warn("Failed to set property", "property", key, "error", err)
			}
		} else if key == "duration" && value != nil {
			// Set duration for fade cues, etc.
			if err := cg.setCueProperty(uniqueID, "duration", value); err != nil {
				log.Warn("Failed to set duration", "error", err)
			}
		}
		// Add more property handlers as needed
	}

	return nil
}

// setCueNumber sets the cue number
func (cg *CueGenerator) setCueNumber(uniqueID string, cueNumber string) error {
	address := cg.workspace.GetAddress(fmt.Sprintf("/cue_id/%s/number", uniqueID))
	cg.workspace.Send(address, cueNumber)
	return nil
}

// setCueName sets the cue name
func (cg *CueGenerator) setCueName(uniqueID string, name string) error {
	address := cg.workspace.GetAddress(fmt.Sprintf("/cue_id/%s/name", uniqueID))
	cg.workspace.Send(address, name)
	return nil
}

// setCueProperty sets a generic cue property
func (cg *CueGenerator) setCueProperty(uniqueID string, property string, value any) error {
	address := cg.workspace.GetAddress(fmt.Sprintf("/cue_id/%s/%s", uniqueID, property))
	cg.workspace.Send(address, fmt.Sprintf("%v", value))
	return nil
}

// extractUniqueIDFromResult extracts the unique ID from an OSC result
func extractUniqueIDFromResult(result []any) string {
	if len(result) == 0 {
		return ""
	}

	// The result is typically a JSON string from QLab
	// Format: [{"data":"UNIQUE-ID","status":"ok"}]
	replyStr, ok := result[0].(string)
	if !ok {
		return ""
	}

	// Parse the JSON reply
	var replyData map[string]any
	err := json.Unmarshal([]byte(replyStr), &replyData)
	if err != nil {
		return ""
	}

	// Extract the unique ID from the "data" field
	if uniqueID, ok := replyData["data"].(string); ok {
		return uniqueID
	}

	return ""
}
