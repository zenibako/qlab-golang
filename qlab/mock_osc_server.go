package qlab

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/hypebeast/go-osc/osc"
)

// ReceivedMessage captures details about received OSC messages for testing
type ReceivedMessage struct {
	Address   string
	Arguments []any
	Timestamp time.Time
}

// safeDispatcher wraps an OSC dispatcher with thread-safe dispatch
type safeDispatcher struct {
	dispatcher osc.Dispatcher
	mu         *sync.RWMutex
}

func (s *safeDispatcher) Dispatch(packet osc.Packet) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.dispatcher.Dispatch(packet)
}

// MockOSCServer simulates QLab OSC server for testing
type MockOSCServer struct {
	host              string
	port              int
	replyPort         int
	server            *osc.Server
	workspaceID       string
	cues              map[string]*MockCue     // uniqueID -> cue
	cueLists          map[string]*MockCueList // uniqueID -> cue list
	cuesByNumber      map[string]string       // number -> uniqueID
	nextCueNumber     int
	nextCueListNumber int
	mu                sync.RWMutex
	dispatcherMu      sync.RWMutex
	isRunning         bool
	alwaysReply       bool
	dispatcher        *osc.StandardDispatcher // Keep reference for dynamic handler registration
	serverReady       chan struct{}           // Signal that server is ready
	receivedMessages  []ReceivedMessage       // Capture all received messages for testing
	registeredCues    map[string]bool         // Track which cues have handlers registered
	registeredLists   map[string]bool         // Track which lists have handlers registered
}

// MockCue represents a cue in the mock QLab workspace
type MockCue struct {
	UniqueID        string            `json:"uniqueID"`
	Type            string            `json:"type"`
	Name            string            `json:"name,omitempty"`
	Number          string            `json:"number,omitempty"`
	FileTarget      string            `json:"fileTarget,omitempty"`
	InfiniteLoop    bool              `json:"infiniteLoop,omitempty"`
	Mode            int               `json:"mode,omitempty"`
	CueTargetNumber string            `json:"cueTargetNumber,omitempty"`
	CueTargetID     string            `json:"cueTargetID,omitempty"`
	Children        []string          `json:"-"` // uniqueIDs of child cues
	Properties      map[string]string `json:"-"` // additional properties
}

// MockCueList represents a cue list in the mock QLab workspace
type MockCueList struct {
	UniqueID   string            `json:"uniqueID"`
	Name       string            `json:"name,omitempty"`
	Type       string            `json:"type"`
	Properties map[string]string `json:"-"` // additional properties
}

// NewMockOSCServer creates a new mock QLab OSC server
func NewMockOSCServer(host string, port int) *MockOSCServer {
	return &MockOSCServer{
		host:              host,
		port:              port,
		replyPort:         port + 1, // Match the reply port calculation in main code
		workspaceID:       "MOCK-WORKSPACE-ID-1234",
		cues:              make(map[string]*MockCue),
		cueLists:          make(map[string]*MockCueList),
		cuesByNumber:      make(map[string]string),
		nextCueNumber:     1,
		nextCueListNumber: 1,
		alwaysReply:       false,
		receivedMessages:  make([]ReceivedMessage, 0),
		registeredCues:    make(map[string]bool),
		registeredLists:   make(map[string]bool),
	}
}

