import { useEffect, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { createMCPClients } from '@inngest/components/MCP/clients';

import type { SetupClient } from './setup';
import type { RunAIAction } from './tracking';

export function RunAISetup({
  client,
  endpoint,
  fullSetupHref,
  onAction,
}: {
  client: SetupClient;
  endpoint: string;
  fullSetupHref: string;
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
    <section
      aria-label="Connection instructions"
      className="border-subtle mt-2 space-y-2 rounded border p-2 text-xs"
    >
      <p className="text-basis font-medium">
        {client === 'claude'
          ? 'Connect Claude Code'
          : client === 'codex'
            ? 'Connect Codex CLI'
            : 'Connect another AI tool'}
      </p>
      <p className="text-muted">
        {client === 'other'
          ? '1. Add a remote MCP server in your tool’s settings using this URL:'
          : '1. Run this command in your terminal:'}
      </p>
      <SetupCommand
        code={setup?.snippet.content ?? endpoint}
        label={client === 'other' ? 'Copy server URL' : 'Copy setup command'}
        client={trackingClient}
        onAction={onAction}
      />
      {client === 'codex' ? (
        <>
          <p className="text-muted">2. Start browser sign-in:</p>
          <SetupCommand
            code="codex mcp login inngest-cloud"
            label="Copy sign-in command"
            client={trackingClient}
            onAction={onAction}
          />
        </>
      ) : (
        <p className="text-muted">
          {client === 'claude'
            ? '2. In Claude Code, run /mcp, select inngest-cloud, and sign in.'
            : '2. Choose OAuth or Sign in to connect your Inngest account.'}
        </p>
      )}
      <p className="text-muted">
        3. Choose your account and environment with read access, then paste the
        prompt below into your AI tool.
      </p>
      {client === 'claude' && (
        <p className="text-muted">
          Already using the Inngest plugin? Use its Cloud MCP connection.
        </p>
      )}
      <a
        href={fullSetupHref}
        target="_blank"
        rel="noopener noreferrer"
        className="text-link inline-block hover:underline"
        onClick={() =>
          onAction({ action: 'guide_clicked', guide: 'connection' })
        }
      >
        Full setup instructions ↗
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
  const [status, setStatus] = useState<
    'idle' | 'copying' | 'copied' | 'failed'
  >('idle');
  useEffect(() => {
    if (status !== 'copied') return;
    const timer = setTimeout(() => setStatus('idle'), 2000);
    return () => clearTimeout(timer);
  }, [status]);

  async function copy() {
    if (status === 'copying') return;
    setStatus('copying');
    try {
      await navigator.clipboard.writeText(code);
      setStatus('copied');
      onAction({ action: 'setup_copied', client });
    } catch {
      setStatus('failed');
      onAction({ action: 'setup_copy_failed', client });
    }
  }
  return (
    <div>
      <div className="bg-canvasSubtle border-subtle flex items-start gap-2 rounded border p-2">
        <code className="text-basis min-w-0 flex-1 select-text break-words font-mono text-xs">
          {code}
        </code>
        <Button
          aria-label={label}
          label={status === 'copied' ? 'Copied' : 'Copy'}
          onClick={() => void copy()}
          disabled={status === 'copying'}
          kind="secondary"
          appearance="outlined"
          size="small"
          className="shrink-0"
        />
      </div>
      <p role="status" className="text-muted mt-1 text-xs">
        {status === 'failed'
          ? 'Couldn’t copy. Select and copy the text above.'
          : status === 'copied'
            ? 'Copied.'
            : ''}
      </p>
    </div>
  );
}
