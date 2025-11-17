// QLab Cue Type Schemas
// This file defines the complete schema for all QLab cue types based on the OSC dictionary.
// Property names exactly match QLab OSC addresses (minus the /cue/{cue_number}/ prefix).
//
// Reference: QLab OSC Dictionary v5

package schemas

// Base cue definition - all cues extend this
#Cue: {
	// === REQUIRED FIELDS ===
	type: string // Cue type: "audio", "video", "text", "light", "fade", "group", etc.
	
	// === IDENTIFICATION ===
	number:    string | *"" // Cue number (e.g., "1", "2.5", "10")
	uniqueID:  string | *"" // Unique identifier assigned by QLab (read-only)
	name:      string | *"" // Display name
	listName:  string | *name // List name (defaults to name)
	
	// === STATUS ===
	flagged:   string | *"" // "" for false, "true" for true
	armed:     string | *"" // "" for false, "true" for true
	
	// === DISPLAY ===
	colorName:        string | *"none" // Color: "red", "orange", "yellow", "green", "blue", "purple", "pink", "gray", "none"
	"colorName/live": string | *"none" // Live color (changes during playback)
	
	// === DOCUMENTATION ===
	notes: string | *"" // Cue notes
	
	// === TIMING ===
	duration:     string | *"" // Duration in seconds as string (e.g., "5.0")
	preWait:      string | *"" // Pre-wait time in seconds
	postWait:     string | *"" // Post-wait time in seconds
	continueMode: int | *0     // 0=Do not continue, 1=Auto-continue, 2=Auto-follow
	
	// === TARGETING ===
	cueTargetID:     string | *"" // Target cue unique ID
	cueTargetNumber: string | *"" // Target cue number
	fileTarget:      string | *"" // File path for audio/video/image cues
	
	// === NESTED CUES ===
	cues?: [...#Cue] // Child cues for groups
	
	// Allow additional fields for specific cue types
	...
}

// === AUDIO CUES ===

#AudioCue: #Cue & {
	type: "audio"
	
	// Audio-specific properties
	infiniteLoop?:      bool | *false
	playCount?:         int | *0
	rate?:              number | *1.0
	"startTime"?:       string
	"endTime"?:         string
	level?:             number
	gang?:              bool
	...
}

// === VIDEO CUES ===

#VideoCue: #Cue & {
	type: "video"
	
	// Video-specific properties
	infiniteLoop?:      bool | *false
	playCount?:         int | *0
	rate?:              number | *1.0
	"startTime"?:       string
	"endTime"?:         string
	layer?:             int
	fullScreen?:        bool
	translation?:       [number, number] // [x, y]
	scale?:             [number, number] // [x, y]
	rotation?:          number
	opacity?:           number | *1.0
	...
}

// === TEXT CUES ===

#TextCue: #Cue & {
	type: "text"
	
	// Text content
	text: string | *""
	
	// Text formatting - use OSC address format with quotes
	"text/format/color"?:           [number, number, number, number] | *[1, 1, 1, 1] // [R, G, B, A] 0.0-1.0
	"text/format/backgroundColor"?: [number, number, number, number] | *[0, 0, 0, 0] // [R, G, B, A] 0.0-1.0
	"text/format/fontSize"?:        number | *72
	"text/format/alignment"?:       string | *"center" // "left", "center", "right"
	"text/format/font"?:            string
	
	// Text geometry
	translation?: [number, number] // [x, y]
	scale?:       [number, number] // [x, y]
	rotation?:    number
	opacity?:     number | *1.0
	
	// Stage assignment
	stageName?: string
	stageID?:   string
	
	...
}

// === LIGHT CUES ===

#LightCue: #Cue & {
	type: "light"
	
	// Light-specific properties
	patchNumber?:      string
	patchTargetID?:    string
	...
}

// === FADE CUES ===

