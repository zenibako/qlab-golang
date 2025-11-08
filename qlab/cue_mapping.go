package qlab

// CueMapping tracks the relationship between cue numbers and unique IDs
type CueMapping struct {
	NumberToID      map[string]string // cue number -> unique ID
	CuesWithTargets []CueTarget       // cues that need target setting after creation
}

// CueTarget represents a cue that needs its target set after creation
type CueTarget struct {
	UniqueID     string
	TargetNumber string
}
