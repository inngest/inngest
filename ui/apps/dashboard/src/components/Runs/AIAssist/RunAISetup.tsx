import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';
import { RiArrowRightUpLine } from '@remixicon/react';

import { Button } from '@inngest/components/Button';

import { CopyButton } from '@inngest/components/CopyButton';
import { InlineCode } from '@inngest/components/Code';
import { createMCPClients } from '@inngest/components/MCP/clients';

import { useRunAICopy } from './useRunAICopy';

import type { RunAIAction } from './tracking';

export type SetupClient = 'claude' | 'codex' | 'other';

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
  if (client === 'codex' && productionCloud) {
    return <OpenAISetup endpoint={endpoint} onAction={onAction} />;
  }
  if (client === 'claude' && productionCloud) {
    return <ClaudeSetup onAction={onAction} />;
  }
  const trackingClient =
    client === 'claude'
      ? 'claude_code'
      : client === 'codex'
        ? 'codex_cli'
        : 'other';
  const command =
    createMCPClients(endpoint, false).find((item) => item.id === client)
      ?.snippet.content ?? endpoint;
  return (
    <section aria-label="Connection instructions" className="mt-5 text-xs">
      <p className="text-muted mb-1 text-xs">
        {client === 'other'
          ? 'Add this MCP server URL:'
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
            Open <InlineCode>/mcp</InlineCode>, select{' '}
            <InlineCode>inngest-cloud</InlineCode>, and sign in.{' '}
          </>
        ) : client === 'other' ? (
          <>Sign in with OAuth. </>
        ) : null}
        Choose your account, environment, and read access in your browser.
      </p>
      {client === 'codex' && (
        <p className="text-muted mt-2 leading-5">
          Already have Inngest tools in Codex? Just copy the prompt.
        </p>
      )}
      <a
        href="https://www.inngest.com/docs/ai-dev-tools/mcp#connect-to-inngest-cloud"
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

function ClaudeSetup({ onAction }: { onAction: (event: RunAIAction) => void }) {
  return (
    <section aria-label="Connection instructions" className="mt-5 text-xs">
      <p className="text-muted leading-5">
        Connect using the official Inngest connector. Then paste the prompt
        below.
      </p>
      <Button
        kind="primary"
        appearance="outlined"
        size="small"
        label="Connect Inngest"
        icon={<RiArrowRightUpLine />}
        iconSide="right"
        href="https://claude.ai/directory/inngest"
        target="_blank"
        rel="noopener noreferrer"
        className="mt-2"
        onClick={() => onAction({ action: 'setup_opened', client: 'claude' })}
      />
      <AccordionList
        type="single"
        collapsible
        className="mt-3 rounded-none border-0"
      >
        <AccordionList.Item value="cli">
          <AccordionList.Trigger className="text-muted text-xs [&_button>div]:px-0 [&_button>div]:py-1 [&_button:hover]:bg-transparent [&_button:hover]:text-basis">
            Set up with Claude Code
          </AccordionList.Trigger>
          <AccordionList.Content className="px-0 pb-0 pt-2 text-xs">
            <p className="text-muted mb-1">Run in your terminal:</p>
            <SetupCommand
              code="claude plugin install inngest@anthropic-plugin-directory"
              label="Copy setup command"
              client="claude_code"
              onAction={onAction}
            />
            <p className="text-muted mt-2 leading-5">
              Start a new Claude Code session, open{' '}
              <InlineCode>/mcp</InlineCode>, select{' '}
              <InlineCode>plugin:inngest:inngest-cloud</InlineCode>, and sign
              in. Choose your account, environment, and read access in your
              browser.
            </p>
            <a
              href="https://www.inngest.com/docs/ai-dev-tools/agent-skills#claude-code-plugin"
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
          </AccordionList.Content>
        </AccordionList.Item>
      </AccordionList>
    </section>
  );
}

function OpenAISetup({
  endpoint,
  onAction,
}: {
  endpoint: string;
  onAction: (event: RunAIAction) => void;
}) {
  const command = createMCPClients(endpoint, false).find(
    ({ id }) => id === 'codex',
  )!.snippet.content;
  return (
    <section aria-label="Connection instructions" className="mt-5 text-xs">
      <p className="text-muted leading-5">
        Connect using the official Inngest plugin. Then start a new chat and
        paste the prompt below.
      </p>
      <Button
        kind="primary"
        appearance="outlined"
        size="small"
        label="Connect Inngest"
        icon={<RiArrowRightUpLine />}
        iconSide="right"
        href={chatGPTPluginURL}
        target="_blank"
        rel="noopener noreferrer"
        className="mt-2"
        onClick={() =>
          onAction({ action: 'setup_opened', client: 'chatgpt_codex' })
        }
      />
      <AccordionList
        type="single"
        collapsible
        className="mt-3 rounded-none border-0"
      >
        <AccordionList.Item value="cli">
          <AccordionList.Trigger className="text-muted text-xs [&_button>div]:px-0 [&_button>div]:py-1 [&_button:hover]:bg-transparent [&_button:hover]:text-basis">
            Set up with Codex CLI
          </AccordionList.Trigger>
          <AccordionList.Content className="px-0 pb-0 pt-2 text-xs">
            <p className="text-muted mb-1">Run in your terminal:</p>
            <SetupCommand
              code={`${command}\ncodex mcp login inngest-cloud`}
              label="Copy setup command"
              client="codex_cli"
              onAction={onAction}
            />
            <p className="text-muted mt-2 leading-5">
              Choose your account, environment, and read access in your browser.
            </p>
          </AccordionList.Content>
        </AccordionList.Item>
      </AccordionList>
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
