# qlab-golang

A Go library for controlling [QLab](https://qlab.app/) via OSC (Open Sound Control).

## Features

- Connect to QLab workspaces via OSC
- Send commands and receive responses
- Query workspace state (cues, cue lists, running cues, etc.)
- Create and manipulate cues programmatically
- Mock OSC server for testing
- Type-safe message definitions
- Automatic retry and timeout handling

## Installation

```bash
go get github.com/zenibako/qlab-golang/qlab
go get github.com/zenibako/qlab-golang/messages  # Optional: OSC protocol definitions
go get github.com/zenibako/qlab-golang/templates # Optional: Cue generation types
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/zenibako/qlab-golang/qlab"
)

func main() {
    // Create a new workspace connection
    workspace := qlab.NewWorkspace("localhost", 53000)
    
    // Initialize connection with passcode
    _, err := workspace.Init("your-passcode")
    if err != nil {
        log.Fatal(err)
    }
    defer workspace.Close()
    
    // Start listening for updates
    workspace.StartUpdateListener(func(address string, args []any) {
        log.Printf("Update from QLab: %s %v", address, args)
    })
    
    // Get running cues
    runningCues := workspace.GetRunningCues()
    log.Printf("Running cues: %v", runningCues)
    
    // Get selected cues
    selectedCues := workspace.GetSelectedCues()
    log.Printf("Selected cues: %v", selectedCues)
}
```

## Sending OSC Commands

The library provides low-level access to QLab's OSC API:

```go
// Send a command and wait for reply
reply := workspace.Send("/go", "")

// Get workspace content
content := workspace.GetContent("/cueLists")
```

## Configuration

### Connection Settings

```go
workspace := qlab.NewWorkspace("localhost", 53000)

// Set timeout (in seconds)
workspace.SetTimeout(15)

// Set max retries for commands
workspace.SetMaxRetries(3)

// Enable dry-run mode (no actual changes to QLab)
workspace.SetDryRun(true)
```

### Update Listener

```go
// Listen for updates from QLab
workspace.StartUpdateListener(func(address string, args []any) {
    // Handle QLab updates
    switch {
    case strings.Contains(address, "/cue/"):
        log.Printf("Cue update: %v", args)
    case strings.Contains(address, "/playhead"):
        log.Printf("Playhead moved: %v", args)
    }
})
```

## Testing

The library includes a mock OSC server for testing:

```go
import "github.com/zenibako/qlab-golang/qlab"

// Create and start mock server
mockServer := qlab.NewMockOSCServer("127.0.0.1", 53000)
err := mockServer.Start()
if err != nil {
    t.Fatal(err)
}
defer mockServer.Stop()

// Use workspace with mock server
workspace := qlab.NewWorkspace("127.0.0.1", 53000)
// ... test your code
```

## Project Structure

```
qlab-golang/
├── qlab/                    # Main package
│   ├── workspace.go         # Workspace API
│   ├── workspace_*.go       # Workspace operations (5 files)
│   ├── cue.go              # Cue API
│   ├── cue_*.go            # Cue operations (2 files)
│   ├── osc.go              # OSC client
│   ├── mock_osc_server.go  # Mock server for testing
│   ├── *_test.go           # Unit tests (21 files)
│   └── ...
├── messages/               # OSC protocol definitions
│   └── messages.go         # Message types, addresses, builders
├── templates/              # Cue generation types
│   └── cue_templates.go    # Generic template types for cue generation
├── integration/            # Integration tests
│   └── real_qlab_test.go   # Tests requiring actual QLab instance
├── go.mod
└── README.md
```

### File Naming Convention

The project mirrors the [QLab OSC Dictionary](https://qlab.app/docs/v5/scripting/osc-dictionary-v5/) structure:

**Workspace-level operations** (`workspace_*.go`)
- Operations on `/workspace/{id}/...` endpoints
- Examples: `workspace_cache.go`, `workspace_change_detection.go`, `workspace_comparison.go`

**Cue-level operations** (`cue_*.go`)  
- Operations on `/cue/{id}/...` endpoints
- Examples: `cue.go`, `cue_templates.go`, `cue_generate.go`, `cue_writer.go`, `cue_mapping.go`

**Application-level** (other `.go` files)
- OSC client: `osc.go`
- Message handling: `messages.go`
- Testing utilities: `mock_osc_server.go`

This organization makes it easy to find the code corresponding to specific parts of QLab's OSC API.

## Documentation

For complete QLab OSC API documentation, see:
- [QLab OSC Dictionary](https://qlab.app/docs/v5/scripting/osc-dictionary-v5/)

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

This library uses:
- [go-osc](https://github.com/hypebeast/go-osc) for OSC protocol support
- [charmbracelet/log](https://github.com/charmbracelet/log) for structured logging
