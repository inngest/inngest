import { useState, type ComponentType, type ReactNode } from 'react';
import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card';
import { InlineCode } from '@inngest/components/Code';
import CommandBlock, { type TabsProps } from '@inngest/components/CodeBlock/CommandBlock';
import { CodeLine } from '@inngest/components/CodeLine';
import { Search } from '@inngest/components/Forms/Search';
import { Pill } from '@inngest/components/Pill';
import { Skeleton } from '@inngest/components/Skeleton';
import TabCards from '@inngest/components/TabCards/TabCards';
import { IconClaude } from '@inngest/components/icons/ai/Claude';
import { IconCursor } from '@inngest/components/icons/ai/Cursor';
import { IconOpenAI } from '@inngest/components/icons/ai/OpenAI';
import { LINE_HEIGHT } from '@inngest/components/utils/monaco';
import { RiBookOpenLine, RiBugLine, RiCheckLine, RiFlaskLine } from '@remixicon/react';
import { useQuery } from '@tanstack/react-query';
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
  /** In-app path to the API keys settings page, linked from the auth step. */
  apiKeysHref?: string;
  bearerTokenEnvVar?: string;
  endpoint: string;
  isDevServer?: boolean;
  operationsEndpoint: string;
};

type Operation = {
  mcp?: MCPTool;
};

type OperationsResponse = {
  data: { operations: Operation[] };
};

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

// CodeLine's ghost copy button renders text-basis, which reads too heavy
// against this page's code lines; mute it until hovered.
const mutedCopyButton = '[&_button]:text-muted [&_button:hover]:text-basis';

const useMCPTools = (operationsEndpoint: string) => {
  const query = useQuery({
    queryKey: ['v2-operations', operationsEndpoint],
    queryFn: async (): Promise<MCPTool[]> => {
      const response = await fetch(operationsEndpoint);
      if (!response.ok) {
        throw new Error(`Operations endpoint returned HTTP ${response.status}`);
      }

      const body = (await response.json()) as OperationsResponse;
      return body.data.operations.flatMap((operation) => (operation.mcp ? [operation.mcp] : []));
    },
    retry: false,
    refetchOnWindowFocus: false,
  });

  return {
    error: query.error?.message,
    loading: query.isPending,
    retry: () => void query.refetch(),
    tools: query.data ?? [],
  };
};

export const MCPSetup = ({
  apiKeysHref,
  bearerTokenEnvVar,
  endpoint,
  isDevServer = false,
  operationsEndpoint,
}: MCPSetupProps) => {
  const surfaceName = isDevServer ? 'Dev Server' : 'Cloud';
  const { error, loading, retry, tools } = useMCPTools(operationsEndpoint);
  const clients = createClients(endpoint, isDevServer, bearerTokenEnvVar);
  const examples = isDevServer ? devServerExamples : cloudExamples;
  const hasAuthStep = Boolean(bearerTokenEnvVar);
  const connectStep = hasAuthStep ? 2 : 1;
  const tryStep = connectStep + 1;

  return (
    <div className="bg-canvasBase min-h-full">
      <div className="mx-auto max-w-5xl px-8 py-8">
        <header className="mb-8">
          <h1 className="text-basis text-2xl font-medium">{surfaceName} MCP Setup</h1>
          <p className="text-muted mt-1 text-sm">
            Connect your AI assistant to Inngest {surfaceName} using the Model Context Protocol.
          </p>
        </header>

        <section className="mb-12">
          <h2 className="text-basis mb-6 text-lg font-medium">Get started</h2>

          {bearerTokenEnvVar && (
            <Step number={1} title="Create an API key">
              <p className="text-muted mb-3 text-sm">
                MCP clients authenticate with an Inngest API key. Create one
                {!apiKeysHref && <> in your organization&apos;s API keys settings</>}, then export
                it in your terminal. See the{' '}
                <a
                  className="text-link hover:underline"
                  href="https://api-docs.inngest.com/authentication"
                  rel="noopener noreferrer"
                  target="_blank"
                >
                  authentication docs
                </a>{' '}
                to learn more.
              </p>
              <CodeLine
                className={mutedCopyButton}
                code={`export ${bearerTokenEnvVar}=<your-api-key>`}
              />
              {apiKeysHref && (
                <div className="mt-3">
                  <Button href={apiKeysHref} kind="primary" label="Create API key" />
                </div>
              )}
            </Step>
          )}

          <Step number={connectStep} title="Connect your AI tool">
            <p className="text-muted mb-2 text-sm">
              Add the Inngest MCP server to your tool of choice. If your tool asks for a server URL,
              use this MCP endpoint:
            </p>
            <CodeLine className={`mb-4 ${mutedCopyButton}`} code={endpoint} />
            <ClientPicker
              bearerTokenEnvVar={bearerTokenEnvVar}
              clients={clients}
              isDevServer={isDevServer}
            />
          </Step>

          <Step isLast number={tryStep} title="Try it">
            <p className="text-muted mb-3 text-sm">
              Ask your AI assistant a question that uses the Inngest tools:
            </p>
            <div className="space-y-2">
              {examples.map((example) => (
                <CodeLine className={mutedCopyButton} code={example} key={example} />
              ))}
            </div>
          </Step>
        </section>

        <section className="mb-12">
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

          <MCPToolList error={error} loading={loading} retry={retry} tools={tools} />
        </section>

        {isDevServer && (
          <section className="mb-12">
            <DevServerBestPractices />
          </section>
        )}

        <section className="mb-12">
          <Troubleshooting
            bearerTokenEnvVar={bearerTokenEnvVar}
            endpoint={endpoint}
            isDevServer={isDevServer}
          />
        </section>
      </div>
    </div>
  );
};