// Start starts the mock OSC server
func (m *MockOSCServer) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("mock server already running")
	}

	// Setup main server dispatcher
	d := osc.NewStandardDispatcher()
	m.dispatcher = d // Store reference for dynamic handler registration

	// Handle connect messages
	_ = d.AddMsgHandler("/connect", m.handleConnect)

	// Handle alwaysReply messages
	_ = d.AddMsgHandler("/alwaysReply", m.handleAlwaysReply)

	// Handle global working directory
	_ = d.AddMsgHandler("/workingDirectory", m.handleGetWorkingDirectory)

	// Handle workspace messages with specific workspace ID
	workspacePrefix := fmt.Sprintf("/workspace/%s", m.workspaceID)
	_ = d.AddMsgHandler(workspacePrefix+"/new", m.handleNewCue)
	// Individual cue handlers will be registered dynamically when cues are created
	_ = d.AddMsgHandler(workspacePrefix+"/cueLists", m.handleGetCueLists)
	// Note: /cueLists/uniqueIDs is intentionally not registered as it conflicts with /cueLists matching
	_ = d.AddMsgHandler(workspacePrefix+"/basePath", m.handleGetWorkspaceBasePath)
	_ = d.AddMsgHandler(workspacePrefix+"/cue/*/children", m.handleGetChildrenByNumber)
	_ = d.AddMsgHandler(workspacePrefix+"/cue/selected/children", m.handleGetSelectedChildren)
	_ = d.AddMsgHandler(workspacePrefix+"/cue_id/*/children", m.handleGetChildrenByID)
	_ = d.AddMsgHandler("/cue/*/children", m.handleGetChildrenByNumber)
	_ = d.AddMsgHandler("/cue/selected/children", m.handleGetSelectedChildren)
	_ = d.AddMsgHandler("/cue_id/*/children", m.handleGetChildrenByID)

	// Dynamic handlers registered per cue - no catchall needed

	// Wrap dispatcher to be thread-safe
	wrappedDispatcher := &safeDispatcher{
		dispatcher: d,
		mu:         &m.dispatcherMu,
	}

	// Start main server
	m.server = &osc.Server{
		Addr:       fmt.Sprintf("%s:%d", m.host, m.port),
		Dispatcher: wrappedDispatcher,
	}

	// Start main server only - no need for separate reply server
	// The mock server will send replies directly to the workspace's reply server
	m.serverReady = make(chan struct{})
	ready := m.serverReady
	go func() {
		if err := m.server.ListenAndServe(); err != nil {
			log.Errorf("Mock OSC server error: %v", err)
		}
	}()

	// Give servers time to start
	time.Sleep(100 * time.Millisecond)

	m.isRunning = true
	close(ready)
	log.Infof("Mock QLab OSC server started on %s:%d (reply: %d)", m.host, m.port, m.replyPort)
	return nil
}

// Stop stops the mock OSC server
func (m *MockOSCServer) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return nil
	}

	// Close the server if it exists
	// The go-osc library has some race conditions, but we need to close
	// the connection to free up the port for subsequent tests.
	// We use a small delay to let the server finish initializing first.
	if m.server != nil {
		server := m.server
		m.server = nil

		// Close in background to avoid blocking
		go func() {
			time.Sleep(100 * time.Millisecond)
			log.Debugf("Closing mock OSC server")
			if err := server.CloseConnection(); err != nil {
				log.Warnf("Failed to close mock server: %v", err)
			}
		}()
	}

	// Clear ready channel
	m.serverReady = nil

	m.isRunning = false
	log.Info("Mock QLab OSC server stopped")
	return nil
}

// GetWorkspaceID returns the mock workspace ID
func (m *MockOSCServer) GetWorkspaceID() string {
	return m.workspaceID
}

