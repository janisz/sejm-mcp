# Sejm MCP Server Configuration Examples

This directory contains example configurations and usage patterns for the Sejm MCP Server.

## Files

### Configuration Examples

- **`claude-desktop-config.json`** - Basic Claude Desktop configuration for binary installation
- **`claude-desktop-docker-config.json`** - Claude Desktop configuration for Docker-based setup
- **`usage-examples.md`** - Comprehensive usage examples and query patterns

## Quick Start Configurations

### For Binary Installation

1. **Build or Download Binary**:
   ```bash
   # From source
   git clone https://github.com/janisz/sejm-mcp.git
   cd sejm-mcp
   make build
   sudo cp sejm-mcp /usr/local/bin/

   # Or download from releases
   wget https://github.com/janisz/sejm-mcp/releases/latest/download/sejm-mcp-linux-amd64
   chmod +x sejm-mcp-linux-amd64
   sudo mv sejm-mcp-linux-amd64 /usr/local/bin/sejm-mcp
   ```

2. **Configure Claude Desktop** (copy to your Claude config file):
   ```json
   {
     "mcpServers": {
       "sejm-mcp": {
         "command": "/usr/local/bin/sejm-mcp",
         "args": [],
         "env": {}
       }
     }
   }
   ```

3. **Restart Claude Desktop** and start using Polish Parliament data!

### For Docker Installation

1. **Build Docker Image**:
   ```bash
   git clone https://github.com/janisz/sejm-mcp.git
   cd sejm-mcp
   make docker-build
   ```

2. **Configure Claude Desktop** (copy to your Claude config file):
   ```json
   {
     "mcpServers": {
       "sejm-mcp": {
         "command": "docker",
         "args": [
           "run",
           "--rm",
           "--interactive",
           "--stdin-open",
           "sejm-mcp:latest"
         ],
         "env": {}
       }
     }
   }
   ```

3. **Restart Claude Desktop** and enjoy the minimal 9MB Docker container!

## Example Queries to Try

Once configured, try these example queries in Claude Desktop:

### Parliamentary Data
```
"Show me all current Members of Parliament from the Finance Committee"
```

```
"Get details about MP with ID 1"
```

```
"What are the most recent voting records from parliamentary term 10?"
```

### Legal Research
```
"Search for Polish laws related to data protection"
```

```
"Get the full text of the Polish Constitution"
```

```
"Show me legal acts published by Monitor Polski in 2023"
```

## Configuration Tips

### For Development
Add debug environment variables:
```json
{
  "mcpServers": {
    "sejm-mcp": {
      "command": "/path/to/sejm-mcp",
      "args": [],
      "env": {
        "DEBUG": "true",
        "LOG_LEVEL": "debug"
      }
    }
  }
}
```

### For Production
Use Docker with resource limits:
```json
{
  "mcpServers": {
    "sejm-mcp": {
      "command": "docker",
      "args": [
        "run",
        "--rm",
        "--interactive",
        "--stdin-open",
        "--memory=64m",
        "--cpus=0.2",
        "sejm-mcp:latest"
      ],
      "env": {}
    }
  }
}
```

### Multiple Instances
For high availability, configure multiple server instances:
```json
{
  "mcpServers": {
    "sejm-mcp-primary": {
      "command": "/usr/local/bin/sejm-mcp",
      "args": [],
      "env": {}
    },
    "sejm-mcp-backup": {
      "command": "docker",
      "args": ["run", "--rm", "--interactive", "--stdin-open", "sejm-mcp:latest"],
      "env": {}
    }
  }
}
```

## Troubleshooting

### Common Issues

1. **"Command not found"** - Verify binary is in PATH or use absolute path
2. **Docker permission errors** - Add user to docker group: `sudo usermod -aG docker $USER`
3. **API timeouts** - Check internet connectivity to `api.sejm.gov.pl`
4. **JSON syntax errors** - Validate configuration file with `jq .` or an online JSON validator

### Testing Your Configuration

Test API connectivity:
```bash
# Test Sejm API
curl -s "https://api.sejm.gov.pl/sejm/term10/MP" | head -100

# Test ELI API
curl -s "https://api.sejm.gov.pl/eli/acts" | head -100
```

Test binary directly:
```bash
sejm-mcp --help
```

Test Docker container:
```bash
docker run --rm sejm-mcp:latest --help
```

## Advanced Usage

See `usage-examples.md` for detailed examples of:
- Complex parliamentary analysis
- Legal research workflows
- Data extraction and processing
- Academic and journalistic use cases
- Integration with other tools

## Support

- 📖 **Full Setup Guide**: See `../SETUP.md`
- 🐛 **Issues**: https://github.com/janisz/sejm-mcp/issues
- 📚 **API Documentation**: https://api.sejm.gov.pl/
- 🔧 **MCP Protocol**: https://spec.modelcontextprotocol.io/