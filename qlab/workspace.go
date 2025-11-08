package qlab

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zenibako/qlab-golang/messages"

	"github.com/charmbracelet/log"
	"github.com/hypebeast/go-osc/osc"
	// Removed cuejitsu dependency
)

type Workspace struct {
	initialized       bool
	host              string
	port              int
	client            *osc.Client
	workspace_id      string
	addressBuilder    *messages.OSCAddressBuilder
	cueNumbers        map[string]string     // Maps cue number -> cue ID for conflict detection
	cueListNames      map[string]string     // Maps cue list name -> cue list ID for duplicate prevention
	inboxID           string                // ID of the "Cuejitsu Inbox" cue list for staging
	forceCueNumbers   bool                  // Whether to force cue number conflicts by clearing existing numbers
	dryRun            bool                  // Whether to run in dry-run mode (no actual changes)
	dryRunCounter     int                   // Counter for generating unique mock IDs in dry-run mode
	replyServer       *osc.Server           // Current reply server for cleanup
	updateServer      *osc.Server           // Persistent server for QLab updates
	replyHandlers     map[string]chan []any // Handlers for reply messages
	replyHandlersMux  sync.Mutex            // Mutex to protect replyHandlers map
	updateHandler     func(string, []any)   // Handler for update messages
	requestCounter    int                   // Counter for generating unique request IDs
	cueListsCache     []any                 // Cached cue lists data to avoid duplicate requests
	videoStagesCache  []map[string]any      // Cached video stages to avoid duplicate queries
	onDisconnect      func()                // Callback for when QLab appears to be disconnected
	wasConnected      bool                  // Tracks if we were previously connected
	consecutiveErrors int                   // Counter for consecutive timeout errors
	serverMux         sync.Mutex            // Mutex to protect server access
	updateServerReady chan struct{}         // Signal that update server is ready
	replyServerReady  chan struct{}         // Signal that reply server is ready
	maxRetries        int                   // Maximum number of retries for OSC commands (default 0)
	timeout           int                   // Timeout in seconds for OSC replies (default 10)
	cueFileDirectory  string                // Directory of the CUE file being processed (for resolving relative paths)
}

func NewWorkspace(host string, port int) Workspace {
	return Workspace{
		initialized:    false,
		host:           host,
		port:           port,
		client:         osc.NewClient(host, port),
		addressBuilder: messages.NewOSCAddressBuilder(""),
		cueNumbers:     make(map[string]string),
		cueListNames:   make(map[string]string),
		replyHandlers:  make(map[string]chan []any),
		timeout:        10,
	}
}

// NewTestWorkspace creates a workspace with a specific workspace ID for testing
func NewTestWorkspace(host string, port int, workspaceID string) *Workspace {
	w := &Workspace{
		initialized:    true, // Mark as initialized for testing
		host:           host,
		port:           port,
		client:         osc.NewClient(host, port),
		workspace_id:   workspaceID,
		addressBuilder: messages.NewOSCAddressBuilder(workspaceID),
		cueNumbers:     make(map[string]string),
		cueListNames:   make(map[string]string),
		replyHandlers:  make(map[string]chan []any),
	}

	// Start update listener to handle replies (with no-op update handler)
	if err := w.StartUpdateListener(func(address string, args []any) {
		// No-op for tests
	}); err != nil {
		// Log error but don't fail - tests may still work without update listener
		log.Warnf("Failed to start update listener: %v", err)
	}

	return w
}

// SetForceCueNumbers sets whether to force cue number conflicts by clearing existing numbers
func (q *Workspace) SetForceCueNumbers(force bool) {
	q.forceCueNumbers = force
}

// SetDryRun sets whether to run in dry-run mode (no actual changes)
func (q *Workspace) SetDryRun(dryRun bool) {
	q.dryRun = dryRun
}

// OnDisconnect sets a callback for when QLab appears to be disconnected
func (q *Workspace) OnDisconnect(callback func()) {
	q.onDisconnect = callback
}

// SetMaxRetries sets the maximum number of retry attempts for OSC commands
func (q *Workspace) SetMaxRetries(retries int) {
	q.maxRetries = retries
}

