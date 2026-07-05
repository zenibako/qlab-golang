// Package threeway implements a destination-agnostic three-way diff and
// conflict-resolution engine for stage-cue workspaces.
//
// It compares a source workspace (e.g. parsed from a CUE file), a cached
// snapshot of the last known synced state, and the current state read back
// from a live destination (e.g. QLab), and produces a structured comparison
// describing what changed and where the three sources disagree.
//
// The package has no dependency on OSC, QLab, or any other transport. It
// operates purely on map[string]any cue data and the types defined here, so
// any destination adapter can reuse it: gather the three states, call
// PerformThreeWayComparison or PerformScopeBasedComparison to get a diff,
// call IdentifyConflicts to find what needs user input, resolve via a
// ConflictResolver, then use GenerateMergedScope/ExtractMergedWorkspaceData
// to compute the final state to apply.
//
// This is the engine a future CueDestination's Plan/Apply methods are
// expected to call into, so that every destination (QLab, ETC Eos, MIDI
// Show Control, CSV cue sheets, ...) gets three-way sync and conflict
// resolution without reimplementing it.
package threeway
