import { useCallback, useEffect, useId, useRef, useState } from 'react';
import { Button } from '@inngest/components/Button';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@inngest/components/Popover';
import { IconClaude } from '@inngest/components/icons/ai/Claude';
import { IconOpenAI } from '@inngest/components/icons/ai/OpenAI';
import { RiAppsLine, RiSparklingLine } from '@remixicon/react';

import { buildRunDiscoveryPrompt, type RunPromptContext } from './prompt';
import { RunAISetup } from './RunAISetup';
import type { SetupClient } from './setup';
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
  getSetupHref,
  endpoint,
}: {
  context: RunPromptContext;
  endpoint: string;
  metadataLoading: boolean;
  metadataError: boolean;
  onRetry: () => void;
  onAction: (event: RunAIAction) => void;
  getSetupHref: (client: SetupClient) => string;
}) {
  const [open, setOpen] = useState(false);
  const [selectedClient, setSelectedClient] = useState<SetupClient | null>(
    null,
  );
  const setupID = useId();
  const [previewTime, setPreviewTime] = useState(() => new Date());
  const [copyState, setCopyState] = useState<
    'idle' | 'copying' | 'copied' | 'failed'
  >('idle');
  const triggerRef = useRef<HTMLButtonElement>(null);
  const viewed = useRef(false);
  const copyAttempt = useRef(0);
  const result = buildRunDiscoveryPrompt(context, previewTime);
  const contextKey = JSON.stringify(context);

  useEffect(() => {
    copyAttempt.current++;
    setCopyState('idle');
  }, [contextKey, metadataLoading, metadataError]);

  useEffect(() => {
    if (copyState !== 'copied') return;
    const timer = setTimeout(() => setCopyState('idle'), 2000);
    return () => clearTimeout(timer);
  }, [copyState]);

  const recordView = useCallback(() => {
    if (!viewed.current) {
      viewed.current = true;
      onAction({ action: 'viewed' });
    }
  }, [onAction]);

  useEffect(() => {
    const trigger = triggerRef.current;
    if (!trigger || viewed.current) return;
    if (typeof IntersectionObserver === 'undefined') {
      recordView();
      return;
    }
    const observer = new IntersectionObserver((entries) => {
      if (entries.some((entry) => entry.isIntersecting)) {
        recordView();
        observer.disconnect();
      }
    });
    observer.observe(trigger);
    return () => observer.disconnect();
  }, [recordView]);

  const blocked = metadataLoading || metadataError || Boolean(result.error);
  const message = metadataLoading
    ? 'Preparing prompt…'
    : metadataError
      ? 'Unable to load page context. Retry to prepare a prompt.'
      : result.error;

  async function copyPrompt() {
    if (blocked || copyState === 'copying') return;
    const now = new Date();
    const snapshot = buildRunDiscoveryPrompt(context, now);
    setPreviewTime(now);
    if (!snapshot.prompt) return;
    const attempt = ++copyAttempt.current;
    setCopyState('copying');
    onAction({ action: 'prompt_copy_clicked' });
    try {
      if (!navigator.clipboard?.writeText) {
        onAction({
          action: 'prompt_copy_failed',
          failure_category: 'unavailable',
        });
        if (attempt === copyAttempt.current) setCopyState('failed');
        return;
      }
      await navigator.clipboard.writeText(snapshot.prompt);
      onAction({ action: 'prompt_copied' });
      if (attempt === copyAttempt.current) setCopyState('copied');
    } catch (error) {
      onAction({
        action: 'prompt_copy_failed',
        failure_category:
          error instanceof DOMException && error.name === 'NotAllowedError'
            ? 'permission_denied'
            : 'other',
      });
      if (attempt === copyAttempt.current) setCopyState('failed');
    }
  }

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        setCopyState('idle');
        if (next) {
          recordView();
          setPreviewTime(new Date());
          onAction({ action: 'opened' });
        }
      }}
    >
      <PopoverTrigger asChild>
        <Button
          ref={triggerRef}
          kind="secondary"
          appearance="outlined"
          size="small"
          label="Use AI"
          icon={<RiSparklingLine />}
          iconSide="left"
          aria-label="Find runs with AI"
          className="h-[26px] shrink-0 rounded"
        />
      </PopoverTrigger>
      <PopoverContent
        align="start"
        collisionPadding={12}
        aria-label="Find runs with your AI tool"
        className="w-[400px] max-w-[calc(100vw-24px)] p-3"
      >
        <h2 className="text-basis text-sm font-medium">
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
              fullSetupHref={getSetupHref(selectedClient)}
              onAction={onAction}
            />
          </div>
        )}
        <div className="mb-2 mt-4 flex items-center justify-between gap-2">
          <p className="text-muted text-xs">Already connected? Try asking</p>
          <Button
            label={
              copyState === 'copied'
                ? 'Copied'
                : copyState === 'copying'
                  ? 'Copying…'
                  : 'Copy prompt'
            }
            onClick={() => void copyPrompt()}
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
          <pre
            aria-label="Prompt preview"
            tabIndex={0}
            className="bg-canvasSubtle text-basis max-h-36 select-text overflow-y-auto whitespace-pre-wrap break-words rounded border p-2 font-mono text-xs"
          >
            {result.prompt}
          </pre>
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
        <p role="status" aria-live="polite" className="text-muted mt-1 text-xs">
          {copyState === 'copied'
            ? 'Prompt copied.'
            : copyState === 'failed'
              ? 'Couldn’t copy. Select and copy the prompt above.'
              : ''}
        </p>
        <a
          className="text-link mt-2 inline-block text-xs hover:underline"
          href="https://www.inngest.com/docs/ai-dev-tools/mcp"
          target="_blank"
          rel="noopener noreferrer"
          onClick={() =>
            onAction({ action: 'guide_clicked', guide: 'connection' })
          }
        >
          View MCP docs ↗
        </a>
      </PopoverContent>
    </Popover>
  );
}