// SetTimeout sets the timeout in seconds for OSC replies
// For large workspaces with many cues, consider increasing this to 30-60 seconds
// Default is 10 seconds
func (q *Workspace) SetTimeout(seconds int) {
	q.timeout = seconds
	if seconds > 10 {
		log.Infof("OSC timeout increased to %d seconds for large workspace support", seconds)
	}
}

// SetCueFileHandler has been removed - file handling is now the caller's responsibility.
// Use TransmitWorkspaceData() and ReceiveWorkspaceData() instead.

// Cleanup closes the update server and cleans up resources
func (q *Workspace) Cleanup() {
	if q.updateServer != nil {
		if err := q.updateServer.CloseConnection(); err != nil {
			log.Warnf("Failed to close update server: %v", err)
		}
		q.updateServer = nil
	}
	// Reply servers are now self-managing and close themselves after receiving replies
}

func (q *Workspace) IsConnected() bool {
	return q.initialized && q.workspace_id != ""
}

// isWriteOperation determines if an OSC address represents a write operation
func (q *Workspace) isWriteOperation(address string) bool {
	// Write operations that should be blocked in dry-run mode - check these FIRST
	writeOps := []string{
		"/new",
		"/move",
		"/delete",
		"/cue_id/",     // Setting cue properties
		"/cueList_id/", // Setting cue list properties
	}

	for _, writeOp := range writeOps {
		if strings.Contains(address, writeOp) {
			return true
		}
	}

	// Read-only operations that should still work in dry-run mode
	readOnlyOps := []string{
		"/connect",
		"/alwaysReply",
		"/cueLists",
		"/cues",
		"/basePath",
	}

	for _, readOp := range readOnlyOps {
		if strings.Contains(address, readOp) {
			return false
		}
	}

	// Default to treating unknown operations as write operations for safety
	return true
}

// mockDryRunResponse returns realistic mock responses for different OSC operations
func (q *Workspace) mockDryRunResponse(address string, input string) []any {
	// Generate mock cue IDs for new cue creation
	if strings.Contains(address, "/new") {
		q.dryRunCounter++
		mockID := fmt.Sprintf("DRYRUN-%08X-%04X-4000-8000-000000000%03X", q.dryRunCounter, q.dryRunCounter, q.dryRunCounter)
		return []any{fmt.Sprintf(`{"status": "ok", "data": "%s", "workspace_id": "%s", "address": "%s"}`, mockID, q.workspace_id, address)}
	}

	// Mock success for property setting and moving operations
	if strings.Contains(address, "/cue_id/") || strings.Contains(address, "/move/") {
		return []any{fmt.Sprintf(`{"status": "ok", "workspace_id": "%s", "address": "%s"}`, q.workspace_id, address)}
	}

	// Default mock success response
	return []any{`{"status": "ok", "dry_run": true}`}
}

