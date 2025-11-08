# Contributing to qlab-golang

Thank you for your interest in contributing to qlab-golang!

## Development Setup

1. Clone the repository:
```bash
git clone https://github.com/zenibako/qlab-golang.git
cd qlab-golang
```

2. Install dependencies:
```bash
go mod download
```

3. Run tests:
```bash
go test ./...
```

## Building

```bash
go build ./...
```

## Code Style

This project follows standard Go conventions:
- Use `go fmt` to format code
- Run `go vet` before committing
- Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `//` for code comments (per AGENTS.md)

## Testing

- Write tests for new functionality
- Ensure all tests pass before submitting a PR
- Use the `MockOSCServer` for testing without a real QLab instance

Example test:
```go
func TestYourFeature(t *testing.T) {
    mockServer := qlab.NewMockOSCServer("127.0.0.1", 53000)
    err := mockServer.Start()
    if err != nil {
        t.Fatal(err)
    }
    defer mockServer.Stop()

    workspace := qlab.NewWorkspace("127.0.0.1", 53000)
    // ... your test code
}
```

## Submitting Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linting
5. Commit your changes using [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation changes
   - `test:` for test changes
   - `refactor:` for code refactoring
6. Push to your fork
7. Open a Pull Request

## Reporting Issues

When reporting issues, please include:
- Go version (`go version`)
- Operating system
- QLab version (if applicable)
- Minimal reproduction steps
- Expected vs actual behavior

## Questions?

Feel free to open an issue for questions or discussions.
