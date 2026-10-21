import { useEffect, useId, useRef, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { CopyButton } from '@inngest/components/CopyButton';
import { Textarea } from '@inngest/components/Forms/Textarea';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@inngest/components/Popover';
import { IconClaude } from '@inngest/components/icons/ai/Claude';
import { IconOpenAI } from '@inngest/components/icons/ai/OpenAI';
import {
  RiAppsLine,
  RiArrowRightUpLine,
  RiSparklingLine,
} from '@remixicon/react';

import { buildRunDiscoveryPrompt, type RunPromptContext } from './prompt';
import { RunAISetup, type SetupClient } from './RunAISetup';
import { useRunAICopy } from './useRunAICopy';
import type { RunAIAction } from './tracking';

const clients = [
  {
    id: 'claude',
    label: 'Claude Code',
    tracking: 'claude_code',
    Icon: IconClaude,
  },
  { id: 'codex', label: 'Codex CLI', tracking: 'codex_cli', Icon: IconOpenAI },
  { id: 'other', label: 'Other tools', tracking: 'other', Icon: RiAppsLine },
] as const;

export function RunAIAssistPopover({
  context,
  metadataLoading,
  metadataError,
  onRetry,
  onAction,
  endpoint,
}: {
  context: RunPromptContext;
  endpoint: string;
  metadataLoading: boolean;
  metadataError: boolean;
  onRetry: () => void;
  onAction: (event: RunAIAction) => void;
}) {
  const [open, setOpen] = useState(false);
  const [selectedClient, setSelectedClient] = useState<SetupClient | null>(
    null,
  );
  const setupID = useId();
  const [previewTime, setPreviewTime] = useState(() => new Date());
  const viewed = useRef(false);
  const result = buildRunDiscoveryPrompt(context, previewTime);
  const {
    status: copyState,
    copy,
    reset,
  } = useRunAICopy(JSON.stringify([context, metadataLoading, metadataError]));
  useEffect(() => {
    if (!viewed.current) {
      viewed.current = true;
      onAction({ action: 'viewed' });
    }
  }, [onAction]);

  const blocked = metadataLoading || metadataError || Boolean(result.error);
  const message = metadataLoading
    ? 'Preparing prompt…'
    : metadataError
      ? 'Unable to load page context. Retry to prepare a prompt.'
      : result.error;

  async function copyPrompt() {
    if (blocked || copyState === 'copying') return;
    if (!result.prompt) return;
    onAction({ action: 'prompt_copy_clicked' });
    const failure = await copy(result.prompt);
    onAction(
      failure
        ? { action: 'prompt_copy_failed', failure_category: failure }
        : { action: 'prompt_copied' },
    );
  }

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        reset();
        if (next) {
          setPreviewTime(new Date());
          onAction({ action: 'opened' });
        }
      }}
    >
      <PopoverTrigger asChild>
        <Button
          kind="primary"
          appearance="outlined"
          size="small"
          title="Find runs with AI"
          icon={<RiSparklingLine />}
          aria-label="Find runs with AI"
          className="h-[26px] w-[26px] shrink-0 rounded"
        />
      </PopoverTrigger>
      <PopoverContent
        align="start"
        collisionPadding={12}
        aria-label="Find runs with your AI tool"
        className="w-[400px] max-w-[calc(100vw-24px)] p-3"
      >
        <h2 className="text-basis text-sm font-semibold">
          Find runs with your AI tool
        </h2>
        <p className="text-muted mt-1 text-xs">
          Connect your AI tool to Inngest Cloud to find runs and inspect their
          traces.
        </p>
        <div className="mt-3 grid grid-cols-1 gap-1.5 min-[420px]:grid-cols-3">
          {clients.map(({ id, label, tracking, Icon }) => (
            <button
              key={id}
              type="button"
              aria-expanded={selectedClient === id}
              aria-controls={selectedClient === id ? setupID : undefined}
              className={`text-basis hover:bg-canvasSubtle inline-flex items-center justify-center gap-1 whitespace-nowrap rounded border px-2 py-1.5 text-xs ${selectedClient === id ? 'border-contrast bg-canvasSubtle' : 'border-muted bg-canvasBase'}`}
              onClick={() => {
                setSelectedClient(selectedClient === id ? null : id);
                if (selectedClient !== id)
                  onAction({ action: 'setup_clicked', client: tracking });
              }}
            >
              <Icon className="h-4 w-4 shrink-0" />
              {label}
            </button>
          ))}
        </div>
        {selectedClient && (
          <div id={setupID}>
            <RunAISetup
              key={`${selectedClient}:${endpoint}`}
              client={selectedClient}
              endpoint={endpoint}
              onAction={onAction}
            />
          </div>
        )}
        <hr className="border-subtle my-4" />
        <div className="mb-2 flex items-center justify-between gap-2">
          <h3 className="text-basis text-xs font-medium">
            Already connected? Try asking
          </h3>
          <CopyButton
            label="Copy prompt"
            ariaLabel={copyState === 'copied' ? 'Copied' : 'Copy prompt'}
            code={result.prompt}
            isCopying={copyState === 'copied'}
            handleCopyClick={() => void copyPrompt()}
            disabled={blocked || copyState === 'copying'}
            appearance="outlined"
            size="small"
          />
        </div>
        {message ? (
          <p role="status" className="text-muted rounded border p-2 text-xs">
            {message}
          </p>
        ) : (
          <Textarea
            aria-label="Prompt preview"
            readOnly
            value={result.prompt ?? ''}
            rows={4}
            className="block h-24 resize-none overflow-y-auto p-2 font-sans text-xs leading-5"
          />
        )}
        {!message && result.summary && (
          <p className="text-light mt-1 text-[11px] leading-4">
            {result.summary}
          </p>
        )}
        {metadataError && !metadataLoading && (
          <Button
            label="Retry"
            onClick={onRetry}
            kind="secondary"
            size="small"
            className="mt-2"
          />
        )}
        {copyState === 'failed' && (
          <p role="status" className="text-muted mt-1 text-xs">
            Couldn’t copy. Select and copy the prompt above.
          </p>
        )}
        <a
          className="text-link mt-2 inline-flex items-center gap-1 text-xs hover:underline"
          href="https://www.inngest.com/docs/ai-dev-tools/mcp#available-mcp-tools"
          target="_blank"
          rel="noopener noreferrer"
          onClick={() =>
            onAction({ action: 'guide_clicked', guide: 'connection' })
          }
        >
          Available MCP tools
          <RiArrowRightUpLine className="h-3.5 w-3.5" aria-hidden="true" />
        </a>
      </PopoverContent>
    </Popover>
  );
}