const Step = ({
  children,
  isLast = false,
  number,
  title,
}: {
  children: ReactNode;
  isLast?: boolean;
  number: number;
  title: string;
}) => (
  <div className="flex gap-4">
    <div className="flex flex-col items-center">
      <div className="border-subtle bg-canvasBase text-basis flex h-7 w-7 shrink-0 items-center justify-center rounded-full border text-xs font-medium">
        {number}
      </div>
      <div className="border-subtle my-1 w-px flex-1 border-l" />
    </div>
    <div className={`min-w-0 flex-1 ${isLast ? '' : 'pb-8'}`}>
      <h3 className="text-basis mb-2 text-base font-medium leading-7">{title}</h3>
      {children}
    </div>
  </div>
);

const ClientPicker = ({
  bearerTokenEnvVar,
  clients,
  isDevServer,
}: {
  bearerTokenEnvVar?: string;
  clients: MCPClient[];
  isDevServer: boolean;
}) => (
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
      <TabCards.Content key={client.id} value={client.id}>
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
        <ClientNotes
          bearerTokenEnvVar={bearerTokenEnvVar}
          client={client.id}
          isDevServer={isDevServer}
        />
      </TabCards.Content>
    ))}
  </TabCards>
);

const ClientNotes = ({
  bearerTokenEnvVar,
  client,
  isDevServer,
}: {
  bearerTokenEnvVar?: string;
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
        {bearerTokenEnvVar && (
          <p className="mb-2">
            Codex reads <InlineCode>{bearerTokenEnvVar}</InlineCode> on every launch, so keep the
            export from step 1 in your shell profile.
          </p>
        )}
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
        {bearerTokenEnvVar && (
          <p className="mt-2">
            Cursor reads <InlineCode>{`\${env:${bearerTokenEnvVar}}`}</InlineCode> on launch, so
            keep the export from step 1 in your shell profile, or replace it with your key in this
            file.
          </p>
        )}
      </div>
    );
  }
  return null;
};

const matchesToolSearch = (tool: MCPTool, query: string) => {
  const q = query.trim().toLowerCase();
  if (!q) {
    return true;
  }
  return [tool.name, tool.title, tool.description].some((field) =>
    field?.toLowerCase().includes(q)
  );
};

// Tool names follow a verb_resource convention (list_functions, get_run_trace,
// send_event), so the resource token gives the list its shape without any
// backend support. Definition order is MATCH priority, not display order:
// the more specific resource wins when a name contains two (sandbox before
// env, webhook before event, run before function). Unknown names land in
// Other.
const toolGroupDefs: Array<{ label: string; tokens: string[] }> = [
  { label: 'Sandboxes', tokens: ['sandbox', 'sandboxes'] },
  { label: 'Environments', tokens: ['env', 'envs', 'environment', 'environments'] },
  { label: 'Apps', tokens: ['app', 'apps'] },
  { label: 'Runs', tokens: ['run', 'runs'] },
  { label: 'Functions', tokens: ['function', 'functions'] },
  { label: 'Webhooks', tokens: ['webhook', 'webhooks'] },
  { label: 'Events', tokens: ['event', 'events'] },
  { label: 'Docs', tokens: ['doc', 'docs'] },
];

// Display order, most-used resources first.
const toolGroupDisplayOrder = [
  'Apps',
  'Runs',
  'Functions',
  'Environments',
  'Events',
  'Webhooks',
  'Sandboxes',
  'Docs',
  'Other',
];

const toolGroup = (tool: MCPTool): string => {
  const tokens = tool.name.toLowerCase().split(/[^a-z0-9]+/);
  for (const group of toolGroupDefs) {
    if (group.tokens.some((token) => tokens.includes(token))) {
      return group.label;
    }
  }
  return 'Other';
};