func (q *Workspace) Init(passcode string) ([]any, error) {
	log.Debugf("Init called with passcode: %q (length: %d)", passcode, len(passcode))
	connectAddr := q.addressBuilder.BuildAddress(messages.MsgConnect, nil)
	reply := q.Send(connectAddr, passcode)

	if len(reply) == 0 {
		return nil, fmt.Errorf("no reply received from QLab - is QLab running and accessible?")
	}

	var arg InitReplyArg
	arg_string, ok := reply[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid reply format from QLab")
	}

	logInfoJSON("Reply object", arg_string)
	err := json.Unmarshal([]byte(arg_string), &arg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection reply: %v", err)
	}

	log.Infof("Connection status: %s", arg.Status)

	// Check if the connection was successful
	if arg.Status == "error" {
		// Check if it's a timeout error
		if arg.Error == "timeout waiting for reply from QLab" {
			return reply, fmt.Errorf("connection timeout - is QLab running and accessible at %s:%d?", q.host, q.port)
		}
		return reply, fmt.Errorf("QLab connection failed - check passcode and workspace availability")
	}

	// Check for "badpass" in the data field (QLab returns this when passcode is incorrect)
	if arg.Data == "badpass" {
		return reply, fmt.Errorf("QLab authentication failed - incorrect passcode. Check your passcode in the CUE file, config file, or --passcode flag")
	}

	if arg.Status != "ok" {
		return reply, fmt.Errorf("unexpected connection status: %s", arg.Status)
	}

	q.workspace_id = arg.WorkspaceId
	q.addressBuilder = messages.NewOSCAddressBuilder(q.workspace_id)
	q.initialized = true
	log.Info("Successfully initialized workspace", "workspace_id", q.workspace_id)

	// Send /alwaysReply 1 to ensure cue messages don't time out
	alwaysReplyReply := q.Send("/alwaysReply", "1")
	if len(alwaysReplyReply) > 0 {
		if jsonStr, ok := alwaysReplyReply[0].(string); ok {
			logInfoJSON("alwaysReply response", jsonStr)
		} else {
			log.Info("alwaysReply response", "data", alwaysReplyReply[0])
		}
	}

	// Ensure "Cuejitsu Inbox" cue list exists for staging imported content
	q.inboxID, err = q.ensureCuejitsuInbox()
	if err != nil {
		log.Warnf("Failed to ensure Cuejitsu Inbox exists: %v", err)
		// Don't fail initialization if inbox creation fails
	}

	// Index existing cues for conflict detection
	err = q.indexExistingCues()
	if err != nil {
		log.Warnf("Failed to index existing cues: %v", err)
		// Don't fail initialization if cue indexing fails
	}

	return reply, nil
}

func (q *Workspace) GetAddress(msg string) string {
	if q.addressBuilder == nil {
		return ""
	}

	if !strings.HasPrefix(msg, "/") {
		return msg
	}

	if strings.HasPrefix(msg, "/workspace/") {
		return msg
	}

	applicationLevelCommands := []string{
		"/connect",
		"/disconnect",
		"/alwaysReply",
		"/version",
		"/updates",
		"/udpReplyPort",
		"/workspaces",
	}

	for _, cmd := range applicationLevelCommands {
		if strings.HasPrefix(msg, cmd) {
			return msg
		}
	}

	if q.workspace_id == "" {
		return msg
	}

	return fmt.Sprintf("/workspace/%s%s", q.workspace_id, msg)
}

func (q *Workspace) GetContent(msg string) string {
	reply := q.Send(q.GetAddress(msg), "")
	if len(reply) > 0 {
		if content, ok := reply[0].(string); ok {
			return content
		}
	}
	return ""
}

func (q *Workspace) GetRunningCues() []map[string]any {
	address := q.GetAddress("/runningCues/shallow")
	reply := q.Send(address, "")

	if len(reply) == 0 {
		return []map[string]any{}
	}

	var runningCues []map[string]any
	if jsonStr, ok := reply[0].(string); ok {
		var responseData map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &responseData); err != nil {
			return []map[string]any{}
		}
		if data, ok := responseData["data"].([]any); ok {
			for _, item := range data {
				if cueMap, ok := item.(map[string]any); ok {
					runningCues = append(runningCues, cueMap)
				}
			}
		}
	}
	return runningCues
}

func (q *Workspace) GetSelectedCues() []map[string]any {
	address := q.GetAddress("/selectedCues/shallow")
	reply := q.Send(address, "")

	if len(reply) == 0 {
		return []map[string]any{}
	}

	var selectedCues []map[string]any
	if jsonStr, ok := reply[0].(string); ok {
		var responseData map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &responseData); err != nil {
			return []map[string]any{}
		}
		if data, ok := responseData["data"].([]any); ok {
			for _, item := range data {
				if cueMap, ok := item.(map[string]any); ok {
					selectedCues = append(selectedCues, cueMap)
				}
			}
		}
	}
	return selectedCues
}

