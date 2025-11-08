# Agent Guidelines for qlab-golang

## Build/Test Commands
- Build: `go build -v ./...`
- Test all: `go test -v ./...` (race detector disabled - see below)
- Test single: `go test -v ./qlab -run TestFunctionName`
- Test with coverage: `go test -v -coverprofile=coverage.txt -covermode=atomic ./...`
- Lint: `golangci-lint run` (uses golangci-lint v6+)
- Format: `go fmt ./...` and `go vet ./...`
- Dependencies: `go mod download`

## Code Style
- Go version: 1.24.0
- Package structure: `qlab/` (core), `messages/` (OSC), `templates/` (generation), `integration/` (tests)
- Use `//` for all code comments (per CONTRIBUTING.md)
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Snake_case for struct fields mapping to QLab API (e.g., `workspace_id`, `file_target`)
- CamelCase for Go-native struct fields and methods
- Error handling: Return errors, don't panic; use `fmt.Errorf("context: %w", err)` for wrapping
- Logging: Use `github.com/charmbracelet/log` with levels: Debug, Info, Warn, Error
- OSC client: Use `github.com/hypebeast/go-osc/osc` for all OSC operations

## Testing
- Use `MockOSCServer` in tests: `mockServer := qlab.NewMockOSCServer("127.0.0.1", 53000); mockServer.Start(); defer mockServer.Stop()`
- Helper: `workspace, mockServer := qlab.setupWorkspaceWithCleanup(t)` for test setup
- **Race detector**: Known issues in `go-osc` library cause race warnings (see RACE_DETECTOR_NOTE.md)
  - CI runs without `-race` flag due to third-party library issues
  - All tests pass; race conditions are in `go-osc`, not our code
- Aim for atomic coverage mode in CI/CD pipelines