// sendReply sends a reply message to the workspace reply server
func (m *MockOSCServer) sendReply(address string, data any) {
	// Build reply address by prepending /reply to the original address
	replyAddress := "/reply" + address

	log.Infof("Mock server preparing reply to %s", replyAddress)

	// For QLab compatibility, we need to send the data directly as OSC arguments
	// rather than as JSON. QLab expects simple values, not complex JSON objects.
	msg := osc.NewMessage(replyAddress)

	switch v := data.(type) {
	case string:
		msg.Append(v)
		log.Debugf("Mock server sending string reply: %s", v)
	case map[string]any:
		// For error responses or complex data, send as JSON
		jsonData, err := json.Marshal(v)
		if err != nil {
			log.Errorf("Failed to marshal reply data: %v", err)
			return
		}
		msg.Append(string(jsonData))
		log.Debugf("Mock server sending JSON reply: %s", string(jsonData))
	case []any:
		// For array data (like children list), send as JSON
		jsonData, err := json.Marshal(v)
		if err != nil {
			log.Errorf("Failed to marshal reply data: %v", err)
			return
		}
		msg.Append(string(jsonData))
		log.Debugf("Mock server sending JSON array reply: %s", string(jsonData))
	default:
		// Convert other types to string
		strValue := fmt.Sprintf("%v", v)
		msg.Append(strValue)
		log.Debugf("Mock server sending converted reply: %s", strValue)
	}

	// Send reply to the workspace reply port
	client := osc.NewClient(m.host, m.replyPort)

	// Small delay to simulate QLab processing time and allow reply server to start
	time.Sleep(150 * time.Millisecond)

	log.Infof("Mock server sending reply to %s:%d with address %s", m.host, m.replyPort, replyAddress)
	if err := client.Send(msg); err != nil {
		log.Errorf("Failed to send mock reply: %v", err)
	} else {
		log.Infof("Mock server successfully sent reply")
	}
}

// handleConnect handles connection requests
func (m *MockOSCServer) handleConnect(msg *osc.Message) {
	log.Debug("Mock server received connect request")

	// Check passcode (simulate authentication)
	var passcode string
	if len(msg.Arguments) > 0 {
		if pc, ok := msg.Arguments[0].(string); ok {
			passcode = pc
		}
	}

	// Simulate authentication failure for "test" passcode (like a real QLab with wrong passcode)
	if passcode == "test" {
		replyData := map[string]any{
			"address":      fmt.Sprintf("/workspace/%s/connect", m.workspaceID),
			"status":       "ok",
			"data":         "badpass",
			"workspace_id": m.workspaceID,
		}
		m.sendReply("/connect", replyData)
		return
	}

	// Simulate successful connection for any other passcode
	replyData := map[string]any{
		"address":      fmt.Sprintf("/workspace/%s/connect", m.workspaceID),
		"status":       "ok",
		"data":         "ok:view|edit|control",
		"workspace_id": m.workspaceID,
	}

	m.sendReply("/connect", replyData)
}

// handleAlwaysReply handles alwaysReply setting
func (m *MockOSCServer) handleAlwaysReply(msg *osc.Message) {
	log.Debug("Mock server received alwaysReply request")

	m.mu.Lock()
	m.alwaysReply = true
	m.mu.Unlock()

	replyData := map[string]any{
		"address": "/alwaysReply",
		"status":  "ok",
	}

	m.sendReply("/alwaysReply", replyData)
}

// handleNewCue handles cue and cue list creation
func (m *MockOSCServer) handleNewCue(msg *osc.Message) {
	log.Debug("Mock server received new request:", msg.String())

	if len(msg.Arguments) == 0 {
		m.sendErrorReply(msg.Address, "no cue type specified")
		return
	}

	cueType, ok := msg.Arguments[0].(string)
	if !ok {
		m.sendErrorReply(msg.Address, "invalid cue type")
		return
	}

	m.mu.Lock()

	// Check if this is a cue list creation request
	if cueType == "list" || cueType == "cuelist" || cueType == "cue list" {
		// Generate unique ID for cue list
		uniqueID := fmt.Sprintf("MOCK-CUELIST-%d", m.nextCueListNumber)
		m.nextCueListNumber++

		// Create the new cue list
		cueList := &MockCueList{
			UniqueID:   uniqueID,
			Name:       "Main Cue List", // Default name, can be changed later
			Type:       "cue_list",
			Properties: make(map[string]string),
		}

		// Store the cue list
		m.cueLists[uniqueID] = cueList

		log.Infof("Mock server created cue list: %s (type: %s)", uniqueID, cueList.Type)

		// Prepare reply data before unlocking
		replyData := map[string]any{
			"status": "ok",
			"data":   uniqueID,
		}
		replyAddress := msg.Address

		// Release the lock before doing any I/O or handler registration
		m.mu.Unlock()

		// Register handlers asynchronously to avoid blocking the dispatcher
		go m.registerCueListHandlers(uniqueID)

		// Send reply immediately
		m.sendReply(replyAddress, replyData)
		return
	}

	// Generate unique ID for regular cue
	uniqueID := fmt.Sprintf("MOCK-CUE-%d", len(m.cues)+1)

	// Create new cue
	cue := &MockCue{
		UniqueID:   uniqueID,
		Type:       cueType,
		Properties: make(map[string]string),
		Children:   make([]string, 0),
	}

	m.cues[uniqueID] = cue

	log.Infof("Mock server created cue: %s (type: %s)", uniqueID, cueType)

	// Prepare reply data before unlocking
	replyData := map[string]any{
		"status": "ok",
		"data":   uniqueID,
	}
	replyAddress := msg.Address

	// Release the lock before doing any I/O or handler registration
	m.mu.Unlock()

	// Register handlers asynchronously to avoid blocking the dispatcher
	go m.registerCueHandlers(uniqueID)

	// Send reply immediately
	m.sendReply(replyAddress, replyData)
}