// TransmitWorkspaceData transmits workspace data to QLab with three-way comparison and conflict resolution.
// The caller is responsible for parsing the file and providing the workspace data.
// filePath is used for caching and logging purposes.
// Returns the comparison results which the caller can use to update source files if needed.
func (q *Workspace) TransmitWorkspaceData(filePath string, workspaceData map[string]any) (*ThreeWayComparison, error) {
	// Store the file directory for resolving relative file paths
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %v", err)
	}
	q.cueFileDirectory = filepath.Dir(absFilePath)
	log.Debug("Set cue file directory", "directory", q.cueFileDirectory)

	// Perform three-way comparison to detect changes
	log.Debug("Starting three-way comparison", "file", filePath)
	comparison, err := q.PerformThreeWayComparison(filePath, workspaceData)
	if err != nil {
		log.Debug("Change detection failed, proceeding without cache optimization", "error", err)
		// Fallback to old behavior if change detection fails
		err = q.transmitCueFileWithoutChangeDetection(workspaceData)
		return nil, err
	}

	// Initialize field-level tracking if not present
	if comparison.QLabChosenFields == nil {
		comparison.QLabChosenFields = make(map[string]map[string]bool)
	}

	// Print detailed results of the three-way comparison
	log.Debug("Printing three-way comparison results")
	log.Debug("Three-way comparison summary",
		"has_cache", comparison.HasCache,
		"has_qlab_data", comparison.HasQLabData,
		"cache_matches_qlab", comparison.CacheMatchesQLab)
	log.Debug("Three-way comparison results", "cue_result_count", len(comparison.CueResults))
	for cueNumber, result := range comparison.CueResults {
		log.Debug("Cue change detected",
			"cue_number", cueNumber,
			"action", result.Action,
			"has_changed", result.HasChanged,
			"reason", result.Reason)
	}
	q.PrintThreeWayComparisonResults(comparison)

	// Check for conflicts that need user resolution
	log.Debug("Identifying conflicts")
	conflicts, err := q.IdentifyConflicts(comparison)
	if err != nil {
		return nil, fmt.Errorf("failed to identify conflicts: %v", err)
	}
	log.Debug("Found", len(conflicts), "conflicts")

	// Prompt user for conflict resolution if needed
	if len(conflicts) > 0 {
		log.Debug("Prompting user for conflict resolution")
		err = q.PromptUserForConflictResolution(conflicts, comparison)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve conflicts: %v", err)
		}

		// Mark conflicts as resolved
		for i := range conflicts {
			conflicts[i].Resolved = true
		}
	}

	// Generate merged scope result if scope comparison was performed
	if comparison.WorkspaceScope != nil {
		log.Debug("Generating merged scope result")
		mergedScope, err := q.GenerateMergedScope(comparison.WorkspaceScope, comparison)
		if err != nil {
			log.Warnf("Failed to generate merged scope: %v", err)
		} else {
			comparison.MergedResult = mergedScope
			log.Infof("Merged result generated with %d top-level scopes", len(mergedScope.ChildScopes))
		}
	}

	// Process the workspace data with change detection
	log.Debug("Transmitting with change detection")
	err = q.transmitCueFileWithChangeDetection(workspaceData, comparison)
	if err != nil {
		return nil, fmt.Errorf("failed to transmit cue file with change detection: %v", err)
	}

	// Save cache after successful transmission
	log.Debug("Saving cache after successful transmission")
	err = q.writeCueFileToCache(filePath, workspaceData, nil, comparison)
	if err != nil {
		// Log warning but don't fail the transmission
		log.Debug("Warning: Failed to save cache", "error", err)
	} else {
		log.Debug("Cache saved successfully")
	}

	// Return comparison results so caller can update source file if needed
	// (for cues where user chose "Keep QLab version")
	return comparison, nil
}

// ReceiveWorkspaceData queries the current QLab workspace state and returns the cues data.
// The caller is responsible for writing this data to a file if needed.
func (q *Workspace) ReceiveWorkspaceData() ([]any, error) {
	currentWorkspace, err := q.queryCurrentWorkspaceState()
	if err != nil {
		return nil, fmt.Errorf("failed to query current workspace state: %v", err)
	}

	cuesData := q.extractCuesFromWorkspace(currentWorkspace)
	if len(cuesData) == 0 {
		log.Warn("No cues found in QLab workspace")
	}

	return cuesData, nil
}

