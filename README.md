# Sejm MCP Server

An MCP (Model Context Protocol) server that provides comprehensive access to Polish Parliament (Sejm) APIs and European Legislation Identifier (ELI) database for legal documents.

## Overview

This server enables AI assistants and applications to interact with Polish parliamentary data and legal documents through a standardized MCP interface. It provides access to information about MPs, committees, votings, interpellations, and the complete Polish legal acts database.

## Features

### 🏛️ Sejm API Tools
Access real-time parliamentary data from the Polish Sejm:

- **sejm_get_mps**: Retrieve lists of Members of Parliament
- **sejm_get_mp_details**: Get detailed MP profiles and statistics
- **sejm_get_committees**: Access parliamentary committee information
- **sejm_search_votings**: Search and analyze voting records
- **sejm_get_interpellations**: Browse parliamentary questions and answers

### ⚖️ ELI (European Legislation Identifier) API Tools
Search and retrieve Polish legal documents:

- **eli_search_acts**: Advanced search across legal acts database
- **eli_get_act_details**: Retrieve comprehensive act metadata
- **eli_get_act_text**: Download full legal text (HTML/PDF formats)
- **eli_get_act_references**: Explore legal document relationships
- **eli_get_publishers**: List available legal publishers

## Installation

### Prerequisites
- Go 1.21 or later
- Git

### Quick Start

```bash
# Clone the repository
git clone https://github.com/janisz/sejm-mcp.git
cd sejm-mcp

# Build the server
go build -o sejm-mcp ./cmd/sejm-mcp

# Run the server
./sejm-mcp
```

### Using with MCP Clients

The server implements the Model Context Protocol and can be integrated with any MCP-compatible client (Claude Desktop, VS Code extensions, etc.).

## Tool Documentation

### Sejm API Tools

#### `sejm_get_mps`
Get a list of Members of Parliament for a specific term.

**Parameters:**
- `term` (optional): Parliamentary term number (1-10, default: 10)

**Example:**
```json
{
  "tool": "sejm_get_mps",
  "arguments": {
    "term": "10"
  }
}
```

**Returns:** Array of MP objects with personal details, club affiliation, district information, and activity status.

---

#### `sejm_get_mp_details`
Retrieve detailed information about a specific Member of Parliament.

**Parameters:**
- `term` (optional): Parliamentary term (1-10, default: 10)
- `mp_id` (required): MP identification number

**Example:**
```json
{
  "tool": "sejm_get_mp_details",
  "arguments": {
    "term": "10",
    "mp_id": "123"
  }
}
```

**Returns:** Comprehensive MP profile including biography, voting statistics, committee memberships, and contact information.

---

#### `sejm_get_committees`
List all parliamentary committees for a specific term.

**Parameters:**
- `term` (optional): Parliamentary term (1-10, default: 10)

**Example:**
```json
{
  "tool": "sejm_get_committees",
  "arguments": {
    "term": "10"
  }
}
```

**Returns:** Array of committee objects with names, codes, members, scope of work, and contact details.

---

#### `sejm_search_votings`
Search parliamentary voting records with filtering options.

**Parameters:**
- `term` (optional): Parliamentary term (1-10, default: 10)
- `sitting` (optional): Specific sitting number
- `limit` (optional): Maximum results (default: 50)

**Example:**
```json
{
  "tool": "sejm_search_votings",
  "arguments": {
    "term": "10",
    "sitting": "1",
    "limit": "25"
  }
}
```

**Returns:** Array of voting records with dates, topics, vote counts, and results.

---

#### `sejm_get_interpellations`
Retrieve parliamentary interpellations (formal questions to government).

**Parameters:**
- `term` (optional): Parliamentary term (1-10, default: 10)
- `limit` (optional): Maximum results (default: 50)

**Example:**
```json
{
  "tool": "sejm_get_interpellations",
  "arguments": {
    "term": "10",
    "limit": "20"
  }
}
```

**Returns:** Array of interpellation objects with questions, recipients, dates, and government responses.

### ELI API Tools

#### `eli_search_acts`
Search the Polish legal acts database with advanced filtering.

**Parameters:**
- `title` (optional): Search keywords in act titles
- `publisher` (optional): Publisher code (e.g., "DU" for Journal of Laws)
- `year` (optional): Publication year
- `type` (optional): Document type
- `limit` (optional): Maximum results (default: 50)

**Example:**
```json
{
  "tool": "eli_search_acts",
  "arguments": {
    "title": "konstytucja",
    "publisher": "DU",
    "year": "1997",
    "limit": "10"
  }
}
```

**Returns:** Search results with act summaries, ELI identifiers, and publication details.

---

#### `eli_get_act_details`
Get comprehensive metadata for a specific legal act.

**Parameters:**
- `publisher` (required): Publisher code
- `year` (required): Publication year
- `position` (required): Position number in journal

**Example:**
```json
{
  "tool": "eli_get_act_details",
  "arguments": {
    "publisher": "DU",
    "year": "1997",
    "position": "78"
  }
}
```

**Returns:** Complete act metadata including title, dates, status, keywords, and legal relationships.

---

#### `eli_get_act_text`
Download the full text of a legal act in HTML or PDF format.

**Parameters:**
- `publisher` (required): Publisher code
- `year` (required): Publication year
- `position` (required): Position number
- `format` (optional): "html" or "pdf" (default: html)

