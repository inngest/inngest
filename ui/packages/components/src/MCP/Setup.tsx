import { useEffect, useRef, useState, type ReactNode } from 'react';

type JSONSchema = {
  properties?: Record<string, JSONSchemaProperty>;
  required?: string[];
};

type JSONSchemaProperty = {
  description?: string;
  enum?: string[];
  items?: { type?: string };
  type?: string;
};

type MCPTool = {
  description?: string;
  inputSchema: JSONSchema;
  name: string;
  title?: string;
};

type MCPSetupProps = {
  bearerTokenEnvVar?: string;
  endpoint: string;
  getAccessToken?: () => Promise<string | null>;
  headers?: Record<string, string>;
  isDevServer?: boolean;
};

type MCPRequest = {
  id?: number;
  method: string;
  params?: Record<string, unknown>;
};

type ToolsListResponse = {
  error?: { message?: string };
  result?: { tools?: MCPTool[] };
};

const emptyHeaders: Record<string, string> = {};

const devServerExamples = [
  'List my registered Inngest functions and their triggers',
  'Inspect recent function runs and their traces',
  'Inspect a failed function run and explain the error',
  'Search the Inngest docs for rate limiting examples',
];

const cloudExamples = [
  'List my Inngest environments',
  'List my registered Inngest functions and their triggers',
  'Inspect a failed function run and explain the error',
];

const resources = [
  {
    description: 'Learn about MCP transports, tools, and client capabilities.',
    href: 'https://modelcontextprotocol.io/introduction',
    title: 'Model Context Protocol',
  },
  {
    description: 'Learn how to run and configure Inngest locally.',
    href: 'https://www.inngest.com/docs/dev-server',
    title: 'Inngest Dev Server',
  },
  {
    description: 'See patterns for durable AI agents and retrieval workflows.',
    href: 'https://www.inngest.com/docs/examples/ai-agents-and-rag',
    title: 'AI Agents and RAG Examples',
  },
  {
    description: 'Learn how tools and context fit into reliable AI applications.',
    href: 'https://www.inngest.com/blog/context-engineering-is-software-engineering-for-llms',
    title: 'Context Engineering',
  },
  {
    description:
      'Configure MCP servers for Codex CLI, ChatGPT Desktop, and the Codex IDE extension.',
    href: 'https://learn.chatgpt.com/docs/extend/mcp',
    title: 'Codex MCP Setup',
  },
  {
    description: 'Understand local and remote MCP connections in Claude Desktop.',
    href: 'https://support.claude.com/en/articles/11175166-get-started-with-custom-connectors-using-remote-mcp',
    title: 'Claude Desktop Connectors',
  },
];

const toolType = (schema: JSONSchemaProperty) => {
  if (schema.type === 'array') {
    return `${schema.items?.type ?? 'value'}[]`;
  }
  return schema.enum ? 'enum' : schema.type ?? 'value';
};

const createConfigs = (endpoint: string, isDevServer: boolean, bearerTokenEnvVar?: string) => {
  const serverName = isDevServer ? 'inngest-dev' : 'inngest-cloud';
  const authorizationHeader = bearerTokenEnvVar
    ? ` --header "Authorization: Bearer $${bearerTokenEnvVar}"`
    : '';
  const bearerTokenOption = bearerTokenEnvVar ? ` --bearer-token-env-var ${bearerTokenEnvVar}` : '';

  return [
    {
      id: 'claude',
      label: 'Claude Code',
      language: 'bash',
      value: `claude mcp add --transport http ${serverName} ${endpoint}${authorizationHeader}`,
    },
    {
      id: 'codex',
      label: 'Codex CLI',
      language: 'bash',
      value: `codex mcp add ${serverName} --url ${endpoint}${bearerTokenOption}`,
    },
    {
      id: 'cursor',
      label: 'Cursor',
      language: 'json',
      value: JSON.stringify(
        {
          mcpServers: {
            [serverName]: {
              url: endpoint,
              ...(bearerTokenEnvVar && {
                headers: {
                  Authorization: `Bearer \${env:${bearerTokenEnvVar}}`,
                },
              }),
            },
          },
        },
        null,
        2
      ),
    },
  ];
};