func (q *Workspace) extractCuesFromWorkspace(workspace map[string]any) []any {
	var cuesData []any

	data, ok := workspace["data"].([]any)
	if !ok {
		return cuesData
	}

	for _, cueListData := range data {
		if cueList, ok := cueListData.(map[string]any); ok {
			if listCues, ok := cueList["cues"].([]any); ok {
				cuesData = append(cuesData, listCues...)
			}
		}
	}

	return cuesData
}

// transmitCueFileWithoutChangeDetection is the original transmission logic
func (q *Workspace) transmitCueFileWithoutChangeDetection(workspaceData map[string]any) error {
	// Process the workspace data to create cues
	// Look for cues under "cues" key or nested under "workspace" key
	var cuesData []any

	if cues, ok := workspaceData["cues"].([]any); ok {
		cuesData = cues
	} else if workspace, ok := workspaceData["workspace"].(map[string]any); ok {
		if cues, ok := workspace["cues"].([]any); ok {
			cuesData = cues
		}
	}

	if len(cuesData) == 0 {
		return fmt.Errorf("no cues found in CUE file")
	}

	// Process each cue
	for _, cueAny := range cuesData {
		cueData, ok := cueAny.(map[string]any)
		if !ok {
			continue // Skip invalid cue data
		}

		err := q.processCueList(cueData, "")
		if err != nil {
			return fmt.Errorf("failed to process cue: %v", err)
		}
	}

	return nil
}

// transmitCueFileWithChangeDetection processes cues using change detection results
func (q *Workspace) transmitCueFileWithChangeDetection(workspaceData map[string]any, comparison *ThreeWayComparison) error {
	// Process the workspace data to create cues
	// Look for cues under "cues" key or nested under "workspace" key
	var cuesData []any

	if cues, ok := workspaceData["cues"].([]any); ok {
		cuesData = cues
	} else if workspace, ok := workspaceData["workspace"].(map[string]any); ok {
		if cues, ok := workspace["cues"].([]any); ok {
			cuesData = cues
		}
	}

	if len(cuesData) == 0 {
		return fmt.Errorf("no cues found in CUE file")
	}

	// Create mapping for target resolution
	mapping := &CueMapping{
		NumberToID:      make(map[string]string),
		CuesWithTargets: []CueTarget{},
	}

	// Process each cue with change detection
	log.Debug("About to process cues from workspace data", "cue_count", len(cuesData))
	for i, cueAny := range cuesData {
		cueData, ok := cueAny.(map[string]any)
		if !ok {
			log.Debug("Skipping invalid cue data", "index", i)
			continue // Skip invalid cue data
		}

		log.Debug("Processing cue", "current", i+1, "total", len(cuesData))
		err := q.processCueListWithMappingAndChangeDetection(cueData, "", mapping, comparison.CueResults)
		if err != nil {
			log.Debug("ERROR - Failed to process cue", "index", i+1, "error", err)
			return fmt.Errorf("failed to process cue: %v", err)
		}
		log.Debug("Completed processing cue", "current", i+1, "total", len(cuesData))
	}

	// Set cue targets using the mapping
	err := q.setCueTargets(mapping)
	if err != nil {
		return fmt.Errorf("failed to set cue targets: %v", err)
	}

	return nil
}

// setCueListProperty sets a property on a cue list
func (q *Workspace) setCueListProperty(cueListID, property, value string) error {
	address := fmt.Sprintf("/workspace/%s/cue_id/%s/%s", q.workspace_id, cueListID, property)
	reply := q.Send(address, value)

	// Check for error in reply
	if len(reply) > 0 {
		replyStr, ok := reply[0].(string)
		if ok {
			var replyData map[string]any
			err := json.Unmarshal([]byte(replyStr), &replyData)
			if err == nil {
				if status, ok := replyData["status"].(string); ok && status == "error" {
					return formatErrorWithJSON("QLab error setting cue list property", replyStr)
				}
			}
		}
	}

	log.Debug("Set cue list property", "property", property, "value", value, "cue_list_id", cueListID)
	return nil
}

// indexCuesFromWorkspace creates a map of cue number -> cue data from workspace data
func (q *Workspace) indexCuesFromWorkspace(workspace map[string]any) map[string]map[string]any {
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
	q.indexCuesRecursively(cuesData, "", cueIndex)
	log.Debug("Final cue index complete", "entry_count", len(cueIndex))

	return cueIndex
}