const groupTools = (tools: MCPTool[]) => {
  // A group defined for matching but missing from the display order must not
  // hide its tools: slot any such group in before Other.
  const missing = toolGroupDefs
    .map((group) => group.label)
    .filter((label) => !toolGroupDisplayOrder.includes(label));
  const order = [
    ...toolGroupDisplayOrder.filter((label) => label !== 'Other'),
    ...missing,
    'Other',
  ];
  return order
    .map((label) => ({ label, tools: tools.filter((tool) => toolGroup(tool) === label) }))
    .filter((group) => group.tools.length > 0);
};

const MCPToolList = ({
  error,
  loading,
  retry,
  tools,
}: {
  error?: string;
  loading: boolean;
  retry: () => void;
  tools: MCPTool[];
}) => {
  const [search, setSearch] = useState('');

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
      <Alert
        button={<Button appearance="outlined" kind="secondary" label="Retry" onClick={retry} />}
        severity="error"
      >
        <p className="font-medium">Unable to load MCP tools</p>
        <p className="mt-1 text-sm">{error}</p>
      </Alert>
    );
  }

  const visibleTools = tools.filter((tool) => matchesToolSearch(tool, search));
  const groups = groupTools(visibleTools);

  return (
    <div className="space-y-4">
      <Search
        className="w-[182px]"
        name="search"
        onUpdate={setSearch}
        placeholder="Search tools"
        value={search}
      />
      {groups.length === 0 ? (
        <p className="text-muted py-4 text-sm">
          No tools match <span className="text-basis">{search}</span>.
        </p>
      ) : (
        /* Accordion content on this page uses forceMount + CSS hiding so the
           full text stays in the DOM: agents and scrapers reading the page get
           everything without having to expand each row. */
        groups.map((group) => (
          <div key={group.label}>
            {!(groups.length === 1 && group.label === 'Other') && (
              <h3 className="text-muted mb-2 text-xs font-medium uppercase tracking-wide">
                {group.label}
              </h3>
            )}
            <AccordionList type="multiple" defaultValue={[]}>
              {group.tools.map((tool) => (
                <AccordionList.Item key={tool.name} value={tool.name}>
                  <AccordionList.Trigger>
                    <div className="flex min-w-0 flex-1 items-baseline gap-x-2 overflow-hidden text-left">
                      <span className="text-basis shrink-0 font-medium">
                        {tool.title ?? tool.name}
                      </span>
                      <InlineCode>{tool.name}</InlineCode>
                      {tool.description && (
                        <span className="text-muted min-w-0 truncate font-normal group-data-[state=open]:hidden">
                          {tool.description}
                        </span>
                      )}
                    </div>
                  </AccordionList.Trigger>
                  <AccordionList.Content className="data-[state=closed]:hidden" forceMount>
                    <MCPToolDetails tool={tool} />
                  </AccordionList.Content>
                </AccordionList.Item>
              ))}
            </AccordionList>
          </div>
        ))
      )}
    </div>
  );
};

const MCPToolDetails = ({ tool }: { tool: MCPTool }) => {
  const required = new Set(tool.inputSchema.required ?? []);
  const properties = Object.entries(tool.inputSchema.properties ?? {});

  return (
    <>
      {tool.description && <p className="text-basis text-sm">{tool.description}</p>}
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
    </>
  );
};

const BestPracticeCard = ({
  children,
  Icon,
  tileClassName,
  title,
}: {
  children: ReactNode;
  Icon: ComponentType<{ className?: string }>;
  tileClassName: string;
  title: string;
}) => (
  <Card className="h-full">
    <Card.Content className="h-full p-4">
      <div className={`${tileClassName} w-fit rounded-sm p-[10px]`}>
        <Icon className="h-5 w-5" />
      </div>
      <p className="text-basis mb-2 mt-3 font-medium">{title}</p>
      <ul className="space-y-1.5">{children}</ul>
    </Card.Content>
  </Card>
);

const BestPracticeItem = ({ children }: { children: ReactNode }) => (
  <li className="text-muted flex gap-2 text-sm">
    <RiCheckLine aria-hidden="true" className="text-primary-moderate mt-0.5 h-4 w-4 shrink-0" />
    <span>{children}</span>
  </li>
);

