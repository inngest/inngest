import { Button } from '@inngest/components/Button';
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
      <ol className="text-muted list-decimal space-y-3 pl-4">
        <li className="pl-1">
          <p>
            {client === 'other'
              ? 'Add this MCP server URL:'
              : 'Run in your terminal:'}
          </p>
          <SetupCommand
            code={setup?.snippet.content ?? endpoint}
            label={
              client === 'other' ? 'Copy server URL' : 'Copy setup command'
            }
            client={trackingClient}
            onAction={onAction}
          />
        </li>
        <li className="pl-1">
          {client === 'codex' ? (
            <>
              <p>Sign in:</p>
              <SetupCommand
                code="codex mcp login inngest-cloud"
                label="Copy sign-in command"
                client={trackingClient}
                onAction={onAction}
              />
            </>
          ) : client === 'claude' ? (
            'In Claude Code: /mcp → inngest-cloud → Sign in.'
          ) : (
            'Sign in with OAuth.'
          )}
        </li>
        <li className="pl-1">
          Choose your account, environment, and read access.
        </li>
      </ol>
      <a
        href="/mcp/setup"
        target="_blank"
        rel="noopener noreferrer"
        className="text-link mt-3 inline-block hover:underline"
        onClick={() =>
          onAction({ action: 'guide_clicked', guide: 'connection' })
        }
      >
        Setup guide ↗
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
    <div className="mt-1.5">
      <div className="bg-canvasSubtle flex items-start gap-2 rounded p-2">
        <code className="text-basis min-w-0 flex-1 select-text break-words font-mono text-xs">
          {code}
        </code>
        <Button
          aria-label={label}
          label={status === 'copied' ? 'Copied' : 'Copy'}
          onClick={() => void copyCommand()}
          disabled={status === 'copying'}
          kind="secondary"
          appearance="outlined"
          size="small"
          className="shrink-0"
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