const useMCPTools = (
  endpoint: string,
  headers: Record<string, string>,
  getAccessToken?: () => Promise<string | null>
) => {
  const [tools, setTools] = useState<MCPTool[]>([]);
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(true);
  const getAccessTokenRef = useRef(getAccessToken);

  useEffect(() => {
    getAccessTokenRef.current = getAccessToken;
  }, [getAccessToken]);

  useEffect(() => {
    const controller = new AbortController();
    let sessionID: string | null = null;
    let requestHeaders = headers;

    const load = async () => {
      setLoading(true);
      setError(undefined);
      try {
        const accessToken = await getAccessTokenRef.current?.();
        requestHeaders = accessToken
          ? { ...headers, Authorization: `Bearer ${accessToken}` }
          : headers;
        const baseHeaders = {
          Accept: 'application/json, text/event-stream',
          'Content-Type': 'application/json',
          ...requestHeaders,
        };
        const post = (request: MCPRequest, headers = baseHeaders) =>
          fetch(endpoint, {
            method: 'POST',
            headers,
            body: JSON.stringify({ jsonrpc: '2.0', ...request }),
            signal: controller.signal,
          });

        const initialize = await post({
          id: 1,
          method: 'initialize',
          params: {
            protocolVersion: '2025-06-18',
            capabilities: {},
            clientInfo: { name: 'inngest-mcp-page', version: '1.0.0' },
          },
        });
        if (!initialize.ok) {
          throw new Error(`MCP initialization returned HTTP ${initialize.status}`);
        }
        await initialize.json();

        sessionID = initialize.headers.get('Mcp-Session-Id');
        const sessionHeaders = sessionID
          ? { ...baseHeaders, 'Mcp-Session-Id': sessionID }
          : baseHeaders;
        await post({ method: 'notifications/initialized' }, sessionHeaders);

        const response = await post({ id: 2, method: 'tools/list' }, sessionHeaders);
        if (!response.ok) {
          throw new Error(`MCP returned HTTP ${response.status}`);
        }

        const body = (await response.json()) as ToolsListResponse;
        if (body.error) {
          throw new Error(body.error.message ?? 'Unable to list MCP tools');
        }
        setTools(body.result?.tools ?? []);
      } catch (error) {
        if (!controller.signal.aborted) {
          setError(error instanceof Error ? error.message : 'Unable to list MCP tools');
        }
      } finally {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      }
    };

    void load();
    return () => {
      controller.abort();
      if (sessionID) {
        void fetch(endpoint, {
          method: 'DELETE',
          headers: { ...requestHeaders, 'Mcp-Session-Id': sessionID },
        });
      }
    };
  }, [endpoint, headers]);

  return { error, loading, tools };
};