const DevServerBestPractices = () => (
  <div>
    <h2 className="text-basis mb-4 text-lg font-medium">Best practices</h2>
    <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
      <BestPracticeCard
        Icon={RiFlaskLine}
        tileClassName="bg-primary-3xSubtle"
        title="Function testing"
      >
        <BestPracticeItem>Test functions individually before whole workflows.</BestPracticeItem>
        <BestPracticeItem>Use clear event names and payloads.</BestPracticeItem>
        <BestPracticeItem>Verify step input and output, including failure paths.</BestPracticeItem>
      </BestPracticeCard>
      <BestPracticeCard
        Icon={RiBugLine}
        tileClassName="bg-tertiary-3xSubtle"
        title="Debugging workflows"
      >
        <BestPracticeItem>
          Use <InlineCode>get_run</InlineCode> for run state and output.
        </BestPracticeItem>
        <BestPracticeItem>
          Use <InlineCode>get_run_trace</InlineCode> for step-by-step execution.
        </BestPracticeItem>
        <BestPracticeItem>Review errors with their inputs and outputs.</BestPracticeItem>
      </BestPracticeCard>
      <BestPracticeCard
        Icon={RiBookOpenLine}
        tileClassName="bg-quaternary-cool3xSubtle"
        title="Documentation usage"
      >
        <BestPracticeItem>
          Use <InlineCode>grep_docs</InlineCode> to find guides and examples.
        </BestPracticeItem>
        <BestPracticeItem>
          Use <InlineCode>read_doc</InlineCode> to read a match in full.
        </BestPracticeItem>
      </BestPracticeCard>
    </div>
  </div>
);

const Troubleshooting = ({
  bearerTokenEnvVar,
  endpoint,
  isDevServer,
}: {
  bearerTokenEnvVar?: string;
  endpoint: string;
  isDevServer: boolean;
}) => (
  <div>
    <h2 className="text-basis mb-3 text-lg font-medium">Troubleshooting</h2>
    {/* Borderless FAQ treatment (same as AppFAQ) so this section reads
        differently from the boxed tool list above. */}
    <AccordionList className="rounded-none border-0" type="multiple" defaultValue={[]}>
      <AccordionList.Item value="server-not-found">
        <AccordionList.Trigger>
          <span className="text-basis font-medium">MCP server not found</span>
        </AccordionList.Trigger>
        <AccordionList.Content className="data-[state=closed]:hidden" forceMount>
          <ul className="text-basis ml-4 list-disc space-y-1 text-sm">
            {isDevServer && <li>Restart the dev server if the endpoint is not responding.</li>}
            <li>
              Confirm your client uses <InlineCode>{endpoint}</InlineCode>.
            </li>
            <li>Check that the client sends its API calls with the streamable HTTP transport.</li>
            <li>Restart the client after adding or changing the server configuration.</li>
          </ul>
        </AccordionList.Content>
      </AccordionList.Item>
      {!isDevServer && bearerTokenEnvVar && (
        <AccordionList.Item value="unauthorized">
          <AccordionList.Trigger>
            <span className="text-basis font-medium">Requests fail with HTTP 401</span>
          </AccordionList.Trigger>
          <AccordionList.Content className="data-[state=closed]:hidden" forceMount>
            <ul className="text-basis ml-4 list-disc space-y-1 text-sm">
              <li>
                Confirm <InlineCode>{bearerTokenEnvVar}</InlineCode> is exported in the shell your
                client runs from, then restart the client.
              </li>
              <li>Check that the API key has not been deleted or expired.</li>
              <li>Terminal-based clients do not see keys exported in another terminal window.</li>
            </ul>
          </AccordionList.Content>
        </AccordionList.Item>
      )}
      <AccordionList.Item value="functions-not-listed">
        <AccordionList.Trigger>
          <span className="text-basis font-medium">Functions not listed</span>
        </AccordionList.Trigger>
        <AccordionList.Content className="data-[state=closed]:hidden" forceMount>
          <ul className="text-basis ml-4 list-disc space-y-1 text-sm">
            {isDevServer ? (
              <>
                <li>Confirm your app has synced successfully with the dev server.</li>
                <li>Check the dev-server logs for registration or connection errors.</li>
                <li>Refresh the MCP tool call after the app finishes syncing.</li>
              </>
            ) : (
              <>
                <li>Confirm your app has synced successfully in this environment.</li>
                <li>
                  <InlineCode>list_functions</InlineCode> requires an <InlineCode>appId</InlineCode>
                  ; list apps first to find it.
                </li>
              </>
            )}
          </ul>
        </AccordionList.Content>
      </AccordionList.Item>
      {isDevServer && (
        <AccordionList.Item value="run-data-missing">
          <AccordionList.Trigger>
            <span className="text-basis font-medium">Run data is missing</span>
          </AccordionList.Trigger>
          <AccordionList.Content className="data-[state=closed]:hidden" forceMount>
            <ul className="text-basis ml-4 list-disc space-y-1 text-sm">
              <li>Allow a moment for event and run data to be stored.</li>
              <li>Confirm the run ID and function trigger match the test you sent.</li>
            </ul>
          </AccordionList.Content>
        </AccordionList.Item>
      )}
    </AccordionList>
  </div>
);
