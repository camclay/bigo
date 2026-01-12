# Contributing to BigO

Thank you for your interest in contributing to BigO! This document provides guidelines and information for contributors.

## Getting Started

### Prerequisites

- Go 1.21+
- Access to an Ollama instance (local or remote)
- Git

### Development Setup

```bash
# Fork and clone
git clone https://github.com/yourusername/bigo.git
cd bigo

# Install dependencies
go mod download

# Build
go build ./cmd/bigo

# Run tests
go test ./...
```

## How to Contribute

### Reporting Bugs

1. Check existing issues to avoid duplicates
2. Use the bug report template
3. Include:
   - BigO version (`bigo --version`)
   - Go version (`go version`)
   - OS and architecture
   - Steps to reproduce
   - Expected vs actual behavior

### Suggesting Features

1. Check the roadmap in README.md
2. Open a discussion or issue
3. Describe the use case and proposed solution

### Pull Requests

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Add tests if applicable
5. Run tests: `go test ./...`
6. Run linter: `golangci-lint run`
7. Commit with clear messages
8. Push and create a PR

## Code Style

### Go Guidelines

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Keep functions focused and small
- Add comments for exported types and functions

### Commit Messages

Use conventional commits:

```
feat: add validation system
fix: handle nil ledger in conductor
docs: update Kubernetes setup guide
refactor: extract classifier patterns
test: add worker pool tests
```

## Project Structure

```
bigo/
├── cmd/bigo/           # CLI entry point - keep minimal
├── internal/           # Private packages
│   ├── cli/           # Command implementations
│   ├── conductor/     # Core orchestration logic
│   ├── config/        # Configuration management
│   ├── ledger/        # SQLite persistence
│   ├── workers/       # Backend implementations
│   ├── validators/    # Validation system
│   └── bus/           # Message bus
├── pkg/               # Public packages (if any)
│   └── types/         # Shared types
├── docs/              # Documentation
├── examples/          # Example configurations
└── scripts/           # Utility scripts
```

## Testing

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./internal/conductor/...

# Verbose
go test -v ./...
```

### Writing Tests

- Place tests in `*_test.go` files
- Use table-driven tests where appropriate
- Mock external dependencies (Ollama, Claude)

Example:
```go
func TestClassifier_Classify(t *testing.T) {
    tests := []struct {
        name     string
        title    string
        wantTier types.Tier
    }{
        {"typo is trivial", "fix typo", types.TierTrivial},
        {"auth is critical", "implement authentication", types.TierCritical},
    }

    c := NewClassifier()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := c.Classify(tt.title, "")
            if result.Tier != tt.wantTier {
                t.Errorf("got %v, want %v", result.Tier, tt.wantTier)
            }
        })
    }
}
```

## Areas for Contribution

### Good First Issues

- Add more classification patterns
- Improve error messages
- Add CLI flags
- Documentation improvements

### Medium Complexity

- Implement validation system
- Add parallel worker support
- Create configuration validation
- Add metrics/telemetry

### Advanced

- Kubernetes integration
- OpenCode integration
- Multi-cluster support
- Web dashboard

## Questions?

- Open a GitHub Discussion
- Check existing issues and PRs
- Read the documentation in `/docs`

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
