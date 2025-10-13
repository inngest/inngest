# Inngest Dev Server MCP Integration

The Inngest dev server now includes a Model Context Protocol (MCP) server accessible at `/mcp` when the dev server is running.

## What is MCP?

Model Context Protocol (MCP) is a standard for connecting AI assistants to external tools and data sources. The Inngest MCP integration allows AI assistants like Claude to interact with your Inngest functions directly.

## Available Tools

### 1. `invoke_function`
Invoke an Inngest function in the dev server.

**Parameters:**
- `functionSlug` (string, required): The function slug/identifier to invoke
- `data` (JSON object, optional): The event data to send to the function
- `eventIdSeed` (string, optional): Seed for generating deterministic event IDs

**Example:**
```json
{
  "functionSlug": "my-function",
  "data": {
    "user": {"id": "123", "name": "Test User"},
    "action": "test-action"
  },
  "eventIdSeed": "test-seed-123"
}
```

### 2. `list_functions`
List all registered functions in the dev server.

**Parameters:** None

**Returns:** Array of functions with their IDs, names, slugs, and triggers.

## Usage

### With Claude Desktop

Add this to your Claude Desktop MCP configuration:

```json
{
  "mcpServers": {
    "inngest-dev": {
      "command": "curl",
      "args": [
        "-X", "POST",
        "http://127.0.0.1:8288/mcp",
        "-H", "Content-Type: application/json",
        "-d", "@-"
      ]
    }
  }
}
```

### Starting the Dev Server

1. Start the Inngest dev server as usual:
   ```bash
   npx inngest-cli dev
   ```

2. The MCP server will be automatically available at `http://127.0.0.1:8288/mcp`

3. Configure your MCP client to connect to this endpoint

## MCP Transport

This implementation uses the Streamable HTTP transport as defined in the MCP specification. The server supports:

- HTTP POST requests for sending messages
- JSON responses (not server-sent events)
- Stateless operation mode
- Multiple concurrent sessions

## Development

The MCP integration is implemented in:
- `pkg/devserver/mcp.go` - Main MCP server implementation
- `pkg/devserver/devserver.go` - Integration into dev server startup

To add new tools:
1. Define the argument struct with JSON schema tags
2. Create a handler function following the MCP pattern
3. Register the tool in `createMCPServer()`