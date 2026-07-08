package qlab

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

// sync_api.go is the pure-data boundary of the QLab client (TASK-273). The
// three-way diff/conflict/merge orchestration lives in the CLI (cli/cuesync +
// cli/qlabsync); this file exposes just the transport and cache halves that
// orchestration needs: read the current workspace, load/save the merge-base
// cache, and apply a resolved list of changes over OSC. It adds no comparison
// logic.
//
// Per doc-21 §2/§5.4: CueChange carries qlab.Cue (the vendor type), NOT
// map[string]any or the IR. The library speaks only its own vocabulary; the
// adapter (cli/qlab) owns the IR translation. The legacy internal primitives
// still consume map[string]any; cueToMap adapts at the boundary until Phase D
// retires them.

// CueChangeAction is the operation a CueChange applies to the destination.
type CueChangeAction string

const (
	// CueCreate makes a new cue from Cue. Creation is non-recursive: a flat list
	// of changes carries one entry per cue (children included), each placed under
	// its parent via ParentNumber.
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
// destination cue for update/delete; Number is the cue's own number (for
// logging, error context, and parent resolution of its children); ParentNumber
// is the number of the group this cue nests under ("" for a top-level cue), used
// to place a created cue under its parent.
type CueChange struct {
	Action       CueChangeAction
	Number       string
	ParentNumber string
	UniqueID     string
	Cue          Cue
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

// SetCueFileDirectory records the directory of the CUE file being synced so the
// client can resolve relative fileTarget paths on create. The orchestrator sets
// this before an apply (the legacy TransmitWorkspaceData derived it internally).
func (q *Workspace) SetCueFileDirectory(dir string) {
	q.cueFileDirectory = dir
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

// ApplyCueChanges applies a list of resolved changes over OSC. Creation is
// non-recursive (createCue), so a flat list carrying one change per cue does not
// double-create children; a created cue is placed under its parent via
// ParentNumber, resolved through a number->uniqueID map seeded from the current
// workspace and extended as cues are created (so a parent created earlier in the
// same batch is found — callers should order parents before their children). It
// honors dry-run (writes are mocked) and stops on the first error, returning the
// count applied so far.
func (q *Workspace) ApplyCueChanges(changes []CueChange) (int, error) {
	// number -> uniqueID, for placing a created cue under its parent. Seed from
	// the live workspace so a child nests under a pre-existing parent; skip the
	// query in dry-run (there is no server and nothing is really placed).
	numberToID := make(map[string]string)
	if !q.dryRun {
		if cues, err := q.ReceiveWorkspaceData(); err == nil {
			for number, cue := range q.indexCuesFromWorkspace(map[string]any{"cues": cues}) {
				if id, ok := cue["uniqueID"].(string); ok && id != "" {
					numberToID[number] = id
				}
			}
		}
	}

	applied := 0
	for _, ch := range changes {
		switch ch.Action {
		case CueCreate:
			cueMap, err := cueToMap(ch.Cue)
			if err != nil {
				return applied, fmt.Errorf("create cue %s: %w", ch.Number, err)
			}
			uniqueID, err := q.createCue(cueMap, ch.Number)
			if err != nil {
				return applied, fmt.Errorf("create cue %s: %w", ch.Number, err)
			}
			if ch.ParentNumber != "" {
				if parentID, ok := numberToID[ch.ParentNumber]; ok {
					if err := q.moveCueToParent(uniqueID, parentID); err != nil {
						return applied, fmt.Errorf("place cue %s under %s: %w", ch.Number, ch.ParentNumber, err)
					}
				}
			}
			if ch.Number != "" {
				numberToID[ch.Number] = uniqueID
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

// cacheDirForCueFile returns the per-user cache directory and the base name used
// for a CUE file's snapshot files (~/.cache/cuejitsu/<base>_<timestamp>.json).
func cacheDirForCueFile(cueFilePath string) (dir, baseName string, err error) {
	usr, err := user.Current()
	if err != nil {
		return "", "", fmt.Errorf("get current user: %w", err)
	}
	dir = filepath.Join(usr.HomeDir, ".cache", "cuejitsu")
	baseName = strings.TrimSuffix(filepath.Base(cueFilePath), filepath.Ext(cueFilePath))
	return dir, baseName, nil
}

// LoadCache returns the most recent cached workspace snapshot for a CUE file, or
// (nil, nil) when none exists. It is the merge base (the "what we last believed
// the device held" state) the three-way engine compares against; a nil result
// means "no base" and the engine degrades to a two-way compare.
func (q *Workspace) LoadCache(cueFilePath string) (map[string]any, error) {
	dir, baseName, err := cacheDirForCueFile(cueFilePath)
	if err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(filepath.Join(dir, baseName+"_*.json"))
	if err != nil {
		return nil, fmt.Errorf("search cache files: %w", err)
	}

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
		return nil, nil
	}

	data, err := os.ReadFile(newestFile)
	if err != nil {
		return nil, fmt.Errorf("read cache file: %w", err)
	}
	var workspace map[string]any
	if err := json.Unmarshal(data, &workspace); err != nil {
		return nil, fmt.Errorf("decode cache file: %w", err)
	}
	return workspace, nil
}

// SaveCache writes workspace as the new merge-base snapshot for a CUE file. The
// orchestrator calls this after a successful apply with the state it just
// converged the device toward, so the next sync has an accurate base. This is
// the pure write half — it stores exactly what it is given, with no QLab query
// or skip-preservation (that coupling to the legacy comparison is gone).
func (q *Workspace) SaveCache(cueFilePath string, workspace map[string]any) error {
	dir, baseName, err := cacheDirForCueFile(cueFilePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(workspace, "", "  ")
	if err != nil {
		return fmt.Errorf("encode workspace: %w", err)
	}
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	cacheFilePath := filepath.Join(dir, fmt.Sprintf("%s_%s.json", baseName, timestamp))
	if err := os.WriteFile(cacheFilePath, data, 0644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}
	return nil
}
