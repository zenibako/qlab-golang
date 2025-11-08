package messages

import (
	"fmt"
	"strings"
)

// OSC Message types and address constants based on lib/qlab/osc/ dictionary

// Message types
type MessageType string

const (
	// Application messages
	MsgConnect    MessageType = "connect"
	MsgDisconnect MessageType = "disconnect"

	// Workspace messages
	MsgWorkspaceConnect    MessageType = "workspace_connect"
	MsgWorkspaceNew        MessageType = "workspace_new"
	MsgWorkspaceBasePath   MessageType = "workspace_base_path"
	MsgWorkspaceWorkingDir MessageType = "workspace_working_directory"

	// Cue messages
	MsgCueName         MessageType = "cue_name"
	MsgCueNumber       MessageType = "cue_number"
	MsgCueType         MessageType = "cue_type"
	MsgCueFileTarget   MessageType = "cue_file_target"
	MsgCueCueTarget    MessageType = "cue_cue_target"
	MsgCueInfiniteLoop MessageType = "cue_infinite_loop"
	MsgCueMode         MessageType = "cue_mode"
)

// OSC Address patterns
const (
	// Application level
	AddrConnect = "/connect"

	// Workspace level
	AddrWorkspaceConnect    = "/workspace/{id}/connect"
	AddrWorkspaceNew        = "/workspace/{id}/new"
	AddrWorkspaceBasePath   = "/workspace/{id}/basePath"
	AddrWorkspaceWorkingDir = "/workingDirectory"

	// Cue level (by number)
	AddrCueName         = "/cue/{cue_number}/name"
	AddrCueNumber       = "/cue/{cue_number}/number"
	AddrCueType         = "/cue/{cue_number}/type"
	AddrCueFileTarget   = "/cue/{cue_number}/fileTarget"
	AddrCueCueTarget    = "/cue/{cue_number}/cueTargetID"
	AddrCueInfiniteLoop = "/cue/{cue_number}/infiniteLoop"
	AddrCueMode         = "/cue/{cue_number}/mode"

	// Cue level (by uniqueID)
	AddrCueIDName         = "/cue_id/{unique_id}/name"
	AddrCueIDNumber       = "/cue_id/{unique_id}/number"
	AddrCueIDType         = "/cue_id/{unique_id}/type"
	AddrCueIDFileTarget   = "/cue_id/{unique_id}/fileTarget"
	AddrCueIDCueTarget    = "/cue_id/{unique_id}/cueTargetID"
	AddrCueIDInfiniteLoop = "/cue_id/{unique_id}/infiniteLoop"
	AddrCueIDMode         = "/cue_id/{unique_id}/mode"
)

// Property mapping for cue properties
var CuePropertyMap = map[string]string{
	"name":         "name",
	"number":       "number",
	"type":         "type",
	"file":         "fileTarget",
	"fileTarget":   "fileTarget",
	"cueTarget":    "cueTargetID",
	"infiniteLoop": "infiniteLoop",
	"mode":         "mode",
}

// OSCAddressBuilder builds OSC addresses from message types and parameters
type OSCAddressBuilder struct {
	workspaceID string
}

// NewOSCAddressBuilder creates a new address builder
func NewOSCAddressBuilder(workspaceID string) *OSCAddressBuilder {
	return &OSCAddressBuilder{
		workspaceID: workspaceID,
	}
}

// BuildAddress builds an OSC address from a message type and parameters
func (b *OSCAddressBuilder) BuildAddress(msgType MessageType, params map[string]string) string {
	var address string

	switch msgType {
	case MsgConnect:
		address = AddrConnect
	case MsgWorkspaceConnect:
		address = AddrWorkspaceConnect
	case MsgWorkspaceNew:
		address = AddrWorkspaceNew
	case MsgWorkspaceBasePath:
		address = AddrWorkspaceBasePath
	case MsgWorkspaceWorkingDir:
		address = AddrWorkspaceWorkingDir
	case MsgCueName:
		address = AddrCueName
	case MsgCueNumber:
		address = AddrCueNumber
	case MsgCueType:
		address = AddrCueType
	case MsgCueFileTarget:
		address = AddrCueFileTarget
	case MsgCueCueTarget:
		address = AddrCueCueTarget
	case MsgCueInfiniteLoop:
		address = AddrCueInfiniteLoop
	case MsgCueMode:
		address = AddrCueMode
	default:
		return ""
	}

	// Replace workspace ID if needed
	if strings.Contains(address, "{id}") && b.workspaceID != "" {
		address = strings.ReplaceAll(address, "{id}", b.workspaceID)
	}

	// Replace other parameters
	for key, value := range params {
		placeholder := fmt.Sprintf("{%s}", key)
		address = strings.ReplaceAll(address, placeholder, value)
	}

	return address
}

// BuildCuePropertyAddress builds an address for setting cue properties by uniqueID
func (b *OSCAddressBuilder) BuildCuePropertyAddress(uniqueID, property string) string {
	if b.workspaceID == "" {
		return ""
	}

	// Map property names to OSC property names
	oscProperty, exists := CuePropertyMap[property]
	if !exists {
		oscProperty = property
	}

	address := fmt.Sprintf("/workspace/%s/cue_id/%s/%s", b.workspaceID, uniqueID, oscProperty)
	return address
}

// BuildReplyAddress builds a reply address for a given request address
func (b *OSCAddressBuilder) BuildReplyAddress(requestAddress string) string {
	return "/reply" + requestAddress
}

// GetWorkspacePrefix returns the workspace prefix for addresses that need it
func (b *OSCAddressBuilder) GetWorkspacePrefix() string {
	if b.workspaceID == "" {
		return ""
	}
	return fmt.Sprintf("/workspace/%s", b.workspaceID)
}
