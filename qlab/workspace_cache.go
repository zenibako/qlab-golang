package qlab

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

// getKeys returns sorted keys from a map for debugging
func getKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// writeCueFileToCache saves the current QLab workspace state to cache for change detection
// If comparison is provided, it preserves cached state for skipped cues to maintain user choices
func (q *Workspace) writeCueFileToCache(filePath string, workspace map[string]any, mapping *CueMapping, comparison *ThreeWayComparison) error {
	// Get current user's home directory
	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

	// Create cache directory
	cacheDir := filepath.Join(usr.HomeDir, ".cache", "cuejitsu")
	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create cache directory: %v", err)
	}

	// Get base filename without extension
	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	// Create timestamp for the cache file
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	cacheFileName := fmt.Sprintf("%s_%s.json", baseName, timestamp)
	cacheFilePath := filepath.Join(cacheDir, cacheFileName)

	// Query current QLab workspace state
	currentWorkspace, err := q.queryCurrentWorkspaceState()
	if err != nil {
		return fmt.Errorf("failed to query current workspace state for caching: %v", err)
	}

	// If comparison is provided, preserve cached state for skipped cues
	if comparison != nil && comparison.HasCache {
		// Load the original cache to preserve skipped cues
		originalCacheFilePath, err := findMostRecentCacheFile(filePath)
		if err == nil {
			originalCache, err := loadCacheFileData(originalCacheFilePath)
			if err == nil {
				// Index cues from original cache
				originalCues := q.indexCuesFromWorkspace(originalCache)

				// For each cue that was skipped, restore its original cached state
				for cueNumber, result := range comparison.CueResults {
					if result.Action == "skip" && result.Reason == "User chose to skip this cue" {
						// Preserve original cached state for this cue
						if originalCue, exists := originalCues[cueNumber]; exists {
							log.Debugf("Preserving original cached state for skipped cue: %s", cueNumber)
							// Replace the current state with the original cached state
							err := q.replaceWorkspaceCueWithCached(currentWorkspace, originalCue, cueNumber)
							if err != nil {
								log.Warnf("Failed to preserve cached state for cue %s: %v", cueNumber, err)
							}
						}
					}
				}
			}
		}
	}

	// Write the current workspace state to cache file
	cacheData, err := json.MarshalIndent(currentWorkspace, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %v", err)
	}

	err = os.WriteFile(cacheFilePath, cacheData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write cache file: %v", err)
	}

	log.Infof("Saved workspace state to cache: %s", cacheFilePath)
	return nil
}

// queryWorkspaceStateLightweight performs a minimal query when full query times out
// Returns basic cue structure without deep enrichment
func (q *Workspace) queryWorkspaceStateLightweight() (map[string]any, error) {
	log.Info("Using lightweight query mode - fetching cue list names only")

	address := fmt.Sprintf("/workspace/%s/cueLists/shallow", q.workspace_id)
	reply := q.Send(address, "")

	if len(reply) == 0 {
		return nil, fmt.Errorf("no reply from lightweight query")
	}

	replyStr, ok := reply[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid reply format")
	}

	var replyData map[string]any
	if err := json.Unmarshal([]byte(replyStr), &replyData); err != nil {
		return nil, fmt.Errorf("failed to parse reply: %v", err)
	}

	// Check for errors including timeouts
	if status, ok := replyData["status"].(string); ok && status == "error" {
		if errorMsg, hasError := replyData["error"].(string); hasError && strings.Contains(errorMsg, "timeout") {
			log.Warn("Lightweight query also timed out - QLab connection may be unstable")
		}
		return nil, formatErrorWithJSON("lightweight query failed", replyStr)
	}

	log.Info("Lightweight query succeeded - using basic cue structure")
	return replyData, nil
}