// indexCuesRecursively recursively indexes cues, handling nested cue structures
func (q *Workspace) indexCuesRecursively(cuesData []any, parentNumber string, cueIndex map[string]map[string]any) {
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
			q.indexCuesRecursively(subCues, fullNumber, cueIndex)
		}
	}
}

// createCuejitsuInbox creates a new "Cuejitsu Inbox" cue list
func (q *Workspace) createCuejitsuInbox() (string, error) {
	// Create a new cue list using /new list
	address := fmt.Sprintf("/workspace/%s/new", q.workspace_id)
	reply := q.Send(address, "list")

	if len(reply) == 0 {
		return "", fmt.Errorf("no reply received when creating cue list")
	}

	replyStr, ok := reply[0].(string)
	if !ok {
		return "", fmt.Errorf("invalid reply format from cue list creation")
	}

	var replyData map[string]any
	err := json.Unmarshal([]byte(replyStr), &replyData)
	if err != nil {
		return "", fmt.Errorf("failed to parse cue list creation reply: %v", err)
	}

	// Check for error status
	if status, ok := replyData["status"].(string); ok && status == "error" {
		return "", formatErrorWithJSON("QLab error creating cue list", replyStr)
	}

	// Extract the new cue list ID
	cueListID, ok := replyData["data"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected cue list creation reply format")
	}

	log.Debug("Created new cue list", "cue_list_id", cueListID)

	// Set the name to "Cuejitsu Inbox"
	err = q.setCueListProperty(cueListID, "name", "Cuejitsu Inbox")
	if err != nil {
		return "", fmt.Errorf("failed to set cue list name: %v", err)
	}

	log.Debug("Set cue list name to 'Cuejitsu Inbox'")
	return cueListID, nil
}

// updateSourceFileWithQLabValues updates the source CUE file with QLab values for cues where user chose "Keep QLab version"
// ExtractQLabUpdates extracts cue field updates from QLab data for user-chosen cues.
// Returns a map of cue identifiers to field updates.
// The caller can use this to update source files.
func (q *Workspace) ExtractQLabUpdates(comparison *ThreeWayComparison) (map[string]map[string]any, error) {
	log.Debugf("ExtractQLabUpdates called: chosenCues=%+v", comparison.QLabChosenCues)

	if len(comparison.QLabChosenCues) == 0 || len(comparison.CurrentQLabData) == 0 {
		log.Debug("ExtractQLabUpdates: No chosen cues or QLab data, returning empty map")
		return make(map[string]map[string]any), nil
	}

	// Extract cue updates from QLab data
	cueUpdates := make(map[string]map[string]any)

	log.Debug("ExtractQLabUpdates: Extracting cue values from QLab data")
	err := q.extractQLabCueValues(comparison.CurrentQLabData, comparison.QLabChosenCues, cueUpdates)
	if err != nil {
		log.Errorf("ExtractQLabUpdates: Failed to extract QLab cue values: %v", err)
		return nil, fmt.Errorf("failed to extract QLab cue values: %v", err)
	}

	log.Debugf("ExtractQLabUpdates: Extracted %d cue updates", len(cueUpdates))
	return cueUpdates, nil
}

// extractQLabCueValues extracts cue field values from QLab workspace data
func (q *Workspace) extractQLabCueValues(qlabData map[string]any, chosenCues map[string]bool, cueUpdates map[string]map[string]any) error {
	log.Debug("extractQLabCueValues called", "chosenCues", chosenCues)

	// Navigate through QLab data structure to find cues
	if data, ok := qlabData["data"].([]any); ok {
		for _, cueListData := range data {
			if cueList, ok := cueListData.(map[string]any); ok {
				if cues, ok := cueList["cues"].([]any); ok {
					q.extractCueValuesFromArrayWithPosition(cues, "", chosenCues, cueUpdates)
				}
			}
		}
	}

	log.Debug("extractQLabCueValues completed", "totalUpdates", len(cueUpdates))
	for id, updates := range cueUpdates {
		log.Debug("Extracted cue update", "identifier", id, "updates", updates)
	}

	return nil
}

