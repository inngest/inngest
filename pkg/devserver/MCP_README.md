# Inngest Dev Server MCP Integration

The Inngest dev server includes a comprehensive Model Context Protocol (MCP) server accessible at `/mcp` when the dev server is running.

## What is MCP?

Model Context Protocol (MCP) is a standard for connecting AI assistants to external tools and data sources. The Inngest MCP integration provides AI assistants like Claude with complete access to your Inngest functions, run monitoring, and embedded documentation.

## Available Tools

### Event Management Tools

#### 1. `send_event`
Send an event to the Inngest dev server which will trigger any functions listening to that event.

**Parameters:**
- `name` (string, required): The event name (e.g., 'app/user.created', 'test/hello.world')
- `data` (JSON object, optional): The event data payload
- `user` (JSON object, optional): User context information
- `eventIdSeed` (string, optional): Seed for deterministic event ID generation

**Returns:** Event ID and run IDs of triggered functions

#### 2. `list_functions`
List all registered functions in the dev server.

**Parameters:** None

**Returns:** Array of functions with their IDs, names, slugs, and triggers.

#### 3. `invoke_function`
Directly invoke a specific function and wait for its result. Unlike send_event (fire-and-forget), this waits for completion and returns the function's actual output data.

**Parameters:**
- `functionId` (string, required): Function slug, ID, or name to invoke
- `data` (JSON object, optional): Function input data
- `user` (JSON object, optional): User context information
- `timeout` (int, optional): Seconds to wait for completion (default: 30)

**Returns:** Function execution result with output data and status

### Execution Monitoring Tools

#### 4. `get_run_status`
Get detailed status and trace information for a specific function run.

**Parameters:**
- `runId` (string, required): The run ID returned from send_event or found in logs

**Returns:** Detailed run information including status, steps, outputs, and errors

#### 5. `poll_run_status`
Poll multiple function runs until they complete or timeout. Essential for integration testing workflows.

**Parameters:**
- `runIds` (array of strings, required): Run IDs to monitor
- `timeout` (int, optional): Total seconds to poll (default: 30)
- `pollInterval` (int, optional): Milliseconds between polls (default: 1000)

**Returns:** Aggregated status for all runs with completion summary

### Documentation Tools

#### 6. `grep_docs`
Search embedded documentation using pattern matching. All Inngest documentation is bundled in the CLI binary for offline access.

**Parameters:**
- `pattern` (string, required): The search pattern (regex supported)
- `limit` (int, optional): Maximum results to return (default: 10)

**Returns:** Matching documentation lines with file paths and line numbers

#### 7. `read_doc`
Read the full content of a specific documentation file from embedded docs.

**Parameters:**
- `path` (string, required): The doc file path relative to docs directory

**Returns:** Complete file content with metadata

#### 8. `list_docs`
List all available documentation categories and their document counts.

**Parameters:** None

**Returns:** Documentation overview with categories, SDKs, and total counts

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