export const MCPSetup = ({
  bearerTokenEnvVar,
  endpoint,
  getAccessToken,
  headers = emptyHeaders,
  isDevServer = false,
}: MCPSetupProps) => {
  const [copiedText, setCopiedText] = useState<string>();
  const surfaceName = isDevServer ? 'Dev Server' : 'Cloud';
  const { error, loading, tools } = useMCPTools(endpoint, headers, getAccessToken);
  const configs = createConfigs(endpoint, isDevServer, bearerTokenEnvVar);
  const examples = isDevServer ? devServerExamples : cloudExamples;

  const copyToClipboard = (text: string, id: string) => {
    void navigator.clipboard.writeText(text);
    setCopiedText(id);
    window.setTimeout(() => setCopiedText(undefined), 2000);
  };

  return (
    <div className="bg-canvasBase min-h-screen">
      <div className="mx-auto max-w-5xl px-8 py-8">
        <header className="mb-12">
          <h1 className="text-basis text-3xl font-semibold">{surfaceName} MCP Setup</h1>
          <p className="text-muted mt-2 text-base">
            Connect your AI assistant to Inngest {surfaceName} using the Model Context Protocol.
          </p>
        </header>

        <section className="mb-12">
          <h2 className="text-basis mb-4 text-2xl font-semibold">MCP Endpoint</h2>
          <div className="bg-canvasSubtle border-subtle rounded border p-4">
            <div className="flex items-center justify-between gap-4">
              <code className="text-basis flex-1 break-all font-mono text-lg">{endpoint}</code>
              <button
                type="button"
                onClick={() => copyToClipboard(endpoint, 'endpoint')}
                className="bg-canvasBase border-subtle hover:bg-canvasMuted shrink-0 rounded border px-4 py-2 text-sm font-medium transition-colors"
              >
                {copiedText === 'endpoint' ? 'Copied!' : 'Copy'}
              </button>
            </div>
          </div>
        </section>

        {bearerTokenEnvVar && (
          <section className="mb-12">
            <h2 className="text-basis mb-2 text-2xl font-semibold">Authentication</h2>
            <p className="text-muted text-sm">
              Installed MCP clients use an Inngest API key. See the{' '}
              <a
                className="text-link hover:underline"
                href="https://api-docs.inngest.com/authentication"
                rel="noopener noreferrer"
                target="_blank"
              >
                authentication docs
              </a>{' '}
              to create one, then export it as{' '}
              <code className="bg-canvasSubtle rounded px-1 py-0.5">{bearerTokenEnvVar}</code>.
            </p>
          </section>
        )}

        <section className="mb-12">
          <h2 className="text-basis mb-4 text-2xl font-semibold">Connect Your AI Tool</h2>
          {configs.map((config) => (
            <div className="mb-4" key={config.id}>
              <h3 className="text-basis mb-2 text-lg font-medium">{config.label}</h3>
              <div className="bg-canvasSubtle border-subtle relative rounded border">
                <div className="border-subtle flex items-center justify-between border-b px-4 py-2">
                  <span className="text-muted text-xs font-medium">{config.language}</span>
                  <button
                    type="button"
                    onClick={() => copyToClipboard(config.value, config.id)}
                    className="text-muted hover:text-basis text-xs"
                  >
                    {copiedText === config.id ? 'Copied!' : 'Copy'}
                  </button>
                </div>
                <pre className="overflow-x-auto p-4">
                  <code className="text-basis text-sm">{config.value}</code>
                </pre>
              </div>
            </div>
          ))}

          <div className="bg-canvasSubtle border-subtle mb-4 rounded border p-4">
            <h3 className="text-basis mb-2 text-lg font-medium">Codex/ChatGPT Desktop</h3>
            <p className="text-basis text-sm">
              Codex CLI and the ChatGPT desktop app share MCP configuration. Run the Codex CLI
              command above, restart the desktop app, then use{' '}
              <strong>Settings → MCP servers</strong> to confirm the server is enabled. You can also
              add it there by choosing <strong>Streamable HTTP</strong> and entering the endpoint
              URL.
            </p>
          </div>

          <div className="bg-canvasSubtle border-subtle mb-4 rounded border p-4">
            <h3 className="text-basis mb-2 text-lg font-medium">Claude Desktop</h3>
            <p className="text-basis text-sm">
              For the desktop app&apos;s <strong>Code</strong> tab, run the Claude Code command
              above and restart Claude Desktop. The Code tab loads the same Claude MCP
              configuration.
            </p>
            {isDevServer ? (
              <p className="text-muted mt-2 text-sm">
                Claude Desktop&apos;s main chat connects to custom HTTP connectors from
                Anthropic&apos;s cloud, so it cannot reach this localhost endpoint directly. That
                flow requires a publicly reachable MCP URL under{' '}
                <strong>Customize → Connectors</strong>.
              </p>
            ) : (
              <p className="text-muted mt-2 text-sm">
                Claude Desktop&apos;s main chat uses remote connectors with OAuth. Cloud MCP
                currently uses API key authentication, so use the Code tab for this connection.
              </p>
            )}
          </div>
        </section>

        <section className="mb-12">
          <h2 className="text-basis mb-4 text-2xl font-semibold">Try It</h2>
          <div className="space-y-2">
            {examples.map((example) => (
              <div className="bg-canvasSubtle border-subtle rounded border p-3" key={example}>
                <code className="text-basis text-sm">{example}</code>
              </div>
            ))}
          </div>
        </section>

        <section className="mb-12">
          <h2 className="text-basis mb-2 text-2xl font-semibold">Available MCP Tools</h2>
          {isDevServer && (
            <div className="bg-warning/10 border-warning/30 mb-6 rounded border p-4">
              <p className="text-basis font-medium">Dev server MCP compatibility changes</p>
              <p className="text-muted mt-1 text-sm">
                <code>list_functions</code> and <code>invoke_function</code> now use the shared REST
                API v2 contract so they work the same in the dev server and Inngest Cloud.{' '}
                <code>list_functions</code> now requires an <code>appId</code>, and{' '}
                <code>invoke_function</code> starts a run without waiting for it to finish. The old
                synchronous invoke behavior remains available as <code>invoke_function_sync</code>.
              </p>
              <p className="text-muted mt-2 text-sm">
                The dev server keeps <code>send_event</code>, <code>get_run_status</code>,{' '}
                <code>poll_run_status</code>, and <code>invoke_function_sync</code> for existing
                integrations. They are deprecated and may be removed in a future release.
              </p>
            </div>
          )}

          <MCPToolList error={error} loading={loading} tools={tools} />
        </section>

        {isDevServer && (
          <>
            <DevServerBestPractices />
            <DevServerTroubleshooting endpoint={endpoint} />
            <Resources />
          </>
        )}
      </div>
    </div>
  );
};

