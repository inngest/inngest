import { useState, useMemo } from 'react';

export default function MCPPage() {
  const [copiedText, setCopiedText] = useState<string | null>(null);

  // Get dynamic MCP endpoint based on current location
  const mcpEndpoint = useMemo(() => {
    if (typeof window === 'undefined') return 'http://127.0.0.1:8288/mcp';
    const hostname = window.location.hostname || '127.0.0.1';
    const port = window.location.port || '8288';
    return `http://${hostname}:${port}/mcp`;
  }, []);

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedText(id);
    setTimeout(() => setCopiedText(null), 2000);
  };

  // Generate config strings with dynamic endpoint
  const claudeCodeCommand = `claude mcp add --transport http inngest-dev ${mcpEndpoint}`;
  const cursorConfig = `{
  "mcpServers": {
    "inngest-dev": {
      "url": "${mcpEndpoint}"
    }
  }
}`;
  const claudeDesktopConfig = `{
  "mcpServers": {
    "inngest-dev": {
      "command": "curl",
      "args": [
        "-X", "POST",
        "${mcpEndpoint}",
        "-H", "Content-Type: application/json",
        "-d", "@-"
      ]
    }
  }
}`;

  return (
    <div className="bg-canvasBase min-h-screen">
      <div className="border-subtle border-b bg-canvasBase sticky top-0 z-10">
        <div className="mx-auto max-w-5xl px-8 py-6">
          <h1 className="text-basis text-3xl font-semibold">MCP Setup</h1>
          <p className="text-muted mt-2 text-base">
            Connect your AI assistant to this dev server using the Model Context
            Protocol (MCP)
          </p>
        </div>
      </div>

      <div className="mx-auto max-w-5xl px-8 py-8">
        <section className="mb-12">
          <h2 className="text-basis mb-4 text-2xl font-semibold">
            MCP Endpoint
          </h2>
          <div className="bg-canvasSubtle border-subtle rounded border p-4">
            <div className="flex items-center justify-between gap-4">
              <code className="text-basis flex-1 break-all text-lg font-mono">
                {mcpEndpoint}
              </code>
              <button
                onClick={() => copyToClipboard(mcpEndpoint, 'endpoint')}
                className="bg-canvasBase border-subtle hover:bg-canvasMuted shrink-0 rounded border px-4 py-2 text-sm font-medium transition-colors"
              >
                {copiedText === 'endpoint' ? 'Copied!' : 'Copy'}
              </button>
            </div>
            <p className="text-muted mt-3 text-sm">
              Use this URL to connect your AI assistant to the Inngest dev
              server.
            </p>
          </div>
        </section>

        <section className="mb-12">
          <h2 className="text-basis mb-4 text-2xl font-semibold">
            Connect Your AI Tool
          </h2>

          <div className="mb-4">
            <h3 className="text-basis mb-2 text-lg font-medium">Claude Code</h3>
            <div className="bg-canvasSubtle border-subtle relative rounded border">
              <div className="border-subtle flex items-center justify-between border-b px-4 py-2">
                <span className="text-muted text-xs font-medium">bash</span>
                <button
                  onClick={() => copyToClipboard(claudeCodeCommand, 'claude')}
                  className="text-muted hover:text-basis text-xs"
                >
                  {copiedText === 'claude' ? 'Copied!' : 'Copy'}
                </button>
              </div>
              <pre className="overflow-x-auto p-4">
                <code className="text-basis text-sm">{claudeCodeCommand}</code>
              </pre>
            </div>
          </div>

          <div className="mb-4">
            <h3 className="text-basis mb-2 text-lg font-medium">Cursor</h3>
            <p className="text-muted mb-2 text-sm">
              Add to{' '}
              <code className="bg-canvasSubtle rounded px-1.5 py-0.5">
                .cursor/mcp.json
              </code>
            </p>
            <div className="bg-canvasSubtle border-subtle relative rounded border">
              <div className="border-subtle flex items-center justify-between border-b px-4 py-2">
                <span className="text-muted text-xs font-medium">json</span>
                <button
                  onClick={() => copyToClipboard(cursorConfig, 'cursor')}
                  className="text-muted hover:text-basis text-xs"
                >
                  {copiedText === 'cursor' ? 'Copied!' : 'Copy'}
                </button>
              </div>
              <pre className="overflow-x-auto p-4">
                <code className="text-basis text-sm">{cursorConfig}</code>
              </pre>
            </div>
          </div>

          <div className="mb-4">
            <h3 className="text-basis mb-2 text-lg font-medium">
              Claude Desktop
            </h3>
            <p className="text-muted mb-2 text-sm">
              Add to Claude Desktop configuration
            </p>
            <div className="bg-canvasSubtle border-subtle relative rounded border">
              <div className="border-subtle flex items-center justify-between border-b px-4 py-2">
                <span className="text-muted text-xs font-medium">json</span>
                <button
                  onClick={() =>
                    copyToClipboard(claudeDesktopConfig, 'desktop')
                  }
                  className="text-muted hover:text-basis text-xs"
                >
                  {copiedText === 'desktop' ? 'Copied!' : 'Copy'}
                </button>
              </div>
              <pre className="overflow-x-auto p-4">
                <code className="text-basis text-sm">
                  {claudeDesktopConfig}
                </code>
              </pre>
            </div>
          </div>
        </section>

        <section className="mb-12">
          <h2 className="text-basis mb-4 text-2xl font-semibold">
            Start Building with AI
          </h2>
          <p className="text-basis mb-3 text-base">
            Once connected, you can ask your AI assistant to:
          </p>
          <div className="space-y-2">
            <div className="bg-canvasSubtle border-subtle rounded border p-3">
              <code className="text-basis text-sm">
                List all my Inngest functions and their triggers
              </code>
            </div>
            <div className="bg-canvasSubtle border-subtle rounded border p-3">
              <code className="text-basis text-sm">
                Send a test event to trigger the user signup workflow
              </code>
            </div>
            <div className="bg-canvasSubtle border-subtle rounded border p-3">
              <code className="text-basis text-sm">
                Monitor the function run and show me any errors
              </code>
            </div>
            <div className="bg-canvasSubtle border-subtle rounded border p-3">
              <code className="text-basis text-sm">
                Search the docs for rate limiting examples
              </code>
            </div>
          </div>
        </section>

        <section className="mb-12">
          <h2 className="text-basis mb-4 text-2xl font-semibold">
            Available MCP Tools
          </h2>
          <p className="text-basis mb-6 text-base">
            The Inngest MCP server provides 8 powerful tools organized into
            three categories:
          </p>

          <div className="mb-8">
            <h3 className="text-basis mb-3 text-xl font-semibold">
              Event Management Tools
            </h3>

            <div className="bg-canvasSubtle border-subtle mb-4 rounded border p-4">
              <h4 className="text-basis mb-2 text-lg font-medium">
                send_event
              </h4>
              <p className="text-basis mb-3 text-sm">
                Send an event to trigger functions and get immediate feedback on
                which runs were created.
              </p>
              <div className="mb-2">
                <span className="text-muted text-xs font-medium">
                  Parameters:
                </span>
                <ul className="text-basis ml-4 mt-1 list-disc text-sm">
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      name
                    </code>{' '}
                    (string, required): Event name (e.g., 'app/user.created')
                  </li>
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      data
                    </code>{' '}
                    (object, optional): Event payload data
                  </li>
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      user
                    </code>{' '}
                    (object, optional): User context information
                  </li>
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      eventIdSeed
                    </code>{' '}
                    (string, optional): Seed for deterministic event IDs
                  </li>
                </ul>
              </div>
            </div>

            <div className="bg-canvasSubtle border-subtle mb-4 rounded border p-4">
              <h4 className="text-basis mb-2 text-lg font-medium">
                list_functions
              </h4>
              <p className="text-basis text-sm">
                Discover all registered functions with their names, IDs, and
                trigger information.
              </p>
            </div>

            <div className="bg-canvasSubtle border-subtle mb-4 rounded border p-4">
              <h4 className="text-basis mb-2 text-lg font-medium">
                invoke_function
              </h4>
              <p className="text-basis mb-3 text-sm">
                Directly execute a function and wait for its complete result -
                perfect for testing specific functions.
              </p>
              <div className="mb-2">
                <span className="text-muted text-xs font-medium">
                  Parameters:
                </span>
                <ul className="text-basis ml-4 mt-1 list-disc text-sm">
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      functionId
                    </code>{' '}
                    (string, required): Function slug, ID, or name
                  </li>
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      data
                    </code>{' '}
                    (object, optional): Input data for the function
                  </li>
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      user
                    </code>{' '}
                    (object, optional): User context
                  </li>
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      timeout
                    </code>{' '}
                    (int, optional): Wait timeout in seconds (default: 30)
                  </li>
                </ul>
              </div>
            </div>
          </div>

          <div className="mb-8">
            <h3 className="text-basis mb-3 text-xl font-semibold">
              Execution Monitoring Tools
            </h3>

            <div className="bg-canvasSubtle border-subtle mb-4 rounded border p-4">
              <h4 className="text-basis mb-2 text-lg font-medium">
                get_run_status
              </h4>
              <p className="text-basis mb-3 text-sm">
                Get detailed information about a specific function run,
                including step-by-step execution details.
              </p>
              <div className="mb-2">
                <span className="text-muted text-xs font-medium">
                  Parameters:
                </span>
                <ul className="text-basis ml-4 mt-1 list-disc text-sm">
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      runId
                    </code>{' '}
                    (string, required): The run ID from send_event or logs
                  </li>
                </ul>
              </div>
            </div>

            <div className="bg-canvasSubtle border-subtle mb-4 rounded border p-4">
              <h4 className="text-basis mb-2 text-lg font-medium">
                poll_run_status
              </h4>
              <p className="text-basis mb-3 text-sm">
                Monitor multiple function runs until completion - essential for
                integration testing workflows.
              </p>
              <div className="mb-2">
                <span className="text-muted text-xs font-medium">
                  Parameters:
                </span>
                <ul className="text-basis ml-4 mt-1 list-disc text-sm">
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      runIds
                    </code>{' '}
                    (array, required): Array of run IDs to monitor
                  </li>
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      timeout
                    </code>{' '}
                    (int, optional): Total polling timeout in seconds (default:
                    30)
                  </li>
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      pollInterval
                    </code>{' '}
                    (int, optional): Milliseconds between polls (default: 1000)
                  </li>
                </ul>
              </div>
            </div>
          </div>

          <div className="mb-8">
            <h3 className="text-basis mb-3 text-xl font-semibold">
              Documentation Tools
            </h3>

            <div className="bg-canvasSubtle border-subtle mb-4 rounded border p-4">
              <h4 className="text-basis mb-2 text-lg font-medium">grep_docs</h4>
              <p className="text-basis mb-3 text-sm">
                Search through embedded Inngest documentation using pattern
                matching.
              </p>
              <div className="mb-2">
                <span className="text-muted text-xs font-medium">
                  Parameters:
                </span>
                <ul className="text-basis ml-4 mt-1 list-disc text-sm">
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      pattern
                    </code>{' '}
                    (string, required): Search pattern (regex supported)
                  </li>
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      limit
                    </code>{' '}
                    (int, optional): Maximum results (default: 10)
                  </li>
                </ul>
              </div>
            </div>

            <div className="bg-canvasSubtle border-subtle mb-4 rounded border p-4">
              <h4 className="text-basis mb-2 text-lg font-medium">read_doc</h4>
              <p className="text-basis mb-3 text-sm">
                Read the complete content of a specific documentation file.
              </p>
              <div className="mb-2">
                <span className="text-muted text-xs font-medium">
                  Parameters:
                </span>
                <ul className="text-basis ml-4 mt-1 list-disc text-sm">
                  <li>
                    <code className="bg-canvasBase rounded px-1 py-0.5">
                      path
                    </code>{' '}
                    (string, required): Document path relative to docs directory
                  </li>
                </ul>
              </div>
            </div>

            <div className="bg-canvasSubtle border-subtle mb-4 rounded border p-4">
              <h4 className="text-basis mb-2 text-lg font-medium">list_docs</h4>
              <p className="text-basis text-sm">
                Get an overview of all available documentation with category
                breakdown.
              </p>
            </div>
          </div>
        </section>

        <section className="mb-12">
          <h2 className="text-basis mb-4 text-2xl font-semibold">
            Best Practices
          </h2>

          <div className="mb-6">
            <h3 className="text-basis mb-3 text-xl font-semibold">
              Function Testing
            </h3>
            <ul className="text-basis ml-6 list-disc space-y-2 text-base">
              <li>
                <strong>Start simple:</strong> Test individual functions before
                complex workflows
              </li>
              <li>
                <strong>Use descriptive events:</strong> Clear event names help
                with debugging
              </li>
              <li>
                <strong>Monitor execution:</strong> Always check run status
                after triggering events
              </li>
              <li>
                <strong>Test error scenarios:</strong> Intentionally trigger
                failures to test error handling
              </li>
            </ul>
          </div>

          <div className="mb-6">
            <h3 className="text-basis mb-3 text-xl font-semibold">
              Debugging Workflows
            </h3>
            <ul className="text-basis ml-6 list-disc space-y-2 text-base">
              <li>
                <strong>Check step details:</strong> Use{' '}
                <code className="bg-canvasSubtle rounded px-1.5 py-0.5">
                  get_run_status
                </code>{' '}
                to see step-by-step execution
              </li>
              <li>
                <strong>Review error context:</strong> Error messages include
                stack traces and context
              </li>
              <li>
                <strong>Verify data flow:</strong> Check inputs and outputs at
                each step
              </li>
              <li>
                <strong>Use polling for async:</strong> Monitor long-running
                workflows with{' '}
                <code className="bg-canvasSubtle rounded px-1.5 py-0.5">
                  poll_run_status
                </code>
              </li>
            </ul>
          </div>

          <div className="mb-6">
            <h3 className="text-basis mb-3 text-xl font-semibold">
              Documentation Usage
            </h3>
            <ul className="text-basis ml-6 list-disc space-y-2 text-base">
              <li>
                <strong>Search before building:</strong> Use{' '}
                <code className="bg-canvasSubtle rounded px-1.5 py-0.5">
                  grep_docs
                </code>{' '}
                to find relevant examples
              </li>
              <li>
                <strong>Reference patterns:</strong> Look for similar use cases
                in the documentation
              </li>
              <li>
                <strong>Cross-reference APIs:</strong> Use{' '}
                <code className="bg-canvasSubtle rounded px-1.5 py-0.5">
                  read_doc
                </code>{' '}
                for complete API documentation
              </li>
            </ul>
          </div>
        </section>

        <section className="mb-12">
          <h2 className="text-basis mb-4 text-2xl font-semibold">
            Troubleshooting
          </h2>

          <div className="bg-canvasSubtle border-subtle mb-4 rounded border p-4">
            <h3 className="text-basis mb-2 text-lg font-semibold">
              MCP server not found
            </h3>
            <ul className="text-basis ml-4 list-disc space-y-1 text-sm">
              <li>Restart the dev server if the endpoint is unresponsive</li>
              <li>
                Verify the MCP endpoint is accessible:{' '}
                <code className="bg-canvasBase rounded px-1 py-0.5">
                  curl {mcpEndpoint}
                </code>
              </li>
              <li>Check your MCP client configuration</li>
            </ul>
          </div>

          <div className="bg-canvasSubtle border-subtle mb-4 rounded border p-4">
            <h3 className="text-basis mb-2 text-lg font-semibold">
              Functions not listed
            </h3>
            <ul className="text-basis ml-4 list-disc space-y-1 text-sm">
              <li>
                Confirm functions are properly registered with the dev server
              </li>
              <li>Check the dev server logs for registration errors</li>
              <li>Verify your app is correctly synced with the dev server</li>
            </ul>
          </div>

          <div className="bg-canvasSubtle border-subtle mb-4 rounded border p-4">
            <h3 className="text-basis mb-2 text-lg font-semibold">
              Runs not found after sending events
            </h3>
            <ul className="text-basis ml-4 list-disc space-y-1 text-sm">
              <li>
                Wait a moment for event processing (500ms delay is built-in)
              </li>
              <li>
                Check if the event name matches your function triggers exactly
              </li>
              <li>Verify the function is actually registered and listening</li>
            </ul>
          </div>

          <div className="bg-canvasSubtle border-subtle mb-4 rounded border p-4">
            <h3 className="text-basis mb-2 text-lg font-semibold">
              Polling timeouts
            </h3>
            <ul className="text-basis ml-4 list-disc space-y-1 text-sm">
              <li>Increase timeout values for long-running functions</li>
              <li>Check function logs for hanging operations</li>
              <li>
                Verify functions are actually completing vs. hanging
                indefinitely
              </li>
            </ul>
          </div>
        </section>

        <section className="mb-12">
          <h2 className="text-basis mb-4 text-2xl font-semibold">Resources</h2>
          <div className="grid gap-4 md:grid-cols-2">
            <a
              href="https://modelcontextprotocol.io/introduction"
              target="_blank"
              rel="noopener noreferrer"
              className="bg-canvasSubtle border-subtle hover:border-emphasis block rounded border p-4 transition-colors"
            >
              <h3 className="text-basis mb-2 text-base font-semibold">
                Model Context Protocol Specification
              </h3>
              <p className="text-muted text-sm">
                Learn more about the MCP standard and how it enables AI tool
                integrations.
              </p>
            </a>
            <a
              href="https://www.inngest.com/docs/dev-server"
              target="_blank"
              rel="noopener noreferrer"
              className="bg-canvasSubtle border-subtle hover:border-emphasis block rounded border p-4 transition-colors"
            >
              <h3 className="text-basis mb-2 text-base font-semibold">
                Inngest Dev Server Documentation
              </h3>
              <p className="text-muted text-sm">
                Comprehensive guide to the Inngest dev server and local
                development.
              </p>
            </a>
            <a
              href="https://www.inngest.com/docs/examples/ai-agents-and-rag"
              target="_blank"
              rel="noopener noreferrer"
              className="bg-canvasSubtle border-subtle hover:border-emphasis block rounded border p-4 transition-colors"
            >
              <h3 className="text-basis mb-2 text-base font-semibold">
                AI Agents and RAG Examples
              </h3>
              <p className="text-muted text-sm">
                Examples of building AI-powered applications with Inngest
                functions.
              </p>
            </a>
            <a
              href="https://www.inngest.com/blog/context-engineering-is-software-engineering-for-llms"
              target="_blank"
              rel="noopener noreferrer"
              className="bg-canvasSubtle border-subtle hover:border-emphasis block rounded border p-4 transition-colors"
            >
              <h3 className="text-basis mb-2 text-base font-semibold">
                Context Engineering Blog Post
              </h3>
              <p className="text-muted text-sm">
                Understanding how tools like MCP enable better AI development
                workflows.
              </p>
            </a>
          </div>
        </section>
      </div>
    </div>
  );
}