// extractCueValuesFromArrayWithPosition recursively extracts cue values from QLab cue arrays with position tracking
func (q *Workspace) extractCueValuesFromArrayWithPosition(cues []any, parentNumber string, chosenCues map[string]bool, cueUpdates map[string]map[string]any) {
	for i, cueItem := range cues {
		if cueMap, ok := cueItem.(map[string]any); ok {
			// Get cue identifier with position context (same logic as indexCuesRecursively)
			cueNumber := q.getQLabCueIdentifierWithPosition(cueMap, parentNumber, i)

			// Debug: log all generated identifiers during extraction
			cueName, _ := cueMap["name"].(string)
			cueType, _ := cueMap["type"].(string)
			log.Debugf("Generated QLab identifier: '%s' (parent=%s, pos=%d, type=%s, name=%s)", cueNumber, parentNumber, i, cueType, cueName)

			// If this cue was chosen to keep QLab version, extract its values
			if cueNumber != "" && chosenCues[cueNumber] {
				updates := make(map[string]any)

				// Extract common updatable fields from QLab
				if name, ok := cueMap["name"].(string); ok {
					updates["name"] = name
				}
				if fileTarget, ok := cueMap["fileTarget"].(string); ok {
					updates["fileTarget"] = fileTarget
				}
				if notes, ok := cueMap["notes"].(string); ok {
					updates["notes"] = notes
				}
				if colorName, ok := cueMap["colorName"].(string); ok {
					updates["colorName"] = colorName
				}

				if len(updates) > 0 {
					cueUpdates[cueNumber] = updates
					log.Debugf("Extracted %d field updates for cue %s from QLab", len(updates), cueNumber)
				}
			}

			// Get the full number for this cue to pass to children (same logic as indexCuesRecursively)
			var currentFullNumber string
			if num, ok := cueMap["number"]; ok && num != nil {
				switch v := num.(type) {
				case string:
					currentFullNumber = v
				case float64:
					if v == float64(int64(v)) && v >= 0 && v <= 999 {
						currentFullNumber = fmt.Sprintf("%.1f", v)
					} else {
						currentFullNumber = fmt.Sprintf("%g", v)
					}
				case int64:
					currentFullNumber = fmt.Sprintf("%d", v)
				case int:
					currentFullNumber = fmt.Sprintf("%d", v)
				default:
					currentFullNumber = fmt.Sprintf("%v", v)
				}

				// Build full cue number with parent prefix
				if parentNumber != "" && currentFullNumber != "" {
					if !strings.Contains(currentFullNumber, ".") {
						currentFullNumber = parentNumber + "." + currentFullNumber
					}
				}
			}

			// Recursively check children (QLab uses "cues" for nested cues)
			if children, ok := cueMap["cues"].([]any); ok {
				q.extractCueValuesFromArrayWithPosition(children, currentFullNumber, chosenCues, cueUpdates)
			}
		}
	}
}

// getQLabCueIdentifier extracts cue identifier from QLab cue data (simple version for backwards compatibility)
func (q *Workspace) getQLabCueIdentifier(cue map[string]any) string {
	return q.getQLabCueIdentifierWithPosition(cue, "", 0)
}