const MCPToolList = ({
  error,
  loading,
  tools,
}: {
  error?: string;
  loading: boolean;
  tools: MCPTool[];
}) => {
  if (loading) {
    return <p className="text-muted">Loading tools…</p>;
  }
  if (error) {
    return (
      <div className="bg-canvasSubtle border-subtle rounded border p-4">
        <p className="text-basis font-medium">Unable to load MCP tools</p>
        <p className="text-muted mt-1 text-sm">{error}</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <p className="text-muted text-sm">
        {tools.length} {tools.length === 1 ? 'tool' : 'tools'} available
      </p>
      {tools.map((tool) => (
        <MCPToolCard key={tool.name} tool={tool} />
      ))}
    </div>
  );
};

const MCPToolCard = ({ tool }: { tool: MCPTool }) => {
  const required = new Set(tool.inputSchema.required ?? []);
  const properties = Object.entries(tool.inputSchema.properties ?? {});

  return (
    <div className="bg-canvasSubtle border-subtle rounded border p-4">
      <h3 className="text-basis text-lg font-medium">{tool.title ?? tool.name}</h3>
      <code className="text-muted text-xs">{tool.name}</code>
      {tool.description && <p className="text-basis mt-2 text-sm">{tool.description}</p>}
      {properties.length > 0 && (
        <ul className="text-basis mt-3 space-y-1 text-sm">
          {properties.map(([name, schema]) => (
            <li key={name}>
              <code className="bg-canvasBase rounded px-1 py-0.5">{name}</code>{' '}
              <span className="text-muted">
                ({toolType(schema)}, {required.has(name) ? 'required' : 'optional'})
                {schema.description ? `: ${schema.description}` : ''}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};

const DevServerBestPractices = () => (
  <section className="mb-12">
    <h2 className="text-basis mb-4 text-2xl font-semibold">Best Practices</h2>

    <div className="mb-6">
      <h3 className="text-basis mb-3 text-xl font-semibold">Function Testing</h3>
      <ul className="text-basis ml-6 list-disc space-y-2 text-base">
        <li>Test individual functions before testing a multi-function workflow.</li>
        <li>Use clear event names and payloads so failures are easier to trace.</li>
        <li>Inspect the run after each test and verify both step input and output.</li>
        <li>Test expected failure paths as well as successful runs.</li>
      </ul>
    </div>

    <div className="mb-6">
      <h3 className="text-basis mb-3 text-xl font-semibold">Debugging Workflows</h3>
      <ul className="text-basis ml-6 list-disc space-y-2 text-base">
        <li>
          Use <code className="bg-canvasSubtle rounded px-1.5 py-0.5">get_run</code> for run state
          and output.
        </li>
        <li>
          Use <code className="bg-canvasSubtle rounded px-1.5 py-0.5">get_run_trace</code> to
          inspect step-by-step execution.
        </li>
        <li>Review the error message, stack trace, inputs, and outputs together.</li>
      </ul>
    </div>

    <div>
      <h3 className="text-basis mb-3 text-xl font-semibold">Documentation Usage</h3>
      <ul className="text-basis ml-6 list-disc space-y-2 text-base">
        <li>
          Use <code className="bg-canvasSubtle rounded px-1.5 py-0.5">grep_docs</code> to find
          relevant guides and examples.
        </li>
        <li>
          Use <code className="bg-canvasSubtle rounded px-1.5 py-0.5">read_doc</code> to read the
          complete source after finding a match.
        </li>
      </ul>
    </div>
  </section>
);

const DevServerTroubleshooting = ({ endpoint }: { endpoint: string }) => (
  <section className="mb-12">
    <h2 className="text-basis mb-4 text-2xl font-semibold">Troubleshooting</h2>
    <div className="space-y-4">
      <TroubleshootingItem title="MCP server not found">
        <li>Restart the dev server if the endpoint is not responding.</li>
        <li>
          Confirm your client uses{' '}
          <code className="bg-canvasBase rounded px-1 py-0.5">{endpoint}</code>.
        </li>
        <li>Check that the client sends its API calls with the streamable HTTP transport.</li>
      </TroubleshootingItem>
      <TroubleshootingItem title="Functions not listed">
        <li>Confirm your app has synced successfully with the dev server.</li>
        <li>Check the dev-server logs for registration or connection errors.</li>
        <li>Refresh the MCP tool call after the app finishes syncing.</li>
      </TroubleshootingItem>
      <TroubleshootingItem title="Run data is missing">
        <li>Allow a moment for event and run data to be stored.</li>
        <li>Confirm the run ID and function trigger match the test you sent.</li>
      </TroubleshootingItem>
    </div>
  </section>
);

const TroubleshootingItem = ({ children, title }: { children: ReactNode; title: string }) => (
  <div className="bg-canvasSubtle border-subtle rounded border p-4">
    <h3 className="text-basis mb-2 text-lg font-semibold">{title}</h3>
    <ul className="text-basis ml-4 list-disc space-y-1 text-sm">{children}</ul>
  </div>
);

const Resources = () => (
  <section className="mb-12">
    <h2 className="text-basis mb-4 text-2xl font-semibold">Resources</h2>
    <div className="grid gap-4 md:grid-cols-2">
      {resources.map((resource) => (
        <a
          className="bg-canvasSubtle border-subtle hover:border-emphasis block rounded border p-4 transition-colors"
          href={resource.href}
          key={resource.href}
          rel="noopener noreferrer"
          target="_blank"
        >
          <h3 className="text-basis mb-2 text-base font-semibold">{resource.title}</h3>
          <p className="text-muted text-sm">{resource.description}</p>
        </a>
      ))}
    </div>
  </section>
);
