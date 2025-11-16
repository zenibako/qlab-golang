package cue

import osc "github.com/zenibako/cuejitsu/lib/qlab/osc"

// Cue messages use workspace prefix
#CueMessage: osc.#Message & {
	useWorkspacePrefix: true
}

// Cue-level OSC messages
messages: {
	cue: {
		// === BASIC PROPERTIES ===
		name: #CueMessage & {
			address: "/cue/{cue_number}/name"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		notes: #CueMessage & {
			address: "/cue/{cue_number}/notes"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		number: #CueMessage & {
			address: "/cue/{cue_number}/number"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		type: #CueMessage & {
			address: "/cue/{cue_number}/type"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		uniqueID: #CueMessage & {
			address: "/cue/{cue_number}/uniqueID"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		listName: #CueMessage & {
			address: "/cue/{cue_number}/listName"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		flagged: #CueMessage & {
			address: "/cue/{cue_number}/flagged"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		// === TIMING & DURATION ===
		actionElapsed: #CueMessage & {
			address: "/cue/{cue_number}/actionElapsed"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		percentActionElapsed: #CueMessage & {
			address: "/cue/{cue_number}/percentActionElapsed"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		actionRemaining: #CueMessage & {
			address: "/cue/{cue_number}/actionRemaining"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "no"
				increment: "no"
			}
			example: """
				[
				  67.890 // number of seconds
				]
				"""
		}

		duration: #CueMessage & {
			address: "/cue/{cue_number}/duration"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "yes"
			}
		}

		preWait: #CueMessage & {
			address: "/cue/{cue_number}/preWait"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "yes"
			}
		}

		postWait: #CueMessage & {
			address: "/cue/{cue_number}/postWait"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "yes"
			}
		}

		allowsEditingDuration: #CueMessage & {
			address: "/cue/{cue_number}/allowsEditingDuration"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		continueMode: #CueMessage & {
			address: "/cue/{cue_number}/continueMode"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "yes"
			}
		}

		captureTimecode: #CueMessage & {
			address: "/cue/{cue_number}/captureTimecode"
			permissions: {
				view:      "no"
				edit:      "yes"
				control:   "no"
				query:     "no"
				increment: "no"
			}
		}

		// === CONTROL STATES ===
		armed: #CueMessage & {
			address: "/cue/{cue_number}/armed"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		autoLoad: #CueMessage & {
			address: "/cue/{cue_number}/autoLoad"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		useSecondColor: #CueMessage & {
			address: "/cue/{cue_number}/useSecondColor"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		auditionGo: #CueMessage & {
			address: "/cue/{cue_number}/auditionGo"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		auditionPreview: #CueMessage & {
			address: "/cue/{cue_number}/auditionPreview"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		// === PLAYBACK CONTROL ===
		go: #CueMessage & {
			address: "/cue/{cue_number}/go"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		start: #CueMessage & {
			address: "/cue/{cue_number}/start"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		stop: #CueMessage & {
			address: "/cue/{cue_number}/stop"
			permissions: {
				view:      "no"
				edit:      "yes"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		pause: #CueMessage & {
			address: "/cue/{cue_number}/pause"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		resume: #CueMessage & {
			address: "/cue/{cue_number}/resume"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		reset: #CueMessage & {
			address: "/cue/{cue_number}/reset"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		load: #CueMessage & {
			address: "/cue/{cue_number}/load"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		hardStop: #CueMessage & {
			address: "/cue/{cue_number}/hardStop"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		panic: #CueMessage & {
			address: "/cue/{cue_number}/panic"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		// === TARGETING ===
		cueTargetID: #CueMessage & {
			address: "/cue/{cue_number}/cueTargetID"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		cueTargetNumber: #CueMessage & {
			address: "/cue/{cue_number}/cueTargetNumber"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		fileTarget: #CueMessage & {
			address: "/cue/{cue_number}/fileTarget"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		patchTargetID: #CueMessage & {
			address: "/cue/{cue_number}/patchTargetID"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		canHavePatchTargets: #CueMessage & {
			address: "/cue/{cue_number}/canHavePatchTargets"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		canHaveAudioMapTargets: #CueMessage & {
			address: "/cue/{cue_number}/canHaveAudioMapTargets"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		// === STATUS QUERIES ===
		isRunning: #CueMessage & {
			address: "/cue/{cue_number}/isRunning"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		isPaused: #CueMessage & {
			address: "/cue/{cue_number}/isPaused"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		isLoaded: #CueMessage & {
			address: "/cue/{cue_number}/isLoaded"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		isBroken: #CueMessage & {
			address: "/cue/{cue_number}/isBroken"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		isAuditioning: #CueMessage & {
			address: "/cue/{cue_number}/isAuditioning"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		hasCueTargets: #CueMessage & {
			address: "/cue/{cue_number}/hasCueTargets"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		hasFileTargets: #CueMessage & {
			address: "/cue/{cue_number}/hasFileTargets"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		isPanicking: #CueMessage & {
			address: "/cue/{cue_number}/isPanicking"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		// === COLORS & DISPLAY ===
		colorName: #CueMessage & {
			address: "/cue/{cue_number}/colorName"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		"colorName/live": #CueMessage & {
			address: "/cue/{cue_number}/colorName/live"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "yes"
				query:     "yes"
				increment: "no"
			}
		}

		secondColorName: #CueMessage & {
			address: "/cue/{cue_number}/secondColorName"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		"secondColorName/live": #CueMessage & {
			address: "/cue/{cue_number}/secondColorName/live"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "yes"
				query:     "yes"
				increment: "no"
			}
		}

		// === GROUP/LIST/CART OPERATIONS ===
		children: #CueMessage & {
			address: "/cue/{cue_number}/children"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		"children/shallow": #CueMessage & {
			address: "/cue/{cue_number}/children/shallow"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		"children/uniqueIDs": #CueMessage & {
			address: "/cue/{cue_number}/children/uniqueIDs"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		"children/uniqueIDs/shallow": #CueMessage & {
			address: "/cue/{cue_number}/children/uniqueIDs/shallow"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		parent: #CueMessage & {
			address: "/cue/{cue_number}/parent"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		cartPosition: #CueMessage & {
			address: "/cue/{cue_number}/cartPosition"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "no"
				increment: "no"
			}
		}

		"cartPosition/column": #CueMessage & {
			address: "/cue/{cue_number}/cartPosition/column"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		"cartPosition/row": #CueMessage & {
			address: "/cue/{cue_number}/cartPosition/row"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		// === ADDITIONAL TARGETING ===
		currentCueTargetID: #CueMessage & {
			address: "/cue/{cue_number}/currentCueTargetID"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		currentCueTargetNumber: #CueMessage & {
			address: "/cue/{cue_number}/currentCueTargetNumber"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		tempCueTargetID: #CueMessage & {
			address: "/cue/{cue_number}/tempCueTargetID"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		tempCueTargetNumber: #CueMessage & {
			address: "/cue/{cue_number}/tempCueTargetNumber"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		// === DISPLAY & NAMING ===
		defaultName: #CueMessage & {
			address: "/cue/{cue_number}/defaultName"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		displayName: #CueMessage & {
			address: "/cue/{cue_number}/displayName"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		// === AUDIO DUCKING ===
		duckLevel: #CueMessage & {
			address: "/cue/{cue_number}/duckLevel"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "yes"
			}
		}

		duckOthers: #CueMessage & {
			address: "/cue/{cue_number}/duckOthers"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		// === ADVANCED PLAYBACK ===
		fadeAndStopOthers: #CueMessage & {
			address: "/cue/{cue_number}/fadeAndStopOthers"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		fullSurfaceID: #CueMessage & {
			address: "/cue/{cue_number}/fullSurfaceID"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		// === MORE STATUS QUERIES ===
		canHaveFileTargets: #CueMessage & {
			address: "/cue/{cue_number}/canHaveFileTargets"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		canHaveCueTargets: #CueMessage & {
			address: "/cue/{cue_number}/canHaveCueTargets"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		isOverridden: #CueMessage & {
			address: "/cue/{cue_number}/isOverridden"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		warnings: #CueMessage & {
			address: "/cue/{cue_number}/warnings"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		// === ADVANCED TIMING ===
		actionRemaining: #CueMessage & {
			address: "/cue/{cue_number}/actionRemaining"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "no"
				increment: "no"
			}
			example: """
				[
				  67.890 // number of seconds
				]
				"""
		}

		// === CONDITIONAL EXECUTION ===
		playCount: #CueMessage & {
			address: "/cue/{cue_number}/playCount"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "yes"
			}
		}

		// === MIDI & TIMECODE ===
		midiCommand: #CueMessage & {
			address: "/cue/{cue_number}/midiCommand"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		// === ADVANCED CONTROL ===
		sliderLevel: #CueMessage & {
			address: "/cue/{cue_number}/sliderLevel"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		sliderLevelString: #CueMessage & {
			address: "/cue/{cue_number}/sliderLevelString"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		// === ADDITIONAL TIMING ===
		duckTime: #CueMessage & {
			address: "/cue/{cue_number}/duckTime"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "yes"
			}
		}

		fadeAndStopOthersTime: #CueMessage & {
			address: "/cue/{cue_number}/fadeAndStopOthersTime"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "yes"
			}
		}

		currentDuration: #CueMessage & {
			address: "/cue/{cue_number}/currentDuration"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		tempDuration: #CueMessage & {
			address: "/cue/{cue_number}/tempDuration"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "yes"
			}
		}

		// === ADDITIONAL PLAYBACK CONTROL ===
		hardPause: #CueMessage & {
			address: "/cue/{cue_number}/hardPause"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		loadAt: #CueMessage & {
			address: "/cue/{cue_number}/loadAt"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		loadActionAt: #CueMessage & {
			address: "/cue/{cue_number}/loadActionAt"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		loadFileAt: #CueMessage & {
			address: "/cue/{cue_number}/loadFileAt"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		// === ADDITIONAL STATUS QUERIES ===
		isActionRunning: #CueMessage & {
			address: "/cue/{cue_number}/isActionRunning"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		isCrossfadingOut: #CueMessage & {
			address: "/cue/{cue_number}/isCrossfadingOut"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		isNextInPlaylist: #CueMessage & {
			address: "/cue/{cue_number}/isNextInPlaylist"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		isTailingOut: #CueMessage & {
			address: "/cue/{cue_number}/isTailingOut"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		isWarning: #CueMessage & {
			address: "/cue/{cue_number}/isWarning"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		// === PATCH & OUTPUT ===
		patchNumber: #CueMessage & {
			address: "/cue/{cue_number}/patchNumber"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		// === SCRIPTING & AUTOMATION ===
		playCount: #CueMessage & {
			address: "/cue/{cue_number}/playCount"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "yes"
			}
		}

		// === CONDITIONAL CONTROL ===
		ignoreUpdates: #CueMessage & {
			address: "/cue/{cue_number}/ignoreUpdates"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		// === WORKSPACE INTEGRATION ===
		surfaceID: #CueMessage & {
			address: "/cue/{cue_number}/surfaceID"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		surfaceSize: #CueMessage & {
			address: "/cue/{cue_number}/surfaceSize"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		// === ADVANCED CONTROL COMMANDS ===
		loadAndSetPlayhead: #CueMessage & {
			address: "/cue/{cue_number}/loadAndSetPlayhead"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		panicInTime: #CueMessage & {
			address: "/cue/{cue_number}/panicInTime"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		// === SEQUENCE & TIMING QUERIES ===
		maxTimeInCueSequence: #CueMessage & {
			address: "/cue/{cue_number}/maxTimeInCueSequence"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		// === ADVANCED PATCH CONTROL ===
		patchNumber: #CueMessage & {
			address: "/cue/{cue_number}/patchNumber"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		// === LEVEL & VOLUME CONTROL ===
		masterLevel: #CueMessage & {
			address: "/cue/{cue_number}/masterLevel"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "yes"
			}
		}

		// === VIDEO SPECIFIC ===
		fullScreen: #CueMessage & {
			address: "/cue/{cue_number}/fullScreen"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		// === UTILITY COMMANDS ===
		preview: #CueMessage & {
			address: "/cue/{cue_number}/preview"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		previewInTime: #CueMessage & {
			address: "/cue/{cue_number}/previewInTime"
			permissions: {
				view:      "no"
				edit:      "no"
				control:   "yes"
				query:     "no"
				increment: "no"
			}
		}

		// === SYNC & TIMING ===
		startNextCueWhenSliceEnds: #CueMessage & {
			address: "/cue/{cue_number}/startNextCueWhenSliceEnds"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		stopTargetWhenSliceEnds: #CueMessage & {
			address: "/cue/{cue_number}/stopTargetWhenSliceEnds"
			permissions: {
				view:      "read"
				edit:      "read_write"
				control:   "read"
				query:     "yes"
				increment: "no"
			}
		}

		// === GROUPING & ORGANIZATION ===
		valuesForKeys: #CueMessage & {
			address: "/cue/{cue_number}/valuesForKeys"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		// === FILE & MEDIA MANAGEMENT ===
		basicDescription: #CueMessage & {
			address: "/cue/{cue_number}/basicDescription"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		// === WORKSPACE DISPLAY ===
		cartColumns: #CueMessage & {
			address: "/cue/{cue_number}/cartColumns"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}

		cartRows: #CueMessage & {
			address: "/cue/{cue_number}/cartRows"
			permissions: {
				view:      "read_only"
				edit:      "read_only"
				control:   "read_only"
				query:     "yes"
				increment: "no"
			}
		}
	}
}
