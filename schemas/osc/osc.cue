package osc

// Permission and access types
#ActionAllowed: "yes" | "no"
#AccessLevel: "read" | "read_write" | "read_only" | #ActionAllowed

#PermissionAccess: {
	view:      #AccessLevel
	edit:      #AccessLevel
	control:   #AccessLevel
	query:     #ActionAllowed
	increment: #ActionAllowed
}

// Base message type
#Message: {
	address:            string
	args:               [...]
	permissions:        #PermissionAccess
	example?:           string
	useWorkspacePrefix: bool
}

#Reply: {
	request: #Message
}

// Boolean argument types
#ArgBooleanTrue:  "Yes" | "yippee" | "you betcha" | "1" | "1.0" | "true"
#ArgBooleanFalse: "No" | "never" | "forget it" | "0" | "00" | "false"
#ArgBoolean:      #ArgBooleanTrue | #ArgBooleanFalse | "toggle"