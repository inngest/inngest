import { useEffect, useRef, useState, type ComponentType, type ReactNode } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Card } from '@inngest/components/Card';
import { InlineCode } from '@inngest/components/Code';
import CommandBlock, { type TabsProps } from '@inngest/components/CodeBlock/CommandBlock';
import { CodeLine } from '@inngest/components/CodeLine';
import { Pill } from '@inngest/components/Pill';
import { Skeleton } from '@inngest/components/Skeleton';
import TabCards from '@inngest/components/TabCards/TabCards';
import { IconClaude } from '@inngest/components/icons/ai/Claude';
import { IconCursor } from '@inngest/components/icons/ai/Cursor';
import { IconOpenAI } from '@inngest/components/icons/ai/OpenAI';
import { LINE_HEIGHT } from '@inngest/components/utils/monaco';
import { ClientOnly } from '@tanstack/react-router';

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

// Abort the tools handshake if it hangs (an unresponsive proxy, a stalled
// auth-token fetch) so the loading skeletons resolve into an actionable
// error instead of shimmering forever.
const toolsFetchTimeoutMs = 15_000;

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

type MCPClient = {
  id: 'claude' | 'codex' | 'cursor';
  name: string;
  Icon: ComponentType<{ className?: string; size?: number }>;
  snippet: TabsProps;
};