// queryCurrentWorkspaceState queries the current QLab workspace state for caching/comparison
func (q *Workspace) queryCurrentWorkspaceState() (map[string]any, error) {
	// Try multiple approaches to get all cues in the workspace

	// Approach 1: Try /cueLists (should work if cue lists are Group cues with children)
	log.Info("Attempting to fetch cues using /cueLists")
	address := fmt.Sprintf("/workspace/%s/cueLists", q.workspace_id)
	reply := q.Send(address, "")

	if len(reply) == 0 {
		log.Warn("No reply received from /cueLists - QLab may be busy or disconnected")
		return nil, fmt.Errorf("no reply received from QLab when querying workspace state")
	}

	replyStr, ok := reply[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid reply format from QLab workspace query")
	}

	var replyData map[string]any
	err := json.Unmarshal([]byte(replyStr), &replyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse QLab workspace reply: %v", err)
	}

	// Check for error status - including timeout errors
	if status, ok := replyData["status"].(string); ok && status == "error" {
		// Check if this is a timeout error
		if errorMsg, hasError := replyData["error"].(string); hasError {
			if strings.Contains(errorMsg, "timeout") {
				log.Warn("QLab query timed out - workspace may be too large or QLab is busy")
				log.Info("Consider increasing timeout with SetTimeout() or reducing workspace size")
			}
		}
		return nil, formatErrorWithJSON("QLab error querying workspace state", replyStr)
	}

	// Check if we have cue lists with actual cue children
	data, ok := replyData["data"].([]any)
	if !ok {
		return replyData, nil // Return as-is if no data array
	}

	log.Info("Received cue lists data", "count", len(data))

	// Count total cues across all lists to see if we have actual cue data
	totalCues := 0
	for _, cueListInterface := range data {
		if cueList, ok := cueListInterface.(map[string]any); ok {
			if cuesArray, exists := cueList["cues"]; exists {
				if cues, ok := cuesArray.([]any); ok {
					totalCues += len(cues)
				}
			}
		}
	}

	log.Info("Total cues found in /cueLists", "count", totalCues)

	// If we found actual cues, enrich and return the data
	if totalCues > 0 {
		log.Info("Successfully retrieved cues using /cueLists")
		q.enrichCuesWithProperties(replyData)
		return replyData, nil
	}

	// Approach 2: DISABLED - /selectedCues approach has timeout issues
	// This approach doesn't work reliably with QLab and causes 20+ second delays
	log.Info("Skipping /selectedCues approach (disabled due to timeout issues)")

	// Approach 3: Try individual cue list traversal
	log.Info("Trying individual cue list traversal")

	for i, cueListInterface := range data {
		cueList, ok := cueListInterface.(map[string]any)
		if !ok {
			continue
		}

		// Check if this cue list already has cues
		if cuesArray, exists := cueList["cues"]; exists {
			if cues, ok := cuesArray.([]any); ok && len(cues) > 0 {
				log.Info("Cue list already has cues", "index", i, "count", len(cues))
				continue // This cue list already has cue data
			}
		}

		// Try to get either uniqueID or number for the cue list
		var cueIdentifier string
		var childrenAddress string

		// First, try uniqueID (preferred method)
		if uniqueID, exists := cueList["uniqueID"]; exists {
			if uniqueIDStr, ok := uniqueID.(string); ok && uniqueIDStr != "" {
				cueIdentifier = uniqueIDStr
				childrenAddress = fmt.Sprintf("/workspace/%s/cue_id/%s/children", q.workspace_id, uniqueIDStr)
				log.Info("Fetching cues for cue list", "index", i, "uniqueID", uniqueIDStr)
			}
		}

		// Fallback to number if uniqueID not available
		if cueIdentifier == "" {
			if listNumber, exists := cueList["number"]; exists {
				if listNumberStr, ok := listNumber.(string); ok && listNumberStr != "" {
					cueIdentifier = listNumberStr
					childrenAddress = fmt.Sprintf("/workspace/%s/cue/%s/children", q.workspace_id, listNumberStr)
					log.Info("Fetching cues for cue list", "index", i, "number", listNumberStr)
				}
			}
		}

		// Skip if no identifier found
		if cueIdentifier == "" {
			log.Warn("Cue list has no number or uniqueID", "index", i)
			continue
		}
		childrenReply := q.Send(childrenAddress, "")

		if len(childrenReply) == 0 {
			log.Warn("No reply received for cue list children", "identifier", cueIdentifier)
			continue
		}

		childrenStr, ok := childrenReply[0].(string)
		if !ok {
			log.Warn("Invalid reply format for cue list children", "identifier", cueIdentifier)
			continue
		}

		var childrenData map[string]any
		err := json.Unmarshal([]byte(childrenStr), &childrenData)
		if err != nil {
			log.Error("Failed to parse cue list children", "identifier", cueIdentifier, "error", err)
			continue
		}

		// Check for error status
		if status, ok := childrenData["status"].(string); ok && status == "error" {
			log.Error("QLab error fetching children for cue list", "identifier", cueIdentifier, "response", childrenStr)
			continue
		}

		// Extract the cues and add them to the cue list
		if childrenCues, ok := childrenData["data"].([]any); ok {
			cueList["cues"] = childrenCues
			log.Info("Successfully fetched cues for cue list", "identifier", cueIdentifier, "count", len(childrenCues))
		} else {
			log.Warn("No cues data found in children response for cue list", "identifier", cueIdentifier)
		}
	}

	// Enrich cues with additional properties not included in /cueLists
	q.enrichCuesWithProperties(replyData)

	// Enrich cues with additional properties not included in /cueLists
	q.enrichCuesWithProperties(replyData)

	// Return the enhanced workspace data
	return replyData, nil
}

