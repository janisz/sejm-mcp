# Contributing to Sejm MCP Server

Thank you for your interest in contributing to the Sejm MCP Server! This document provides guidelines and information for contributors.

## Table of Contents

1. [Code of Conduct](#code-of-conduct)
2. [Getting Started](#getting-started)
3. [Development Setup](#development-setup)
4. [Project Structure](#project-structure)
5. [Contributing Guidelines](#contributing-guidelines)
6. [Testing](#testing)
7. [Documentation](#documentation)
8. [Pull Request Process](#pull-request-process)

## Code of Conduct

This project adheres to a code of conduct to ensure a welcoming environment for all contributors. By participating, you agree to:

- Be respectful and inclusive
- Focus on constructive feedback
- Help create a positive learning environment
- Respect different viewpoints and experiences

## Getting Started

### Prerequisites

- Go 1.21 or later
- Git
- `yq` for YAML processing
- `oapi-codegen` for type generation

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
```bash
git clone https://github.com/YOUR_USERNAME/sejm-mcp.git
cd sejm-mcp
```

3. Add the upstream repository:
```bash
git remote add upstream https://github.com/janisz/sejm-mcp.git
```

## Development Setup

### Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install development tools
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

# Verify yq is available
which yq || echo "Please install yq for YAML processing"
```

### Build and Test

```bash
# Build the project
go build -o sejm-mcp ./cmd/sejm-mcp

# Run tests
go test ./...

# Generate types from OpenAPI specs
make generate-types  # If Makefile exists, or run commands manually
```

### Regenerate Types

When OpenAPI specifications change or you need fresh types:

```bash
# Download latest specs
curl -s https://api.sejm.gov.pl/sejm/openapi/ -o sejm-openapi.json
curl -s https://api.sejm.gov.pl/eli/openapi/ -o eli-openapi.json

# Convert to JSON
yq eval -j sejm-openapi.json > sejm-openapi-converted.json
yq eval -j eli-openapi.json > eli-openapi-converted.json

# Generate Go types
oapi-codegen -config sejm-codegen.yaml sejm-openapi-converted.json
oapi-codegen -config eli-codegen.yaml eli-openapi-converted.json
```

## Project Structure

```
├── cmd/sejm-mcp/          # Main application entry point
│   └── main.go           # Application bootstrap
├── internal/server/       # Private server implementation
│   ├── server.go         # Core MCP server setup
│   ├── sejm_tools.go     # Sejm API tool implementations
│   └── eli_tools.go      # ELI API tool implementations
├── pkg/                   # Public packages
│   ├── sejm/             # Generated Sejm API types
│   └── eli/              # Generated ELI API types
├── docs/                  # Documentation
│   ├── API_REFERENCE.md  # Complete API documentation
│   ├── EXAMPLES.md       # Usage examples
│   └── CONTRIBUTING.md   # This file
├── *-codegen.yaml        # OpenAPI code generation configs
├── go.mod               # Go module definition
├── README.md            # Project overview
└── .gitignore          # Git ignore patterns
```

### Key Design Principles

1. **Generated Types**: All API types are generated from OpenAPI specs
2. **Clean Architecture**: Separation of concerns between server, tools, and types
3. **Error Handling**: Consistent error responses across all tools
4. **Documentation**: Comprehensive documentation for all features

## Contributing Guidelines

### Types of Contributions

We welcome several types of contributions:

#### 🐛 Bug Fixes
- Fix type generation issues
- Resolve API integration problems
- Correct documentation errors
- Address performance issues

#### ✨ New Features
- Additional MCP tools for existing APIs
- Integration with new Polish government APIs
- Enhanced error handling and logging
- Performance optimizations

#### 📚 Documentation
- API reference improvements
- Usage examples and tutorials
- Code comments and inline documentation
- Architecture and design documentation

#### 🧪 Testing
- Unit tests for tool implementations
- Integration tests with live APIs
- Performance and load testing
- Documentation testing

### Before Contributing

1. **Check existing issues**: Look for related issues or feature requests
2. **Discuss major changes**: Open an issue to discuss significant features
3. **Follow conventions**: Maintain consistency with existing code style
4. **Test thoroughly**: Ensure your changes don't break existing functionality

## Development Guidelines

### Code Style

Follow standard Go conventions:

```go
// Good: Clear function names and documentation
// handleGetMPs retrieves Members of Parliament for a specific term
func (s *SejmServer) handleGetMPs(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
    term, err := s.validateTerm(getStringArg(arguments, "term"))
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Invalid term: %v", err)), nil
    }
    // ... implementation
}
```

### Error Handling

Use consistent error handling patterns:

```go
// API request errors
if err != nil {
    return mcp.NewToolResultError(fmt.Sprintf("API request failed: %v", err)), nil
}

// Validation errors
if required_param == "" {
    return mcp.NewToolResultError("parameter_name is required"), nil
}

// JSON parsing errors
if err := json.Unmarshal(data, &result); err != nil {
    return mcp.NewToolResultError(fmt.Sprintf("Failed to parse response: %v", err)), nil
}
```

### Adding New Tools

When adding new MCP tools:

1. **Define the tool schema** in the appropriate `register*Tools()` function
2. **Implement the handler** following existing patterns
3. **Add comprehensive documentation** in API_REFERENCE.md
4. **Create usage examples** in EXAMPLES.md
5. **Write tests** for the new functionality

Example tool addition:

```go
// In sejm_tools.go or eli_tools.go
s.server.AddTool(mcp.Tool{
    Name:        "tool_name",
    Description: "Clear description of what the tool does",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param_name": map[string]interface{}{
                "type":        "string",
                "description": "Parameter description",
            },
        },
        "required": []string{"required_param"},
    },
}, s.handleNewTool)

func (s *SejmServer) handleNewTool(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
    // Implementation here
}
```

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/server/

# Run tests with verbose output
go test -v ./...
```

### Writing Tests

Create test files alongside implementation:

```go
// internal/server/sejm_tools_test.go
package server

import (
    "context"
    "testing"
)

func TestHandleGetMPs(t *testing.T) {
    server := NewSejmServer()

    tests := []struct {
        name      string
        arguments map[string]interface{}
        wantErr   bool
    }{
        {
            name:      "valid term",
            arguments: map[string]interface{}{"term": "10"},
            wantErr:   false,
        },
        {
            name:      "invalid term",
            arguments: map[string]interface{}{"term": "invalid"},
            wantErr:   true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := server.handleGetMPs(context.Background(), tt.arguments)
            if (err != nil) != tt.wantErr {
                t.Errorf("handleGetMPs() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            // Additional assertions...
        })
    }
}
```

### Integration Testing

For testing against live APIs:

```go
func TestLiveAPIIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Test against real API
    server := NewSejmServer()
    result, err := server.handleGetMPs(context.Background(), map[string]interface{}{"term": "10"})

    if err != nil {
        t.Fatalf("Integration test failed: %v", err)
    }

    // Validate real response structure
}
```

Run integration tests:
```bash
go test ./... -tags=integration
```

## Documentation

### API Reference Updates

When adding or modifying tools, update `docs/API_REFERENCE.md`:

1. Add tool description and parameters
2. Provide example requests and responses
3. Document error conditions
4. Include use case suggestions

### Example Updates

Add practical examples to `docs/EXAMPLES.md`:

1. Show realistic usage scenarios
2. Provide complete workflows
3. Include error handling examples
4. Demonstrate best practices

### Code Documentation

Use Go doc conventions:

```go
// Package server implements the MCP server for Polish Parliament APIs.
package server

// SejmServer provides MCP tools for accessing Sejm and ELI APIs.
type SejmServer struct {
    server *server.MCPServer
    client *http.Client
}

// NewSejmServer creates a new instance of SejmServer with all tools registered.
func NewSejmServer() *SejmServer {
    // Implementation...
}
```

## Pull Request Process

### Before Submitting

1. **Update your fork**:
```bash
git fetch upstream
git checkout main
git merge upstream/main
```

2. **Create a feature branch**:
```bash
git checkout -b feature/your-feature-name
```

3. **Make your changes** following the guidelines above

4. **Test thoroughly**:
```bash
go test ./...
go build -o sejm-mcp ./cmd/sejm-mcp
```

5. **Update documentation** as needed

### Commit Messages

Use clear, descriptive commit messages:

```
feat: add sejm_get_legislative_processes tool

- Implement new tool for retrieving legislative process data
- Add comprehensive parameter validation
- Include usage examples in documentation
- Add unit tests for all scenarios

Fixes #123
```

Commit message format:
- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `test:` for adding tests
- `refactor:` for code refactoring
- `chore:` for maintenance tasks

### Pull Request Template

When submitting a PR, include:

```markdown
## Description
Brief description of changes made.

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Code refactoring

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass (if applicable)
- [ ] Manual testing completed

## Documentation
- [ ] API reference updated
- [ ] Examples added/updated
- [ ] Code comments added

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Breaking changes documented
```

### Review Process

1. **Automated checks**: Ensure all CI checks pass
2. **Code review**: Address reviewer feedback promptly
3. **Testing**: Verify functionality works as expected
4. **Documentation**: Ensure docs are complete and accurate

## Release Process

Releases follow semantic versioning:

- `MAJOR.MINOR.PATCH`
- `MAJOR`: Breaking changes
- `MINOR`: New features (backward compatible)
- `PATCH`: Bug fixes (backward compatible)

## Getting Help

- **Issues**: Open a GitHub issue for bugs or feature requests
- **Discussions**: Use GitHub Discussions for questions
- **Email**: Contact maintainers for security issues

## Recognition

Contributors will be recognized in:
- CONTRIBUTORS.md file
- Release notes for significant contributions
- GitHub contributor stats

## License

By contributing, you agree that your contributions will be licensed under the same MIT License that covers the project.