import { RiArrowRightUpLine } from '@remixicon/react';

import { CopyButton } from '@inngest/components/CopyButton';
import { InlineCode } from '@inngest/components/Code';
import { createMCPClients } from '@inngest/components/MCP/clients';

import { useRunAICopy } from './useRunAICopy';

import type { RunAIAction } from './tracking';

export type SetupClient = 'chatgpt' | 'claude' | 'codex' | 'other';

const chatGPTPluginURL =
  'https://chatgpt.com/plugins/plugin_asdk_app_6abe99ad4764819183b795f0d74e5216';

export function RunAISetup({
  client,
  endpoint,
  onAction,
}: {
  client: SetupClient;
  endpoint: string;
  onAction: (event: RunAIAction) => void;
}) {
  const productionCloud = endpoint === 'https://api.inngest.com/mcp';
  if (client === 'chatgpt') {
    if (!productionCloud) return null;
    return (
      <section aria-label="Connection instructions" className="mt-5 text-xs">
        <p className="text-muted mb-2 leading-5">
          Install the Inngest plugin and connect your account. Then paste the
          prompt below into ChatGPT.
        </p>
        <a
          href={chatGPTPluginURL}
          target="_blank"
          rel="noopener noreferrer"
          className="text-link inline-flex items-center gap-1 hover:underline"
          onClick={() =>
            onAction({ action: 'setup_opened', client: 'chatgpt' })
          }
        >
          Open in ChatGPT
          <RiArrowRightUpLine className="h-3.5 w-3.5" aria-hidden="true" />
        </a>
      </section>
    );
  }
  const claudePlugin = client === 'claude' && productionCloud;
  const trackingClient =
    client === 'claude'
      ? 'claude_code'
      : client === 'codex'
        ? 'codex_cli'
        : 'other';
  const command = claudePlugin
    ? 'claude plugin install inngest@anthropic-plugin-directory'
    : (createMCPClients(endpoint, false).find((item) => item.id === client)
        ?.snippet.content ?? endpoint);
  return (
    <section aria-label="Connection instructions" className="mt-5 text-xs">
      <p className="text-muted mb-1 text-xs">
        {client === 'other'
          ? 'Add this MCP server URL:'
          : claudePlugin
            ? 'Install the Inngest plugin:'
            : 'Run in your terminal:'}
      </p>
      <SetupCommand
        code={
          client === 'codex'
            ? `${command}\ncodex mcp login inngest-cloud`
            : command
        }
        label={client === 'other' ? 'Copy server URL' : 'Copy setup command'}
        client={trackingClient}
        onAction={onAction}
      />
      <p className="text-muted mt-2 leading-5">
        {client === 'claude' ? (
          <>
            {claudePlugin ? 'Start a new Claude Code session, open ' : 'Open '}
            <InlineCode>/mcp</InlineCode>, select{' '}
            <InlineCode>
              {claudePlugin ? 'plugin:inngest:inngest-cloud' : 'inngest-cloud'}
            </InlineCode>
            , and sign in.{' '}
          </>
        ) : client === 'other' ? (
          <>Sign in with OAuth. </>
        ) : null}
        Choose your account, environment, and read access in your browser.
      </p>
      {claudePlugin && (
        <p className="text-muted mt-2 leading-5">
          Already using the plugin or Cloud MCP? Use that connection—no extra
          setup needed.
        </p>
      )}
      {client === 'codex' && (
        <p className="text-muted mt-2 leading-5">
          Already have Inngest tools in Codex? Just copy the prompt.
        </p>
      )}
      <a
        href={
          claudePlugin
            ? 'https://www.inngest.com/docs/ai-dev-tools/agent-skills#claude-code-plugin'
            : 'https://www.inngest.com/docs/ai-dev-tools/mcp#connect-to-inngest-cloud'
        }
        target="_blank"
        rel="noopener noreferrer"
        className="text-link mt-3 inline-flex items-center gap-1 hover:underline"
        onClick={() =>
          onAction({ action: 'guide_clicked', guide: 'connection' })
        }
      >
        Full setup guide
        <RiArrowRightUpLine className="h-3.5 w-3.5" aria-hidden="true" />
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
        <code className="text-basis min-w-0 flex-1 select-text whitespace-pre-wrap break-words font-mono text-[11px] leading-4">
          {code}
        </code>
        <CopyButton
          ariaLabel={label}
          code={code}
          appearance="outlined"
          className="shrink-0"
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