// enrichCuesWithProperties queries additional cue properties not included in /cueLists response
// According to QLab OSC docs, /cueLists only returns: uniqueID, number, name, listName, type,
// colorName, flagged, armed. We need to query fileTarget and other properties separately.
func (q *Workspace) enrichCuesWithProperties(workspace map[string]any) {
	data, ok := workspace["data"].([]any)
	if !ok {
		return
	}

	for _, cueListData := range data {
		if cueList, ok := cueListData.(map[string]any); ok {
			if cues, ok := cueList["cues"].([]any); ok {
				q.enrichCueArrayWithProperties(cues)
			}
		}
	}
}

// enrichCueArrayWithProperties recursively enriches an array of cues with additional properties
func (q *Workspace) enrichCueArrayWithProperties(cues []any) {
	for _, cueData := range cues {
		if cue, ok := cueData.(map[string]any); ok {
			// Get uniqueID for property queries
			uniqueID, ok := cue["uniqueID"].(string)
			if !ok || uniqueID == "" {
				continue
			}

			// Query fileTarget property
			q.queryCueProperty(cue, uniqueID, "fileTarget")

			// Query cueTargetNumber property
			q.queryCueProperty(cue, uniqueID, "cueTargetNumber")

			// Recursively enrich child cues
			if children, ok := cue["cues"].([]any); ok {
				q.enrichCueArrayWithProperties(children)
			}
		}
	}
}

// queryCueProperty queries a single property from QLab and adds it to the cue map if not empty
func (q *Workspace) queryCueProperty(cue map[string]any, uniqueID, property string) {
	address := fmt.Sprintf("/workspace/%s/cue_id/%s/%s", q.workspace_id, uniqueID, property)
	reply := q.Send(address, "")
	log.Debug("Querying cue property", "uniqueID", uniqueID, "property", property, "reply_count", len(reply))
	if len(reply) > 0 {
		if replyStr, ok := reply[0].(string); ok {
			log.Debug("Got reply for property", "property", property, "reply", replyStr)
			var replyData map[string]any
			if err := json.Unmarshal([]byte(replyStr), &replyData); err == nil {
				if status, ok := replyData["status"].(string); ok && status == "ok" {
					if value, ok := replyData["data"].(string); ok && value != "" {
						cue[property] = value
						log.Debug("Enriched cue with property", "uniqueID", uniqueID, "property", property, "value", value)
					} else {
						log.Debug("Property value is empty or not a string", "property", property, "data", replyData["data"])
					}
				} else {
					log.Debug("Property query status not ok", "property", property, "status", status)
				}
			}
		}
	}
}