// handleSetCueProperty handles setting cue properties
func (m *MockOSCServer) handleSetCueProperty(msg *osc.Message) {
	log.Debug("Mock server received set property request:", msg.String())

	// Capture the message for testing verification
	m.captureMessage(msg)

	// Extract cue ID and property from address
	addressParts := strings.Split(msg.Address, "/")
	var cueID, property string

	for i, part := range addressParts {
		if part == "cue_id" && i+1 < len(addressParts) {
			cueID = addressParts[i+1]
			if i+2 < len(addressParts) {
				property = addressParts[i+2]
			}
			break
		}
	}

	if cueID == "" || property == "" {
		m.sendErrorReply(msg.Address, "invalid property address")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cue, exists := m.cues[cueID]
	if !exists {
		m.sendErrorReply(msg.Address, fmt.Sprintf("cue %s not found", cueID))
		return
	}

	// If no arguments, this is a query - return the property value
	if len(msg.Arguments) == 0 {
		var data any
		switch property {
		case "name":
			data = cue.Name
		case "number":
			data = cue.Number
		case "fileTarget":
			if cue.FileTarget != "" {
				basePath := "/Users/test/Desktop/QLab Workspace"
				if strings.HasPrefix(cue.FileTarget, basePath+"/") {
					data = strings.TrimPrefix(cue.FileTarget, basePath+"/")
				} else {
					data = cue.FileTarget
				}
			} else {
				data = cue.FileTarget
			}
		case "file":
			data = cue.FileTarget
		case "infiniteLoop":
			if cue.InfiniteLoop {
				data = "1"
			} else {
				data = "0"
			}
		case "mode":
			data = fmt.Sprintf("%d", cue.Mode)
		case "cueTarget", "cueTargetID":
			data = cue.CueTargetID
		case "cueTargetNumber":
			data = cue.CueTargetNumber
		default:
			if val, ok := cue.Properties[property]; ok {
				data = val
			} else {
				data = ""
			}
		}

		log.Debugf("Mock server query %s.%s = %v", cueID, property, data)
		replyData := map[string]any{
			"status": "ok",
			"data":   data,
		}
		m.sendReply(msg.Address, replyData)
		return
	}

	// Set property based on value
	value := fmt.Sprintf("%v", msg.Arguments[0])

	switch property {
	case "name":
		cue.Name = value
	case "number":
		// Remove old number mapping if it exists
		if cue.Number != "" {
			delete(m.cuesByNumber, cue.Number)
		}
		cue.Number = value
		// Only add to mapping if the new value is not empty
		if value != "" {
			m.cuesByNumber[value] = cueID
		}
	case "fileTarget", "file":
		cue.FileTarget = value
	case "infiniteLoop":
		cue.InfiniteLoop = value == "1" || value == "true"
	case "mode":
		if modeInt, err := strconv.Atoi(value); err == nil {
			cue.Mode = modeInt
		}
	case "cueTarget":
		cue.CueTargetID = value
	case "cueTargetNumber":
		cue.CueTargetNumber = value
	case "cueTargetID":
		cue.CueTargetID = value
	default:
		cue.Properties[property] = value
	}

	log.Debugf("Mock server set %s.%s = %s", cueID, property, value)

	// Send reply in the format expected by the workspace
	replyData := map[string]any{
		"status": "ok",
	}
	m.sendReply(msg.Address, replyData)
}

// handleMoveCue handles moving cues
func (m *MockOSCServer) handleMoveCue(msg *osc.Message) {
	log.Debug("Mock server received move cue request:", msg.String())

	// Extract cue ID from address
	addressParts := strings.Split(msg.Address, "/")
	var cueID string

	for i, part := range addressParts {
		if part == "move" && i+1 < len(addressParts) {
			cueID = addressParts[i+1]
			break
		}
	}

	if cueID == "" {
		m.sendErrorReply(msg.Address, "invalid move address")
		return
	}

	// Check arguments - should be index and parent cue ID
	if len(msg.Arguments) != 2 {
		log.Debugf("Mock server received %d arguments for move, expected 2", len(msg.Arguments))
		m.sendErrorReply(msg.Address, fmt.Sprintf("expected 2 arguments for move, got %d", len(msg.Arguments)))
		return
	}

	index, indexOk := msg.Arguments[0].(int32)
	parentID, parentOk := msg.Arguments[1].(string)

	if !indexOk || !parentOk {
		log.Debugf("Mock server received invalid argument types for move: %T, %T", msg.Arguments[0], msg.Arguments[1])
		m.sendErrorReply(msg.Address, "invalid argument types for move")
		return
	}

	log.Debugf("Mock server acknowledging move of cue %s to index %d under parent %s", cueID, index, parentID)
	replyData := map[string]any{"status": "ok"}
	m.sendReply(msg.Address, replyData)
}

// handleDeleteCue handles deleting cues
func (m *MockOSCServer) handleDeleteCue(msg *osc.Message) {
	log.Debug("Mock server received delete cue request:", msg.String())

	// Extract cue ID from address
	addressParts := strings.Split(msg.Address, "/")
	var cueID string

	for i, part := range addressParts {
		if part == "delete_id" && i+1 < len(addressParts) {
			cueID = addressParts[i+1]
			break
		}
	}

	if cueID == "" {
		m.sendErrorReply(msg.Address, "invalid delete address")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cue, exists := m.cues[cueID]
	if !exists {
		m.sendErrorReply(msg.Address, fmt.Sprintf("cue %s not found", cueID))
		return
	}

	// Remove from number mapping
	if cue.Number != "" {
		delete(m.cuesByNumber, cue.Number)
	}

	// Remove cue
	delete(m.cues, cueID)

	log.Debugf("Mock server deleted cue %s", cueID)
	replyData := map[string]any{"status": "ok"}
	m.sendReply(msg.Address, replyData)
}

// handleGetChildrenByNumber handles getting children by cue number
func (m *MockOSCServer) handleGetChildrenByNumber(msg *osc.Message) {
	log.Debug("Mock server received get children by number request:", msg.String())

	// For mock, return empty children list
	children := make([]any, 0)
	m.sendReply(msg.Address, children)
}

// handleGetSelectedChildren handles getting selected cue children
func (m *MockOSCServer) handleGetSelectedChildren(msg *osc.Message) {
	log.Debug("Mock server received get selected children request:", msg.String())

	// For mock, return empty children list
	children := make([]any, 0)
	m.sendReply(msg.Address, children)
}

// handleGetChildrenByID handles getting children by cue ID
func (m *MockOSCServer) handleGetChildrenByID(msg *osc.Message) {
	log.Debug("Mock server received get children by ID request:", msg.String())

	// For mock, return empty children list
	children := make([]any, 0)
	m.sendReply(msg.Address, children)
}

// handleGetCueLists handles getting full cue lists structure
func (m *MockOSCServer) handleGetCueLists(msg *osc.Message) {
	log.Debug("Mock server received cueLists request")

	m.mu.Lock()
	defer m.mu.Unlock()

	// Create response containing all cue lists
	var cueLists []any

	// Add the default main cue list with all cues
	mainCueList := map[string]any{
		"uniqueID": "main-cue-list",
		"name":     "Main Cue List",
		"type":     "cue_list",
		"cues":     make([]any, 0),
	}

	// Add all cues to the main cue list
	var cues []any
	for _, cue := range m.cues {
		cueData := map[string]any{
			"uniqueID": cue.UniqueID,
			"type":     cue.Type,
		}

		// Add properties if they exist
		if cue.Name != "" {
			cueData["name"] = cue.Name
		}
		if cue.Number != "" {
			cueData["number"] = cue.Number
		}
		// Per QLab OSC docs, /cueLists only returns: uniqueID, number, name, listName, type,
		// colorName, flagged, armed. Properties like fileTarget and cueTargetNumber must be
		// queried separately via /cue_id/{id}/{property}

		// Add any additional properties
		for key, value := range cue.Properties {
			cueData[key] = value
		}

		cues = append(cues, cueData)
	}

	mainCueList["cues"] = cues
	cueLists = append(cueLists, mainCueList)

	// Add any additional cue lists that were created
	for _, cueList := range m.cueLists {
		cueListData := map[string]any{
			"uniqueID": cueList.UniqueID,
			"name":     cueList.Name,
			"type":     cueList.Type,
			"cues":     make([]any, 0), // Empty for now - could be populated later if needed
		}
		cueLists = append(cueLists, cueListData)
	}

	// Return as array of cue lists (QLab can have multiple cue lists)

	replyData := map[string]any{
		"status": "ok",
		"data":   cueLists,
	}

	m.sendReply(msg.Address, replyData)
}

// handleGetWorkspaceBasePath handles getting the workspace base path
func (m *MockOSCServer) handleGetWorkspaceBasePath(msg *osc.Message) {
	log.Debug("Mock server received workspace basePath request:", msg.String())

	// Return a mock base path for testing
	replyData := map[string]any{
		"status": "ok",
		"data":   "/Users/test/Desktop/QLab Workspace",
	}

	m.sendReply(msg.Address, replyData)
}

// handleGetWorkingDirectory handles getting the global working directory
func (m *MockOSCServer) handleGetWorkingDirectory(msg *osc.Message) {
	log.Debug("Mock server received /workingDirectory request:", msg.String())

	// Return a mock working directory for testing
	replyData := map[string]any{
		"status": "ok",
		"data":   "/Users/test/Desktop",
	}

	m.sendReply(msg.Address, replyData)
}

// sendErrorReply sends an error reply
func (m *MockOSCServer) sendErrorReply(address, errorMsg string) {
	// For compatibility with QLab error format, send error as JSON
	replyData := map[string]any{
		"status": "error",
		"error":  errorMsg,
	}

	m.sendReply(address, replyData)
}

// GetCueCount returns the number of cues created
func (m *MockOSCServer) GetCueCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.cues)
}

