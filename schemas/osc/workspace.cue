package workspace

import osc "github.com/zenibako/cuejitsu/lib/qlab/osc"

// Workspace messages use workspace prefix
#WorkspaceMessage: osc.#Message & {
	useWorkspacePrefix: true
}

// Workspace-level OSC messages
messages: {
	workspace: {
		alwaysAudition: #WorkspaceMessage & {
			address: "/workspace/{id}/alwaysAudition"
			args: [bool]
			permissions: {
				view:    "read"
				edit:    "read_write"
				control: "read"
				query:   "yes"
			}
		}

		connect: #WorkspaceMessage & {
			address: "/workspace/{id}/connect"
			args: [string] // passcode_string
			permissions: {
				view:    "yes"
				edit:    "yes"
				control: "yes"
				query:   "no"
			}
		}

		cueLists: #WorkspaceMessage & {
			address: "/workspace/{id}/cueLists"
			permissions: {
				view:    "read_only"
				edit:    "read_only"
				control: "read_only"
				query:   "no"
			}
			example: """
				-- Cue schema
				[
				    {
				        "uniqueID": string,
				        "number": string
				        "name": string
				        "listName": string
				        "type": string
				        "colorName": string
				        "colorName/live": string
				        "flagged": number
				        "armed": number
				    }
				]
				--- Group cues
				[
				    {
				        "number": "{string}",
				        "uniqueID": {string},
				        "cues": [ {a cue dictionary}, {another dictionary}, {and another} ],
				        "flagged": true|false,
				        "listName": "{string}",
				        "type": "{string}",
				        "colorName": "{string}",
				        "colorName/live": "{string}",
				        "name": "{string}",
				        "armed": true|false,
				    }
				]
				"""
		}

		selectedCues: messages.workspace.cueLists & {
			address: "/workspace/{id}/selectedCues"
		}

		runningCues: messages.workspace.cueLists & {
			address: "/workspace/{id}/runningCues"
		}

		runningOrPausedCues: messages.workspace.cueLists & {
			address: "/workspace/{id}/runningOrPausedCues"
		}
	}
}