// findMostRecentCacheFile finds the most recent cache file for a given CUE file
func findMostRecentCacheFile(filePath string) (string, error) {
	// Get current user's home directory
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %v", err)
	}

	cacheDir := filepath.Join(usr.HomeDir, ".cache", "cuejitsu")

	// Check if cache directory exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return "", fmt.Errorf("cache directory does not exist")
	}

	// Get base filename without extension
	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	// Find all cache files matching the pattern
	pattern := filepath.Join(cacheDir, baseName+"_*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to search for cache files: %v", err)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no cache files found for %s", baseName)
	}

	// Sort by modification time (most recent first)
	// We'll use a simple approach: sort by filename (timestamp is in filename)
	var newestFile string
	var newestTime time.Time

	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newestFile = match
		}
	}

	if newestFile == "" {
		return "", fmt.Errorf("no valid cache files found")
	}

	return newestFile, nil
}

// loadCacheFileData loads workspace data from a cache file
func loadCacheFileData(cacheFilePath string) (map[string]any, error) {
	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %v", err)
	}

	var workspace map[string]any
	err = json.Unmarshal(data, &workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache data: %v", err)
	}

	return workspace, nil
}

// extractCueIdentifier extracts the cue identifier (similar to indexCuesFromWorkspace logic)
func (q *Workspace) extractCueIdentifier(cue map[string]any, parentNumber string) string {
	// Extract cue number (same logic as indexCuesFromWorkspace)
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

	// If no number, create position-based identifier (same as indexCuesFromWorkspace)
	if fullNumber == "" {
		cueName, _ := cue["name"].(string)
		cueType, _ := cue["type"].(string)
		// Use position-based identification for cues without numbers
		fullNumber = fmt.Sprintf("[%s:%s]", cueType, cueName)
	}

	return fullNumber
}