#FadeCue: #Cue & {
	type: "fade"
	
	// Fade target is required
	cueTargetNumber: string
	
	// Fade parameters - what to fade
	doOpacity?:      bool | *false
	doTranslation?:  bool | *false
	doScale?:        bool | *false
	doRotation?:     bool | *false
	doLevel?:        bool | *false // Audio level
	doColor?:        bool | *false // Text color
	
	// Fade target values
	opacity?:                       number
	translation?:                   [number, number]
	scale?:                         [number, number]
	rotation?:                      number
	level?:                         number
	"text/format/color"?:           [number, number, number, number]
	"text/format/backgroundColor"?: [number, number, number, number]
	
	// Fade curve
	curve?: string | *"linear" // "linear", "exponential", "logarithmic", "s-curve"
	
	...
}

// === START/STOP/PAUSE CUES ===

#StartCue: #Cue & {
	type:            "start"
	cueTargetNumber: string // Required - which cue to start
	...
}

#StopCue: #Cue & {
	type:            "stop"
	cueTargetNumber: string // Required - which cue to stop
	...
}

#PauseCue: #Cue & {
	type:            "pause"
	cueTargetNumber: string // Required - which cue to pause
	...
}

#LoadCue: #Cue & {
	type:            "load"
	cueTargetNumber: string // Required - which cue to load
	...
}

#ResetCue: #Cue & {
	type:            "reset"
	cueTargetNumber: string // Required - which cue to reset
	...
}

// === GROUP CUES ===

#GroupCue: #Cue & {
	type: "group"
	mode: int & >=0 & <=6 // Group mode determines behavior
	
	// Group-specific properties
	cues: [...#Cue] // Child cues (required for groups)
	
	...
}

// Specific group types by mode

#CueList: #GroupCue & {
	mode: 0
}

#StartFirstAndEnterGroup: #GroupCue & {
	mode: 1
}

#StartFirstGroup: #GroupCue & {
	mode: 2
}

#Timeline: #GroupCue & {
	mode: 3
}

#StartRandomGroup: #GroupCue & {
	mode: 4
}

#Cart: #GroupCue & {
	mode:         5
	cartRows?:    int
	cartColumns?: int
}

#Playlist: #GroupCue & {
	mode: 6
}

// === CAMERA CUES ===

#CameraCue: #Cue & {
	type: "camera"
	...
}

// === MIC CUES ===

#MicCue: #Cue & {
	type: "mic"
	...
}

// === MIDI CUES ===

#MIDICue: #Cue & {
	type:         "midi"
	midiCommand?: string
	...
}

#MIDIFileCue: #Cue & {
	type: "midi-file"
	...
}

#MSCCue: #Cue & {
	type: "msc"
	...
}

// === TIMECODE CUES ===

#TimecodeCue: #Cue & {
	type: "timecode"
	...
}

// === NETWORK CUES ===

#NetworkCue: #Cue & {
	type:          "network"
	"messageType"?: string
	"customString"?: string
	...
}

#OSCCue: #Cue & {
	type:           "osc"
	"customString"?: string
	...
}

// === SCRIPT CUES ===

#ScriptCue: #Cue & {
	type:   "script"
	script?: string
	...
}

// === GOTO CUES ===

#GotoCue: #Cue & {
	type:            "goto"
	cueTargetNumber: string
	...
}

#TargetCue: #Cue & {
	type: "target"
	...
}

// === MEMO CUES ===

#MemoCue: #Cue & {
	type: "memo"
	// Memos only have basic cue properties, no special fields
	...
}

// === DEVAMP CUES ===

#DevampCue: #Cue & {
	type: "devamp"
	...
}

#ArmsModeCue: #Cue & {
	type: "arms"
	...
}

#WaitCue: #Cue & {
	type: "wait"
	...
}

// Workspace definition
#Workspace: {
	name:     string
	uniqueID: string | *""
	
	// Connection settings
	host?:     string
	port?:     int
	passcode?: int
	
	// Workspace settings
	settings?: [...]
	
	// Root-level cues
	cues: [...#Cue]
	
	// Allow additional fields
	...
}
