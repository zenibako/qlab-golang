package qlab

// Cue represents a generic QLab cue with all possible properties.
// Different cue types will use different subsets of these fields.
type Cue struct {
	// Common properties (all cue types)
	Type          string `json:"type"`
	Name          string `json:"name,omitempty"`
	Number        string `json:"number,omitempty"`
	UniqueID      string `json:"uniqueID,omitempty"`
	Flagged       bool   `json:"flagged,omitempty"`
	ListName      string `json:"listName,omitempty"`
	ColorName     string `json:"colorName,omitempty"`
	ColorNameLive string `json:"colorName/live,omitempty"`
	Armed         bool   `json:"armed,omitempty"`
	Notes         string `json:"notes,omitempty"`

	// Timing properties
	Duration     float64 `json:"duration,omitempty"`
	PreWait      float64 `json:"preWait,omitempty"`
	PostWait     float64 `json:"postWait,omitempty"`
	ContinueMode int     `json:"continueMode,omitempty"` // 0=none, 1=auto-continue, 2=auto-follow

	// Target properties (for Start, Stop, Fade cues, etc.)
	CueTargetNumber string `json:"cueTargetNumber,omitempty"`
	CueTargetID     string `json:"cueTargetID,omitempty"`
	FileTarget      string `json:"fileTarget,omitempty"`

	// Group/List properties
	Mode         int   `json:"mode,omitempty"` // 0=list, 1=start first+enter, 2=start first, 3=timeline, 4=start random, 5=cart, 6=playlist
	InfiniteLoop bool  `json:"infiniteLoop,omitempty"`
	Cues         []Cue `json:"cues,omitempty"` // Child cues for Group/List cues

	// Text cue properties
	Text          string    `json:"text,omitempty"`                        // Text content
	TextColor     []float64 `json:"text/format/color,omitempty"`           // [R, G, B, A] 0.0-1.0
	TextBgColor   []float64 `json:"text/format/backgroundColor,omitempty"` // [R, G, B, A] 0.0-1.0
	TextFontSize  float64   `json:"text/format/fontSize,omitempty"`        // Font size in points
	TextAlignment string    `json:"text/format/alignment,omitempty"`       // "left", "center", "right", "justify"

	// Video/Text cue stage properties
	StageID   string `json:"stageID,omitempty"`   // Video stage unique ID
	StageName string `json:"stageName,omitempty"` // Video stage name

	// Geometry properties (Video, Camera, Text cues)
	Translation  []float64 `json:"translation,omitempty"`  // [x, y] position on stage
	Scale        []float64 `json:"scale,omitempty"`        // [x, y] scaling factors
	Rotation     float64   `json:"rotation,omitempty"`     // Single-axis rotation in degrees
	RotationType int       `json:"rotationType,omitempty"` // 0=3D orientation, 1=X, 2=Y, 3=Z
	Quaternion   []float64 `json:"quaternion,omitempty"`   // [a, b, c, d] for 3D rotation
	Opacity      float64   `json:"opacity,omitempty"`      // 0.0 to 1.0

	// Fade cue geometry parameter enables (checkboxes)
	DoOpacity     bool `json:"doOpacity,omitempty"`     // Enable opacity fading
	DoTranslation bool `json:"doTranslation,omitempty"` // Enable translation fading
	DoScale       bool `json:"doScale,omitempty"`       // Enable scale fading
	DoRotation    bool `json:"doRotation,omitempty"`    // Enable rotation fading
}

// WorkspaceData represents the parsed workspace structure
type WorkspaceData struct {
	Name string `json:"name"`
	Cues []Cue  `json:"cues"`
}

// CueType constants for type-safe cue type checking
const (
	CueTypeAudio      = "audio"
	CueTypeVideo      = "video"
	CueTypeText       = "text"
	CueTypeFade       = "fade"
	CueTypeStart      = "start"
	CueTypeStop       = "stop"
	CueTypePause      = "pause"
	CueTypeReset      = "reset"
	CueTypeDevamp     = "devamp"
	CueTypeGoto       = "goto"
	CueTypeTarget     = "target"
	CueTypeGroup      = "group"
	CueTypeMemo       = "memo"
	CueTypeScript     = "script"
	CueTypeMIDI       = "midi"
	CueTypeMIDIFile   = "midi file"
	CueTypeTimecode   = "timecode"
	CueTypeNetwork    = "network"
	CueTypeMSC        = "msc"
	CueTypeCamera     = "camera"
	CueTypeMicrophone = "microphone"
	CueTypeList       = "cue list"
	CueTypeCart       = "cart"
)

// TextAlignment constants
const (
	TextAlignLeft    = "left"
	TextAlignCenter  = "center"
	TextAlignRight   = "right"
	TextAlignJustify = "justify"
)

// RotationType constants
const (
	RotationType3D = 0 // 3D orientation (quaternion)
	RotationTypeX  = 1 // X-axis rotation
	RotationTypeY  = 2 // Y-axis rotation
	RotationTypeZ  = 3 // Z-axis rotation
)

// ContinueMode constants
const (
	ContinueModeNone         = 0 // No continue
	ContinueModeAutoContinue = 1 // Auto-continue
	ContinueModeAutoFollow   = 2 // Auto-follow
)

// GroupMode constants
const (
	GroupModeList               = 0 // Cue list
	GroupModeStartFirstAndEnter = 1 // Start first and enter
	GroupModeStartFirst         = 2 // Start first
	GroupModeTimeline           = 3 // Timeline
	GroupModeStartRandom        = 4 // Start random
	GroupModeCart               = 5 // Cart
	GroupModePlaylist           = 6 // Playlist
)