// replaceWorkspaceCueWithCached replaces a cue in the workspace structure with cached data
func (q *Workspace) replaceWorkspaceCueWithCached(workspace map[string]any, cachedCue map[string]any, cueNumber string) error {
	// Navigate through the workspace structure to find and replace the cue
	data, ok := workspace["data"].([]any)
	if !ok {
		return fmt.Errorf("no data array found in workspace")
	}

	// Look through each cue list
	for _, cueListInterface := range data {
		cueList, ok := cueListInterface.(map[string]any)
		if !ok {
			continue
		}

		// Check if this cue list has cues
		if cuesArray, exists := cueList["cues"]; exists {
			if cues, ok := cuesArray.([]any); ok {
				// Look for the cue to replace
				if q.findAndReplaceCueInArray(cues, cachedCue, cueNumber, "") {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("cue %s not found in workspace structure", cueNumber)
}

// findAndReplaceCueInArray searches for and replaces a cue in an array, handling nested structures
func (q *Workspace) findAndReplaceCueInArray(cues []any, cachedCue map[string]any, cueNumber string, parentNumber string) bool {
	for i, cueInterface := range cues {
		if cue, ok := cueInterface.(map[string]any); ok {
			// Check if this is the cue we're looking for
			if q.extractCueIdentifier(cue, parentNumber) == cueNumber {
				// Found the cue - replace it with cached data
				cues[i] = cachedCue
				log.Debugf("Replaced cue %s with cached data", cueNumber)
				return true
			}

			// Check nested cues (for Group cues)
			if childCuesArray, exists := cue["cues"]; exists {
				if childCues, ok := childCuesArray.([]any); ok {
					// Build parent number for children
					childParentNumber := q.extractCueIdentifier(cue, parentNumber)
					if q.findAndReplaceCueInArray(childCues, cachedCue, cueNumber, childParentNumber) {
						return true
					}
				}
			}
		}
	}
	return false
}

// PerformThreeWayComparison compares source CUE file, cache, and current QLab state
func (q *Workspace) PerformThreeWayComparison(filePath string, sourceCueData map[string]any) (*ThreeWayComparison, error) {
	log.Debugf("PerformThreeWayComparison called for file: %s", filePath)
	comparison := &ThreeWayComparison{
		CueResults:       make(map[string]*CueChangeResult),
		HasCache:         false,
		HasQLabData:      false,
		CacheMatchesQLab: false,
		QLabChosenCues:   make(map[string]bool),
		QLabChosenFields: make(map[string]map[string]bool),
		CurrentQLabData:  make(map[string]any),
		WorkspaceScope:   nil,
		MergedResult:     nil,
	}

	// Step 1: Try to load cache data
	var cachedWorkspace map[string]any
	cacheFilePath, err := findMostRecentCacheFile(filePath)
	if err != nil {
		log.Infof("No cache file found: %v", err)
	} else {
		cachedWorkspace, err = loadCacheFileData(cacheFilePath)
		if err != nil {
			log.Warnf("Failed to load cache data: %v", err)
		} else {
			comparison.HasCache = true
			log.Infof("Loaded cache from: %s", cacheFilePath)
		}
	}

	// Step 2: Query current QLab workspace state
	var currentWorkspace map[string]any
	currentWorkspace, err = q.queryCurrentWorkspaceState()
	if err != nil {
		if q.wasConnected {
			log.Warnf("Failed to query current QLab state: %v", err)

			// Try lightweight fallback query if full query times out
			if strings.Contains(err.Error(), "timeout") {
				log.Info("Attempting lightweight fallback query...")
				currentWorkspace, err = q.queryWorkspaceStateLightweight()
				if err == nil {
					log.Info("Lightweight fallback query succeeded")
					comparison.HasQLabData = true
					comparison.CurrentQLabData = currentWorkspace
				} else {
					log.Warnf("Lightweight fallback query also failed: %v", err)
					comparison.HasQLabData = false
					comparison.CurrentQLabData = nil
				}
			} else {
				// IMPORTANT: Set HasQLabData to false when query fails
				comparison.HasQLabData = false
				comparison.CurrentQLabData = nil
			}
		} else {
			log.Debugf("Failed to query current QLab state (not connected): %v", err)
			comparison.HasQLabData = false
			comparison.CurrentQLabData = nil
		}
	} else {
		comparison.HasQLabData = true
		comparison.CurrentQLabData = currentWorkspace
		log.Info("Queried current QLab workspace state")
	}

	// Step 3: Compare cache with current QLab state if both available
	if comparison.HasCache && comparison.HasQLabData {
		comparison.CacheMatchesQLab = q.compareCacheWithCurrentState(cachedWorkspace, currentWorkspace)
		if comparison.CacheMatchesQLab {
			log.Info("Cache matches current QLab state")
		} else {
			log.Warn("Cache differs from current QLab state")
		}

		// Perform scope-based comparison for granular conflict detection
		log.Debug("Performing scope-based comparison")
		scopeComparison, err := q.PerformScopeBasedComparison(sourceCueData, cachedWorkspace, currentWorkspace)
		if err != nil {
			log.Warnf("Scope-based comparison failed: %v", err)
		} else {
			comparison.WorkspaceScope = scopeComparison
			log.Infof("Scope-based comparison complete: hasChanges=%t, hasConflicts=%t",
				scopeComparison.HasChanges, scopeComparison.ConflictExists)
		}
	} else if comparison.HasCache && !comparison.HasQLabData {
		// Cache exists but QLab query failed - use cache as fallback for comparison
		log.Warn("QLab query failed - using cache-only comparison mode")
		log.Info("Will compare source against cached state (QLab state unavailable)")
	}

	// Step 4: Build cue comparison results
	sourceCues := q.indexCuesFromWorkspace(sourceCueData)

	var cachedCues map[string]map[string]any
	var currentCues map[string]map[string]any

	if comparison.HasCache {
		cachedCues = q.indexCuesFromWorkspace(cachedWorkspace)
	}
	if comparison.HasQLabData {
		currentCues = q.indexCuesFromWorkspace(currentWorkspace)
	} else {
		// Initialize empty map to prevent nil pointer issues
		currentCues = make(map[string]map[string]any)
	}

	// Compare each source cue
	for cueNumber, sourceCue := range sourceCues {
		result := &CueChangeResult{
			HasChanged:     true,
			Action:         "create",
			Reason:         "new cue",
			FieldConflicts: make(map[string]*FieldConflict),
		}

		// Check if cue exists in current QLab state
		if currentCue, existsInQLab := currentCues[cueNumber]; existsInQLab {
			// Extract existing ID
			if id, ok := currentCue["uniqueID"].(string); ok {
				result.ExistingID = id
			}

			// Debug position-based cues specifically
			if strings.Contains(cueNumber, "[audio:") {
				log.Debugf("Position-based audio cue found in QLab: %s", cueNumber)
				log.Debugf("Checking if exists in cache...")
			}

			// Check if cue exists in cache
			if cachedCue, existsInCache := cachedCues[cueNumber]; existsInCache {
				if strings.Contains(cueNumber, "[audio:") {
					log.Debugf("Position-based audio cue FOUND in cache: %s", cueNumber)
				}
				// Debug: Show first cue properties in detail
				if cueNumber == "0" {
					log.Debugf("=== CUE 0 DETAILED COMPARISON ===")
					log.Debugf("Source cue keys: %v", getKeys(sourceCue))
					log.Debugf("Cached cue keys: %v", getKeys(cachedCue))
					log.Debugf("Current cue keys: %v", getKeys(currentCue))
					log.Debugf("Source name: '%v'", sourceCue["name"])
					log.Debugf("Cached name: '%v'", cachedCue["name"])
					log.Debugf("Current name: '%v'", currentCue["name"])
				}

				// Three-way comparison: source vs cache vs current
				sourceCacheDiffs := q.compareCuePropertiesDetailed(sourceCue, cachedCue)
				cacheCurrentDiffs := q.compareCuePropertiesDetailed(cachedCue, currentCue)
				sourceMatchesCache := len(sourceCacheDiffs) == 0
				cacheMatchesCurrent := len(cacheCurrentDiffs) == 0

				// Store cue ID for traceability
				if cueID, ok := currentCue["uniqueID"].(string); ok {
					result.CueID = cueID
					result.ExistingID = cueID // For backward compatibility with existing logging
				}

				if sourceMatchesCache && cacheMatchesCurrent {
					// Source == Cache == Current: No changes needed
					result.HasChanged = false
					result.Action = "skip"
					result.Reason = "unchanged since last transmission"
					result.ModifiedFields = make(map[string]string)
				} else if sourceMatchesCache && !cacheMatchesCurrent {
					// Source == Cache != Current: QLab was modified externally
					result.HasChanged = true
					result.Action = "update"
					result.Reason = "QLab modified externally, reverting to source"
					result.ModifiedFields = cacheCurrentDiffs
				} else if !sourceMatchesCache && cacheMatchesCurrent {
					// Source != Cache == Current: Source was modified
					result.HasChanged = true
					result.Action = "update"
					result.Reason = "source file modified"
					result.ModifiedFields = sourceCacheDiffs
				} else {
					// Source != Cache != Current: Both modified
					result.HasChanged = true
					result.Action = "update"
					result.Reason = "both source and QLab modified"
					// Merge both difference sets for complete visibility
					result.ModifiedFields = make(map[string]string)
					for field, diff := range sourceCacheDiffs {
						result.ModifiedFields["source_vs_cache_"+field] = diff
					}
					for field, diff := range cacheCurrentDiffs {
						result.ModifiedFields["cache_vs_current_"+field] = diff
					}
				}
			} else {
				// Exists in QLab but not in cache - compare source vs current
				sourceCurrentDiffs := q.compareCuePropertiesDetailed(sourceCue, currentCue)

				// Store cue ID for traceability
				if cueID, ok := currentCue["uniqueID"].(string); ok {
					result.CueID = cueID
					result.ExistingID = cueID // For backward compatibility with existing logging
				}

				if len(sourceCurrentDiffs) == 0 {
					result.HasChanged = false
					result.Action = "skip"
					result.Reason = "matches current QLab state"
					result.ModifiedFields = make(map[string]string)
				} else {
					result.HasChanged = true
					result.Action = "update"
					result.Reason = "differs from current QLab state"
					result.ModifiedFields = sourceCurrentDiffs
				}
			}
		} else {
			// Cue doesn't exist in QLab
			result.HasChanged = true
			result.Action = "create"
			result.Reason = "new cue"
			result.ModifiedFields = make(map[string]string) // No existing cue to compare against
		}

		comparison.CueResults[cueNumber] = result
	}

	// Link scope data to cue results if scope comparison was performed
	if comparison.WorkspaceScope != nil {
		q.linkScopeDataToCueResults(comparison)
	}

	return comparison, nil
}

// linkScopeDataToCueResults links scope comparison data to individual cue results
func (q *Workspace) linkScopeDataToCueResults(comparison *ThreeWayComparison) {
	if comparison.WorkspaceScope == nil {
		return
	}

	for _, cueScope := range comparison.WorkspaceScope.ChildScopes {
		if cueScope.Scope == ScopeCue {
			if result, exists := comparison.CueResults[cueScope.Identifier]; exists {
				result.ScopeData = cueScope
				result.FieldConflicts = cueScope.FieldChanges
			}
		}
	}
}

// PrintThreeWayComparisonResults outputs a detailed summary of the three-way comparison
func (q *Workspace) PrintThreeWayComparisonResults(comparison *ThreeWayComparison) {
	log.Info("=== Three-Way Comparison Results ===")

	// Print overall status
	log.Infof("Has Cache: %t", comparison.HasCache)
	log.Infof("Has QLab Data: %t", comparison.HasQLabData)
	log.Infof("Cache Matches QLab: %t", comparison.CacheMatchesQLab)

	// Count results by action
	actionCounts := map[string]int{
		"create": 0,
		"update": 0,
		"skip":   0,
	}

	for _, result := range comparison.CueResults {
		actionCounts[result.Action]++
	}

	log.Infof("Action Summary: %d create, %d update, %d skip",
		actionCounts["create"], actionCounts["update"], actionCounts["skip"])

	// Print detailed results for each cue
	if len(comparison.CueResults) > 0 {
		log.Info("--- Cue-by-Cue Results ---")
		for cueNumber, result := range comparison.CueResults {
			status := "CHANGED"
			if !result.HasChanged {
				status = "UNCHANGED"
			}

			// Build cue identification info
			cueInfo := fmt.Sprintf("Cue [%s]", cueNumber)
			if result.CueID != "" {
				cueInfo += fmt.Sprintf(" (ID: %s)", result.CueID)
			}
			if result.ExistingID != "" {
				cueInfo += fmt.Sprintf(" (existing ID: %s)", result.ExistingID)
			}

			log.Infof("%s: %s - Action: %s - Reason: %s",
				cueInfo, status, result.Action, result.Reason)

			// Show field differences if any
			if len(result.ModifiedFields) > 0 {
				log.Info("  Modified fields:")
				for field, diff := range result.ModifiedFields {
					log.Infof("    %s: %s", field, diff)
				}
			}
		}
	} else {
		log.Info("No cues found in source file")
	}

	log.Info("=== End Three-Way Comparison ===")
}