const createClients = (
  endpoint: string,
  isDevServer: boolean,
  bearerTokenEnvVar?: string
): MCPClient[] => {
  const serverName = isDevServer ? 'inngest-dev' : 'inngest-cloud';
  const authorizationHeader = bearerTokenEnvVar
    ? ` --header "Authorization: Bearer $${bearerTokenEnvVar}"`
    : '';
  const bearerTokenOption = bearerTokenEnvVar ? ` --bearer-token-env-var ${bearerTokenEnvVar}` : '';

  return [
    {
      id: 'claude',
      name: 'Claude Code',
      Icon: IconClaude,
      snippet: {
        title: 'Terminal command',
        language: 'shell',
        content: `claude mcp add --transport http ${serverName} ${endpoint}${authorizationHeader}`,
      },
    },
    {
      id: 'codex',
      name: 'Codex CLI',
      Icon: IconOpenAI,
      snippet: {
        title: 'Terminal command',
        language: 'shell',
        content: `codex mcp add ${serverName} --url ${endpoint}${bearerTokenOption}`,
      },
    },
    {
      id: 'cursor',
      name: 'Cursor',
      Icon: IconCursor,
      snippet: {
        title: 'mcp.json',
        language: 'json',
        content: JSON.stringify(
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
    },
  ];
};

// Approximate rendered height of a CommandBlock (tabs header + monaco padding
// + content lines) so the SSR fallback skeleton matches and hydration doesn't
// shift the page.
const snippetHeight = (snippet: TabsProps) => snippet.content.split('\n').length * LINE_HEIGHT + 60;

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
    // Distinguishes unmount (stay silent) from the watchdog abort below
    // (surface a timeout error) — both flip controller.signal.aborted.
    let unmounted = false;
    const watchdog = window.setTimeout(() => controller.abort(), toolsFetchTimeoutMs);
    // getAccessToken is outside fetch's abort handling, so race it against
    // the watchdog too; a hung token fetch would otherwise never settle.
    const abortSignal = new Promise<never>((_, reject) => {
      const fail = () => reject(new Error('aborted'));
      if (controller.signal.aborted) {
        fail();
        return;
      }
      controller.signal.addEventListener('abort', fail, { once: true });
    });

    const load = async () => {
      setLoading(true);
      setError(undefined);
      try {
        const accessToken = await Promise.race([
          getAccessTokenRef.current?.() ?? null,
          abortSignal,
        ]);
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
        if (!unmounted) {
          setError(
            controller.signal.aborted
              ? `The MCP endpoint did not respond within ${
                  toolsFetchTimeoutMs / 1000
                } seconds. Check that ${endpoint} is reachable from this page.`
              : error instanceof Error
              ? error.message
              : 'Unable to list MCP tools'
          );
        }
      } finally {
        window.clearTimeout(watchdog);
        if (!unmounted) {
          setLoading(false);
        }
      }
    };

    void load();
    return () => {
      unmounted = true;
      window.clearTimeout(watchdog);
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
  const surfaceName = isDevServer ? 'Dev Server' : 'Cloud';
  const { error, loading, tools } = useMCPTools(endpoint, headers, getAccessToken);
  const clients = createClients(endpoint, isDevServer, bearerTokenEnvVar);
  const examples = isDevServer ? devServerExamples : cloudExamples;

  return (
    <div className="bg-canvasBase min-h-full">
      <div className="mx-auto max-w-5xl px-8 py-8">
        <header className="mb-10">
          <h1 className="text-basis text-2xl font-medium">{surfaceName} MCP Setup</h1>
          <p className="text-muted mt-1 text-sm">
            Connect your AI assistant to Inngest {surfaceName} using the Model Context Protocol.
          </p>
        </header>

        <section className="mb-10">
          <h2 className="text-basis mb-3 text-lg font-medium">MCP endpoint</h2>
          <CodeLine code={endpoint} />
        </section>

        {bearerTokenEnvVar && (
          <section className="mb-10">
            <h2 className="text-basis mb-2 text-lg font-medium">Authentication</h2>
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
              to create one, then export it as <InlineCode>{bearerTokenEnvVar}</InlineCode>.
            </p>
          </section>
        )}

        <section className="mb-10">
          <h2 className="text-basis mb-3 text-lg font-medium">Connect your AI tool</h2>
          <TabCards defaultValue="claude">
            <TabCards.ButtonList>
              {clients.map((client) => (
                <TabCards.Button className="w-36" key={client.id} value={client.id}>
                  <div className="flex items-center gap-1.5">
                    <client.Icon className="h-4 w-4" /> {client.name}
                  </div>
                </TabCards.Button>
              ))}
            </TabCards.ButtonList>
            {clients.map((client) => (
              <TabCards.Content className="min-h-[24rem]" key={client.id} value={client.id}>
                <div className="mb-4 flex items-center gap-2">
                  <div className="bg-canvasMuted flex h-9 w-9 items-center justify-center rounded">
                    <client.Icon className="text-basis h-4 w-4" />
                  </div>
                  <p className="text-basis">{client.name}</p>
                </div>
                <ClientOnly
                  fallback={
                    <div style={{ height: snippetHeight(client.snippet) }}>
                      <Skeleton className="h-full w-full" />
                    </div>
                  }
                >
                  <CommandBlock.Wrapper>
                    <CommandBlock.Header className="flex items-center justify-between pr-4">
                      <CommandBlock.Tabs tabs={[client.snippet]} activeTab={client.snippet.title} />
                      <CommandBlock.CopyButton content={client.snippet.content} />
                    </CommandBlock.Header>
                    <CommandBlock currentTabContent={client.snippet} />
                  </CommandBlock.Wrapper>
                </ClientOnly>
                <ClientNotes client={client.id} isDevServer={isDevServer} />
              </TabCards.Content>
            ))}
          </TabCards>
        </section>

        <section className="mb-10">
          <h2 className="text-basis mb-3 text-lg font-medium">Try it</h2>
          <p className="text-muted mb-3 text-sm">
            Ask your AI assistant a question that uses the Inngest tools:
          </p>
          <div className="space-y-2">
            {examples.map((example) => (
              <CodeLine code={example} key={example} />
            ))}
          </div>
        </section>

        <section className="mb-10">
          <h2 className="text-basis mb-3 text-lg font-medium">Available MCP tools</h2>
          {isDevServer && (
            <Alert className="mb-6" severity="warning">
              <p className="font-medium">Dev server MCP compatibility changes</p>
              <p className="mt-1 text-sm">
                <code>list_functions</code> and <code>invoke_function</code> now use the shared REST
                API v2 contract so they work the same in the dev server and Inngest Cloud.{' '}
                <code>list_functions</code> now requires an <code>appId</code>, and{' '}
                <code>invoke_function</code> starts a run without waiting for it to finish. The old
                synchronous invoke behavior remains available as <code>invoke_function_sync</code>.
              </p>
              <p className="mt-2 text-sm">
                The dev server keeps <code>send_event</code>, <code>get_run_status</code>,{' '}
                <code>poll_run_status</code>, and <code>invoke_function_sync</code> for existing
                integrations. They are deprecated and may be removed in a future release.
              </p>
            </Alert>
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

const ClientNotes = ({
  client,
  isDevServer,
}: {
  client: MCPClient['id'];
  isDevServer: boolean;
}) => {
  if (client === 'claude') {
    return (
      <div className="text-muted mt-3 text-sm">
        <p>
          For the desktop app&apos;s <strong>Code</strong> tab, run the command above and restart
          Claude Desktop. The Code tab loads the same Claude MCP configuration.
        </p>
        {isDevServer ? (
          <p className="mt-2">
            Claude Desktop&apos;s main chat connects to custom HTTP connectors from Anthropic&apos;s
            cloud, so it cannot reach this localhost endpoint directly. That flow requires a
            publicly reachable MCP URL under <strong>Customize → Connectors</strong>.
          </p>
        ) : (
          <p className="mt-2">
            Claude Desktop&apos;s main chat uses remote connectors with OAuth. Cloud MCP currently
            uses API key authentication, so use the Code tab for this connection.
          </p>
        )}
      </div>
    );
  }
  if (client === 'codex') {
    return (
      <div className="text-muted mt-3 text-sm">
        <p>
          Codex CLI and the ChatGPT desktop app share MCP configuration. Run the command above,
          restart the desktop app, then use <strong>Settings → MCP servers</strong> to confirm the
          server is enabled. You can also add it there by choosing <strong>Streamable HTTP</strong>{' '}
          and entering the endpoint URL.
        </p>
      </div>
    );
  }
  if (client === 'cursor') {
    return (
      <div className="text-muted mt-3 text-sm">
        <p>
          Add this to <InlineCode>.cursor/mcp.json</InlineCode> in your project, or to{' '}
          <InlineCode>~/.cursor/mcp.json</InlineCode> to enable it in every project, then reload
          Cursor.
        </p>
      </div>
    );
  }
  return null;
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
    return (
      <div className="space-y-4">
        <Skeleton className="h-28 w-full" />
        <Skeleton className="h-28 w-full" />
        <Skeleton className="h-28 w-full" />
      </div>
    );
  }
  if (error) {
    return (
      <Alert severity="error">
        <p className="font-medium">Unable to load MCP tools</p>
        <p className="mt-1 text-sm">{error}</p>
      </Alert>
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
    <Card>
      <Card.Content>
        <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
          <h3 className="text-basis text-base font-medium">{tool.title ?? tool.name}</h3>
          <InlineCode>{tool.name}</InlineCode>
        </div>
        {tool.description && <p className="text-basis mt-2 text-sm">{tool.description}</p>}
        {properties.length > 0 && (
          <ul className="mt-3 space-y-1.5 text-sm">
            {properties.map(([name, schema]) => (
              <li className="flex flex-wrap items-center gap-x-2 gap-y-1" key={name}>
                <InlineCode>{name}</InlineCode>
                <span className="text-muted">{toolType(schema)}</span>
                {required.has(name) && <Pill appearance="outlined">required</Pill>}
                {schema.description && <span className="text-muted">{schema.description}</span>}
              </li>
            ))}
          </ul>
        )}
      </Card.Content>
    </Card>
  );
};

const DevServerBestPractices = () => (
  <section className="mb-10">
    <h2 className="text-basis mb-4 text-lg font-medium">Best practices</h2>

    <div className="mb-6">
      <h3 className="text-basis mb-2 text-base font-medium">Function testing</h3>
      <ul className="text-basis ml-6 list-disc space-y-1.5 text-sm">
        <li>Test individual functions before testing a multi-function workflow.</li>
        <li>Use clear event names and payloads so failures are easier to trace.</li>
        <li>Inspect the run after each test and verify both step input and output.</li>
        <li>Test expected failure paths as well as successful runs.</li>
      </ul>
    </div>

    <div className="mb-6">
      <h3 className="text-basis mb-2 text-base font-medium">Debugging workflows</h3>
      <ul className="text-basis ml-6 list-disc space-y-1.5 text-sm">
        <li>
          Use <InlineCode>get_run</InlineCode> for run state and output.
        </li>
        <li>
          Use <InlineCode>get_run_trace</InlineCode> to inspect step-by-step execution.
        </li>
        <li>Review the error message, stack trace, inputs, and outputs together.</li>
      </ul>
    </div>

    <div>
      <h3 className="text-basis mb-2 text-base font-medium">Documentation usage</h3>
      <ul className="text-basis ml-6 list-disc space-y-1.5 text-sm">
        <li>
          Use <InlineCode>grep_docs</InlineCode> to find relevant guides and examples.
        </li>
        <li>
          Use <InlineCode>read_doc</InlineCode> to read the complete source after finding a match.
        </li>
      </ul>
    </div>
  </section>
);

const DevServerTroubleshooting = ({ endpoint }: { endpoint: string }) => (
  <section className="mb-10">
    <h2 className="text-basis mb-3 text-lg font-medium">Troubleshooting</h2>
    <div className="space-y-4">
      <TroubleshootingItem title="MCP server not found">
        <li>Restart the dev server if the endpoint is not responding.</li>
        <li>
          Confirm your client uses <InlineCode>{endpoint}</InlineCode>.
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
  <Card>
    <Card.Content>
      <h3 className="text-basis mb-2 text-base font-medium">{title}</h3>
      <ul className="text-basis ml-4 list-disc space-y-1 text-sm">{children}</ul>
    </Card.Content>
  </Card>
);

const Resources = () => (
  <section className="mb-10">
    <h2 className="text-basis mb-3 text-lg font-medium">Resources</h2>
    <div className="grid gap-4 md:grid-cols-2">
      {resources.map((resource) => (
        <a
          className="block"
          href={resource.href}
          key={resource.href}
          rel="noopener noreferrer"
          target="_blank"
        >
          <Card
            className="h-full"
            contentClassName="hover:border-emphasis h-full transition-colors"
          >
            <Card.Content>
              <h3 className="text-basis mb-1 text-sm font-medium">{resource.title}</h3>
              <p className="text-muted text-sm">{resource.description}</p>
            </Card.Content>
          </Card>
        </a>
      ))}
    </div>
  </section>
);