**Example:**
```json
{
  "tool": "eli_get_act_text",
  "arguments": {
    "publisher": "DU",
    "year": "1997",
    "position": "78",
    "format": "html"
  }
}
```

**Returns:** Full legal text in requested format, suitable for analysis or display.

---

#### `eli_get_act_references`
Explore legal relationships between acts (citations, amendments, etc.).

**Parameters:**
- `publisher` (required): Publisher code
- `year` (required): Publication year
- `position` (required): Position number

**Example:**
```json
{
  "tool": "eli_get_act_references",
  "arguments": {
    "publisher": "DU",
    "year": "1997",
    "position": "78"
  }
}
```

**Returns:** Array of related legal documents with relationship types and descriptions.

---

#### `eli_get_publishers`
List all available legal document publishers in the ELI database.

**Parameters:** None

**Example:**
```json
{
  "tool": "eli_get_publishers",
  "arguments": {}
}
```

**Returns:** Array of publisher objects with codes, names, and descriptions.

## Use Cases

### Research & Analysis
- **Political Science**: Analyze voting patterns, committee compositions, MP activity
- **Legal Research**: Search legislation, track legal changes, find cited documents
- **Journalism**: Access up-to-date parliamentary proceedings and legal developments
- **Academic**: Study Polish political system and legal framework

### AI Integration
- **Legal AI**: Enable AI assistants to answer questions about Polish law
- **Political Chatbots**: Provide real-time information about MPs and parliamentary activities
- **Research Tools**: Automate data collection for political and legal analysis
- **Compliance Systems**: Monitor legal changes and regulatory updates

## Data Sources

This server provides access to official Polish government APIs:

- **Sejm API**: https://api.sejm.gov.pl/sejm.html
  - Real-time parliamentary data
  - Official MP profiles and voting records
  - Committee schedules and compositions

- **ELI API**: https://api.sejm.gov.pl/eli_pl.html
  - Complete Polish legal acts database
  - European Legislation Identifier compliance
  - Full-text search capabilities

## Development

### Architecture

The server uses a clean, maintainable architecture:

```
├── cmd/sejm-mcp/          # Main application entry point
├── internal/server/       # MCP server implementation
│   ├── server.go         # Core server and HTTP client
│   ├── sejm_tools.go     # Sejm API tool implementations
│   └── eli_tools.go      # ELI API tool implementations
├── pkg/
│   ├── sejm/             # Auto-generated Sejm API types
│   └── eli/              # Auto-generated ELI API types
├── *-codegen.yaml        # OpenAPI code generation configs
├── *.json                # OpenAPI specifications (downloaded)
└── README.md
```

### Type Generation

The project automatically generates Go types from official OpenAPI specifications:

```bash
# Install required tools
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

# Download latest OpenAPI specs
curl -s https://api.sejm.gov.pl/sejm/openapi/ -o sejm-openapi.json
curl -s https://api.sejm.gov.pl/eli/openapi/ -o eli-openapi.json

# Convert YAML to JSON format
yq eval -j sejm-openapi.json > sejm-openapi-converted.json
yq eval -j eli-openapi.json > eli-openapi-converted.json

# Generate Go types from OpenAPI specs
oapi-codegen -config sejm-codegen.yaml sejm-openapi-converted.json
oapi-codegen -config eli-codegen.yaml eli-openapi-converted.json

# Rebuild the server
go build -o sejm-mcp ./cmd/sejm-mcp
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Ensure all types are generated from OpenAPI specs
4. Add tests for new functionality
5. Update documentation
6. Submit a pull request

### Testing

The project has a comprehensive test suite with different testing levels:

```bash
# Run all tests (recommended before committing)
make test

# Run unit tests only (fast, no network required)
make test-unit

# Run API connectivity smoke test (quick network check)
make test-smoke

# Run integration tests (requires network access)
make test-integration

# Run with coverage report
make test-coverage

# Check API connectivity manually
make check-apis
```

#### Test Architecture

- **Unit Tests**: Use mocked data, run offline, focus on logic validation
- **Smoke Tests**: Single connectivity test to verify API accessibility
- **Integration Tests**: End-to-end tests requiring real API access
- **Mock Servers**: `httptest.NewServer` for controlled HTTP testing

#### Performance

- Unit tests: ~0.01s (instant with mocked data)
- Smoke tests: ~0.1s (quick connectivity check)
- Full test suite: ~6-8s (includes mock HTTP servers)

## Troubleshooting

### Common Issues

**Build Errors**: Usually caused by duplicate fields in generated types
- Re-run type generation: `oapi-codegen -config sejm-codegen.yaml sejm-openapi-converted.json`
- Check for API specification changes

**API Rate Limits**: The official APIs may have rate limiting
- Implement request throttling if needed
- Cache responses for frequently accessed data

**Network Timeouts**: API responses can be slow for large datasets
- Adjust HTTP client timeout in server configuration
- Use pagination parameters to limit response size

**Test Failures**:
- Unit tests failing: Check for recent code changes, unit tests use mocked data
- Smoke tests failing: Verify internet connectivity, APIs may be temporarily unavailable
- Integration tests failing: Check API status and network connectivity

### Performance Tips

- Use `limit` parameters to control response sizes
- Cache frequently accessed reference data (committees, publishers)
- Implement request deduplication for repeated queries

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Polish Parliament (Sejm) for providing open APIs
- MCP protocol developers for standardized AI tool integration
- OpenAPI Initiative for specification standards