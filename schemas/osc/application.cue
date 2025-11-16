package application

import osc "github.com/zenibako/lxm/lib/qlab/osc"

// Application messages don't use workspace prefix
#ApplicationMessage: osc.#Message & {
	useWorkspacePrefix: false
}

// Application-level OSC messages
messages: {
	alwaysReply: #ApplicationMessage & {
		address: "/alwaysReply"
		args: [number]
		permissions: {
			view:    "read_write"
			edit:    "read_write"
			control: "read_write"
			query:   "yes"
		}
	}

	disconnect: #ApplicationMessage & {
		address: "/disconnect"
		permissions: {
			view:    "no"
			edit:    "yes"
			control: "no"
			query:   "no"
		}
	}

	fontNames: #ApplicationMessage & {
		address: "/fontNames"
		permissions: {
			view:    "read_only"
			edit:    "read_only"
			control: "read_only"
			query:   "no"
		}
		example: """
			[
			  "AppleColorEmoji",
			  "AppleSDGothicNeo-Bold",
			  "AppleSDGothicNeo-ExtraBold",
			  "AppleSDGothicNeo-Heavy",
			  "AppleSDGothicNeo-Light",
			  ...
			]
			"""
	}

	fontFamiliesAndStyles: #ApplicationMessage & {
		address: "/fontFamiliesAndStyles"
		permissions: {
			view:    "read_only"
			edit:    "read_only"
			control: "read_only"
			query:   "no"
		}
		example: """
			{
			  "Apple Color Emoji" :
			    [
			      "Regular"
			    ],
			  "Apple SD Gothic Neo" :
			    [
			      "Regular",
			      "Medium",
			      "Light",
			      "UltraLight",
			      "Thin",
			      "SemiBold",
			      "Bold",
			      "ExtraBold",
			      "Heavy"
			    ],
			  ...
			}
			"""
	}

	forgetMeNot: #ApplicationMessage & {
		address: "/forgetMeNot"
		args: [bool]
		permissions: {
			view:    "read_write"
			edit:    "read_write"
			control: "read_write"
			query:   "no"
		}
	}

	udpKeepAlive: #ApplicationMessage & {
		address: "/udpKeepAlive"
		args: [bool]
		permissions: {
			view:    "read_write"
			edit:    "read_write"
			control: "read_write"
			query:   "no"
		}
	}

	overrides: {
		dmxOutputEnabled: #ApplicationMessage & {
			address: "/overrides/dmxOutputEnabled"
			args: [bool]
			permissions: {
				view:    "read"
				edit:    "read_write"
				control: "read_write"
				query:   "yes"
			}
		}

		toggleDmxOutput: #ApplicationMessage & {
			address: "/overrides/toggleDmxOutput"
			permissions: {
				view:    "no"
				edit:    "yes"
				control: "yes"
				query:   "no"
			}
		}

		midiInputEnabled: #ApplicationMessage & {
			address: "/overrides/midiInputEnabled"
			args: [bool]
			permissions: {
				view:    "read"
				edit:    "read_write"
				control: "read_write"
				query:   "yes"
			}
		}
	}
}