// GetCue returns a cue by its unique ID
func (m *MockOSCServer) GetCue(uniqueID string) *MockCue {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cues[uniqueID]
}

// Clear removes all cues
func (m *MockOSCServer) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cues = make(map[string]*MockCue)
	m.cuesByNumber = make(map[string]string)
	m.nextCueNumber = 1

	log.Debug("Mock server cleared all cues")
}

// registerCueHandlers dynamically registers handlers for a specific cue
func (m *MockOSCServer) registerCueHandlers(cueID string) {
	m.mu.RLock()
	if m.registeredCues[cueID] {
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()

	m.mu.Lock()
	m.registeredCues[cueID] = true
	m.mu.Unlock()

	workspacePrefix := fmt.Sprintf("/workspace/%s", m.workspaceID)

	m.dispatcherMu.Lock()
	defer m.dispatcherMu.Unlock()

	// Register handlers for all supported properties for this specific cue
	properties := []string{"name", "number", "fileTarget", "file", "infiniteLoop", "mode", "cueTarget", "cueTargetNumber", "cueTargetID"}
	for _, prop := range properties {
		address := fmt.Sprintf("%s/cue_id/%s/%s", workspacePrefix, cueID, prop)
		_ = m.dispatcher.AddMsgHandler(address, m.handleSetCueProperty)
	}

	// Register move and delete handlers for this cue
	_ = m.dispatcher.AddMsgHandler(fmt.Sprintf("%s/move/%s", workspacePrefix, cueID), m.handleMoveCue)
	_ = m.dispatcher.AddMsgHandler(fmt.Sprintf("%s/delete_id/%s", workspacePrefix, cueID), m.handleDeleteCue)
}

// registerCueListHandlers registers OSC handlers for a specific cue list
func (m *MockOSCServer) registerCueListHandlers(cueListID string) {
	m.mu.RLock()
	if m.registeredLists[cueListID] {
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()

	m.mu.Lock()
	m.registeredLists[cueListID] = true
	m.mu.Unlock()

	workspacePrefix := fmt.Sprintf("/workspace/%s", m.workspaceID)

	m.dispatcherMu.Lock()
	defer m.dispatcherMu.Unlock()

	// Register handlers for cue list properties
	properties := []string{"name"}
	for _, prop := range properties {
		address := fmt.Sprintf("%s/cue_id/%s/%s", workspacePrefix, cueListID, prop)
		_ = m.dispatcher.AddMsgHandler(address, m.handleSetCueListProperty)
	}
}

// handleSetCueListProperty handles setting properties on cue lists
func (m *MockOSCServer) handleSetCueListProperty(msg *osc.Message) {
	// Parse the message address to extract cue list ID and property
	// Format: /workspace/{workspaceID}/cue_id/{cueListID}/{property}
	parts := strings.Split(msg.Address, "/")
	if len(parts) < 5 {
		m.sendErrorReply(msg.Address, "invalid cue list property address format")
		return
	}

	cueListID := parts[4]
	property := parts[5]

	var value string
	if len(msg.Arguments) == 0 {
		// Allow empty arguments for property clearing
		value = ""
	} else {
		value = fmt.Sprintf("%v", msg.Arguments[0])
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cueList, exists := m.cueLists[cueListID]
	if !exists {
		m.sendErrorReply(msg.Address, fmt.Sprintf("cue list %s not found", cueListID))
		return
	}

	// Set property based on type
	switch property {
	case "name":
		cueList.Name = value
	default:
		cueList.Properties[property] = value
	}

	log.Debugf("Mock server set %s.%s = %s", cueListID, property, value)

	// Send success reply
	replyData := map[string]any{"status": "ok"}
	m.sendReply(msg.Address, replyData)
}

// captureMessage records a received message for testing verification
func (m *MockOSCServer) captureMessage(msg *osc.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.receivedMessages = append(m.receivedMessages, ReceivedMessage{
		Address:   msg.Address,
		Arguments: append([]any{}, msg.Arguments...), // Deep copy arguments
		Timestamp: time.Now(),
	})
}

// GetReceivedMessages returns all captured messages for testing
func (m *MockOSCServer) GetReceivedMessages() []ReceivedMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent race conditions
	messages := make([]ReceivedMessage, len(m.receivedMessages))
	copy(messages, m.receivedMessages)
	return messages
}

// ClearReceivedMessages clears the captured messages
func (m *MockOSCServer) ClearReceivedMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.receivedMessages = make([]ReceivedMessage, 0)
}

// GetMessagesForAddress returns messages matching a specific address pattern
func (m *MockOSCServer) GetMessagesForAddress(addressPattern string) []ReceivedMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matches []ReceivedMessage
	for _, msg := range m.receivedMessages {
		if strings.Contains(msg.Address, addressPattern) {
			matches = append(matches, msg)
		}
	}
	return matches
}