// getQLabCueIdentifierWithPosition extracts cue identifier from QLab cue data with position context
// Uses the same logic as indexCuesRecursively to ensure consistent identifiers
func (q *Workspace) getQLabCueIdentifierWithPosition(cue map[string]any, parentNumber string, position int) string {
	// Extract cue number (same logic as indexCuesRecursively)
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

	// Build full cue number with parent prefix (same logic as indexCuesRecursively)
	fullNumber := cueNumber
	if parentNumber != "" && cueNumber != "" {
		if strings.Contains(cueNumber, ".") {
			fullNumber = cueNumber
		} else {
			fullNumber = parentNumber + "." + cueNumber
		}
	}

	cueName, _ := cue["name"].(string)
	cueType, _ := cue["type"].(string)
	log.Debug("getQLabCueIdentifierWithPosition called", "parentNumber", parentNumber, "position", position, "cueNumber", cueNumber, "fullNumber", fullNumber, "cueName", cueName, "cueType", cueType)

	// Return numbered identifier if we have one
	if fullNumber != "" {
		log.Debug("Returning numbered identifier", "identifier", fullNumber)
		return fullNumber
	}

	// Fallback: use position-based identification for cues without numbers (same logic as indexCuesRecursively)
	// Create composite key: parent@position[type:name]
	// Normalize type to lowercase for consistent matching
	normalizedType := strings.ToLower(cueType)
	var positionKey string
	if parentNumber != "" {
		positionKey = fmt.Sprintf("%s@%d[%s:%s]", parentNumber, position, normalizedType, cueName)
	} else {
		positionKey = fmt.Sprintf("@%d[%s:%s]", position, normalizedType, cueName)
	}

	// Only return if we have enough identifying information
	if cueType != "" || cueName != "" {
		log.Debug("Returning position-based identifier", "identifier", positionKey)
		return positionKey
	}

	log.Debug("No identifier found - returning empty string")
	return ""
}

// applyCueUpdatesToSourceFile has been removed - file writing is now handled by the caller.
// Use ExtractQLabUpdates() to get the updates, then write them to your file format.

// getVideoStages queries QLab for available video stages (cached)
func (q *Workspace) getVideoStages() ([]map[string]any, error) {
	// Return cached result if available
	if q.videoStagesCache != nil {
		log.Debugf("Returning cached video stages (%d stages)", len(q.videoStagesCache))
		return q.videoStagesCache, nil
	}

	if q.workspace_id == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	log.Debugf("Querying QLab for video stages")
	address := fmt.Sprintf("/workspace/%s/settings/video/stages", q.workspace_id)
	reply := q.Send(address, "")

	if len(reply) == 0 {
		return nil, fmt.Errorf("no reply received from QLab")
	}

	replyStr, ok := reply[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid reply format")
	}

	var replyData map[string]any
	if err := json.Unmarshal([]byte(replyStr), &replyData); err != nil {
		return nil, fmt.Errorf("failed to parse stages reply: %v", err)
	}

	if status, ok := replyData["status"].(string); ok && status == "error" {
		return nil, fmt.Errorf("QLab returned error getting stages")
	}

	data, ok := replyData["data"].([]any)
	if !ok {
		return nil, fmt.Errorf("no stages data in reply")
	}

	stages := make([]map[string]any, 0, len(data))
	for _, stageAny := range data {
		if stage, ok := stageAny.(map[string]any); ok {
			stages = append(stages, stage)
		}
	}

	// Cache the result
	q.videoStagesCache = stages
	log.Debugf("Cached %d video stages", len(stages))

	return stages, nil
}

// Close cleans up resources used by the workspace
func (q *Workspace) Close() {
	q.serverMux.Lock()
	defer q.serverMux.Unlock()

	// Close update server if it exists
	// The go-osc library has some race conditions, but we need to close
	// the connections to free up the ports for subsequent tests.
	// We use a small delay to let the server finish initializing first.
	if q.updateServer != nil {
		server := q.updateServer
		q.updateServer = nil

		// Close in background to avoid blocking
		go func() {
			time.Sleep(100 * time.Millisecond)
			log.Debugf("Closing update server")
			if err := server.CloseConnection(); err != nil {
				log.Warnf("Failed to close update server: %v", err)
			}
		}()
	}

	// Close reply server if it exists
	if q.replyServer != nil {
		server := q.replyServer
		q.replyServer = nil

		// Close in background to avoid blocking
		go func() {
			time.Sleep(100 * time.Millisecond)
			log.Debugf("Closing reply server")
			if err := server.CloseConnection(); err != nil {
				log.Warnf("Failed to close reply server: %v", err)
			}
		}()
	}

	// Clear ready channels
	q.updateServerReady = nil
	q.replyServerReady = nil

	// Don't close reply handler channels as they may still be in use
	// Just clear the map
	q.replyHandlersMux.Lock()
	q.replyHandlers = make(map[string]chan []any)
	q.replyHandlersMux.Unlock()
}
