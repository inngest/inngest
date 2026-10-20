import { CopyButton } from '@inngest/components/CopyButton';
import { InlineCode } from '@inngest/components/Code';
import { SetupStep } from '@inngest/components/MCP/SetupStep';
import { createMCPClients } from '@inngest/components/MCP/clients';

import { useRunAICopy } from './useRunAICopy';

import type { RunAIAction } from './tracking';

export type SetupClient = 'claude' | 'codex' | 'other';

export function RunAISetup({
  client,
  endpoint,
  onAction,
}: {
  client: SetupClient;
  endpoint: string;
  onAction: (event: RunAIAction) => void;
}) {
  const trackingClient =
    client === 'claude'
      ? 'claude_code'
      : client === 'codex'
        ? 'codex_cli'
        : 'other';
  const setup = createMCPClients(endpoint, false).find(
    (item) => item.id === client,
  );
  return (
    <section aria-label="Connection instructions" className="mt-3 text-xs">
      <SetupStep
        compact
        number={1}
        title={
          client === 'other'
            ? 'Add this MCP server URL'
            : 'Run in your terminal'
        }
      >
        <SetupCommand
          code={setup?.snippet.content ?? endpoint}
          label={client === 'other' ? 'Copy server URL' : 'Copy setup command'}
          client={trackingClient}
          onAction={onAction}
        />
      </SetupStep>
      <SetupStep
        compact
        number={2}
        title={client === 'other' ? 'Sign in with OAuth' : 'Sign in'}
      >
        {client === 'codex' ? (
          <SetupCommand
            code="codex mcp login inngest-cloud"
            label="Copy sign-in command"
            client={trackingClient}
            onAction={onAction}
          />
        ) : client === 'claude' ? (
          <p className="text-muted leading-5">
            Open <InlineCode>/mcp</InlineCode>, select{' '}
            <InlineCode>inngest-cloud</InlineCode>, then sign in.
          </p>
        ) : null}
      </SetupStep>
      <SetupStep compact number={3} isLast title="Choose access">
        <p className="text-muted leading-5">
          Select your account, environment, and read access.
        </p>
      </SetupStep>
      <a
        href="/mcp/setup"
        target="_blank"
        rel="noopener noreferrer"
        className="text-link mt-3 inline-block hover:underline"
        onClick={() =>
          onAction({ action: 'guide_clicked', guide: 'connection' })
        }
      >
        Setup guide
      </a>
    </section>
  );
}

function SetupCommand({
  code,
  label,
  client,
  onAction,
}: {
  code: string;
  label: string;
  client: 'claude_code' | 'codex_cli' | 'other';
  onAction: (event: RunAIAction) => void;
}) {
  const { status, copy } = useRunAICopy(code);
  async function copyCommand() {
    if (status === 'copying') return;
    const failure = await copy(code);
    onAction({
      action: failure ? 'setup_copy_failed' : 'setup_copied',
      client,
    });
  }

  return (
    <div>
      <div className="bg-codeEditor flex items-center gap-2 rounded p-2">
        <code className="text-basis min-w-0 flex-1 select-text break-words font-mono text-[11px] leading-4">
          {code}
        </code>
        <CopyButton
          ariaLabel={label}
          code={code}
          appearance="outlined"
          className="text-muted border-subtle shrink-0"
          size="small"
          isCopying={status === 'copied'}
          handleCopyClick={() => void copyCommand()}
        />
      </div>
      {status === 'failed' && (
        <p role="status" className="text-muted mt-1 text-xs">
          Couldn’t copy. Select and copy the text above.
        </p>
      )}
    </div>
  );
}
