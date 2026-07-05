package threeway

import (
	"fmt"
	"path/filepath"
	"strings"
)

// CompareCacheWithCurrentState compares cached workspace with current QLab state
func CompareCacheWithCurrentState(cachedWorkspace, currentWorkspace map[string]any) bool {
	// For a basic comparison, we'll check if the structure and main properties match
	// A more sophisticated comparison could check individual cue properties

	cachedCues := IndexCuesFromWorkspace(cachedWorkspace)
	currentCues := IndexCuesFromWorkspace(currentWorkspace)

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

		if !compareCueProperties(cachedCue, currentCue) {
			return false
		}
	}

	return true
}

// compareCueProperties compares the important properties of two cues
func compareCueProperties(cue1, cue2 map[string]any) bool {
	differences := CompareCuePropertiesDetailed(cue1, cue2)
	return len(differences) == 0
}

// CompareCueProperties is the exported form of compareCueProperties, for
// callers outside this package (e.g. the qlab.Workspace delegating wrapper).
func CompareCueProperties(cue1, cue2 map[string]any) bool {
	return compareCueProperties(cue1, cue2)
}

// CompareCuePropertiesDetailed compares properties and returns detailed differences
func CompareCuePropertiesDetailed(cue1, cue2 map[string]any) map[string]string {
	// List of all properties we might want to compare
	allProperties := []string{
		"name", "type", "fileTarget", "duration", "cueTargetNumber",
		"armed", "colorName", "flagged", "notes",
	}

	differences := make(map[string]string)

	for _, prop := range allProperties {
		// Only compare properties that exist in both cues or where one has a meaningful value
		val1 := normalizeProperty(cue1[prop])
		val2 := normalizeProperty(cue2[prop])

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
		if !comparePropertyValues(prop, val1, val2) {
			differences[prop] = fmt.Sprintf("'%s' -> '%s'", val1, val2)
		}
	}

	return differences
}

// comparePropertyValues applies smart comparison logic for specific properties
func comparePropertyValues(property, val1, val2 string) bool {
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
func normalizeProperty(value any) string {
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

// NormalizeProperty is the exported form of normalizeProperty, for callers
// outside this package (e.g. the qlab.Workspace delegating wrapper).
func NormalizeProperty(value any) string {
	return normalizeProperty(value)
}
