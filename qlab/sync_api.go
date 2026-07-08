package qlab

import (
	"encoding/json"
	"fmt"
)

// sync_api.go is the pure-data boundary of the QLab client (TASK-273). The
// three-way diff/conflict/merge orchestration lives in the CLI (cli/cuesync +
// cli/qlabsync); this file exposes just the transport halves that orchestration
// needs: read the current workspace, and apply a resolved list of changes over
// OSC. It reuses the existing create/update/delete primitives and adds no
// comparison logic.
//
// Per doc-21 §2/§5.4: CueChange carries qlab.Cue (the vendor type), NOT
// map[string]any or the IR. The library speaks only its own vocabulary; the
// adapter (cli/qlab) owns the IR translation. The legacy internal primitives
// still consume map[string]any; cueToMap adapts at the boundary until Phase D
// retires them.

// CueChangeAction is the operation a CueChange applies to the destination.
type CueChangeAction string

const (
	// CueCreate makes a new cue from Cue (its "cues" sub-array is created too).
	CueCreate CueChangeAction = "create"
	// CueUpdate sets the fields in Cue on the existing cue identified by UniqueID.
	CueUpdate CueChangeAction = "update"
	// CueDelete removes the existing cue identified by UniqueID.
	CueDelete CueChangeAction = "delete"
)

// CueChange is one resolved change to apply over OSC. It is the pure-data
// hand-off from the CLI's sync orchestration (which computes the changes from
// the neutral engine) to this transport client: Cue carries the desired field
// values for create/update as the vendor type; UniqueID identifies the existing
// destination cue for update/delete; Number is used for logging and error
// context.
type CueChange struct {
	Action   CueChangeAction
	Number   string
	UniqueID string
	Cue      Cue
}

// ReadWorkspaceState returns the current QLab workspace as a {"cues": [...]}
// map, ready to feed to the neutral diff engine as the live destination state.
func (q *Workspace) ReadWorkspaceState() (map[string]any, error) {
	cues, err := q.ReceiveWorkspaceData()
	if err != nil {
		return nil, err
	}
	return map[string]any{"cues": cues}, nil
}

// cueToMap converts a qlab.Cue to the map[string]any shape the legacy
// create/update primitives consume. The Cue struct's JSON tags match the QLab
// OSC field names those primitives index by. This adapter exists only at the
// pure-data boundary; Phase D retires the map-shaped primitives and removes it.
func cueToMap(c Cue) (map[string]any, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("encode Cue: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("decode Cue: %w", err)
	}
	return m, nil
}

// ApplyCueChanges applies a list of resolved changes over OSC, reusing the
// existing create/update/delete primitives. It honors dry-run mode (writes are
// mocked). It stops and returns the count applied so far on the first error.
func (q *Workspace) ApplyCueChanges(changes []CueChange) (int, error) {
	applied := 0
	for _, ch := range changes {
		switch ch.Action {
		case CueCreate:
			cueMap, err := cueToMap(ch.Cue)
			if err != nil {
				return applied, fmt.Errorf("create cue %s: %w", ch.Number, err)
			}
			if _, err := q.processCueListWithParent(cueMap, "", ""); err != nil {
				return applied, fmt.Errorf("create cue %s: %w", ch.Number, err)
			}
		case CueUpdate:
			if ch.UniqueID == "" {
				return applied, fmt.Errorf("update cue %s: missing uniqueID", ch.Number)
			}
			cueMap, err := cueToMap(ch.Cue)
			if err != nil {
				return applied, fmt.Errorf("update cue %s: %w", ch.Number, err)
			}
			if err := q.updateCueProperties(ch.UniqueID, cueMap); err != nil {
				return applied, fmt.Errorf("update cue %s: %w", ch.Number, err)
			}
		case CueDelete:
			if ch.UniqueID == "" {
				return applied, fmt.Errorf("delete cue %s: missing uniqueID", ch.Number)
			}
			if err := q.DeleteCue(ch.UniqueID); err != nil {
				return applied, fmt.Errorf("delete cue %s: %w", ch.Number, err)
			}
		default:
			return applied, fmt.Errorf("cue %s: unknown change action %q", ch.Number, ch.Action)
		}
		applied++
	}
	return applied, nil
}
