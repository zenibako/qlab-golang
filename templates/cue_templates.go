package templates

// CueTemplate represents a template for generating QLab cues
type CueTemplate struct {
	Type       string         `json:"type"`       // QLab cue type: "light", "audio", "group", etc.
	Name       string         `json:"name"`       // Cue name
	Properties map[string]any `json:"properties"` // QLab-specific properties
	Children   []CueTemplate  `json:"children"`   // Child cues (for groups)
}

// CueGenerationRequest represents a request to generate cues
type CueGenerationRequest struct {
	AnchorID  string      `json:"anchor_id"`
	CueNumber string      `json:"cue_number"`
	Template  CueTemplate `json:"template"`
	ParentID  string      `json:"parent_id,omitempty"` // Optional: where to insert in hierarchy
}

// CueGenerationResult represents the result of cue generation
type CueGenerationResult struct {
	Success     bool         `json:"success"`
	CuesCreated []CreatedCue `json:"cues_created,omitempty"`
	Errors      []string     `json:"errors,omitempty"`
}

// CreatedCue represents a successfully created cue
type CreatedCue struct {
	UniqueID  string `json:"unique_id"`
	CueNumber string `json:"cue_number"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	ParentID  string `json:"parent_id,omitempty"`
}
