import { Link } from '@inngest/components/Link';
import { useEffect, useId, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Textarea } from '@inngest/components/Forms/Textarea';
import { RiCheckLine, RiSparklingLine } from '@remixicon/react';
import { useQuickstart } from './QuickstartProvider';
import { createOnboardingPrompt } from './prompt';
import { useQuickstartTracking } from './useQuickstartTracking';

export function AgentSetup({ server }: { server: string }) {
  const { attempt, setAttempt } = useQuickstart();
  const track = useQuickstartTracking(server);
  const [expanded, setExpanded] = useState(false);
  const [prompt, setPrompt] = useState(() => createOnboardingPrompt(server));
  const [copyState, setCopyState] = useState<
    'idle' | 'copying' | 'copied' | 'failed'
  >('idle');
  const promptID = useId();
  useEffect(() => {
    if (copyState !== 'copied') return;
    const timer = setTimeout(() => setCopyState('idle'), 2000);
    return () => clearTimeout(timer);
  }, [copyState]);

  async function copyPrompt() {
    if (copyState === 'copying') return;
    setCopyState('copying');
    try {
      await navigator.clipboard.writeText(prompt);
      setCopyState('copied');
      track('agent_prompt_copied');
      if (!attempt)
        setAttempt({
          mode: 'agent',
          functionID: '',
          startedAt: new Date().toISOString(),
        });
    } catch {
      setCopyState('failed');
      setExpanded(true);
    }
  }

  return (
    <section
      className="border-subtle bg-canvasBase space-y-3 rounded border p-4"
      aria-label="Set up with a coding agent"
    >
      <div className="flex flex-wrap items-center gap-4">
        <div className="bg-canvasSubtle flex h-16 w-16 shrink-0 items-center justify-center rounded-md">
          <img
            src="/images/onboarding-monitor.png"
            alt=""
            width={46}
            height={36}
            className="h-9 w-auto"
            style={{ imageRendering: 'pixelated' }}
          />
        </div>
        <div className="min-w-0 flex-[1_1_180px]">
          <h2 className="font-medium">Set up with a coding agent</h2>
          <p className="text-muted text-sm">
            Paste into your AI editor, or follow the{' '}
            <Link
              href="https://www.inngest.com/docs/ai-dev-tools/mcp#connect-to-the-dev-server"
              onClick={() => track('docs_clicked', { guide: 'mcp' })}
              className="inline-flex align-baseline whitespace-nowrap"
            >
              MCP guide
            </Link>
          </p>
          <div className="-ml-2 mt-1">
            <Button
              kind="primary"
              appearance="ghost"
              size="small"
              label={expanded ? 'Hide prompt' : 'View prompt'}
              aria-expanded={expanded}
              aria-controls={promptID}
              onClick={() => setExpanded(!expanded)}
            />
          </div>
        </div>
        <div className="shrink-0">
          <Button
            className="min-w-[116px]"
            label={copyState === 'copied' ? 'Copied!' : 'Copy prompt'}
            aria-label="Copy onboarding prompt"
            icon={
              copyState === 'copied' ? <RiCheckLine /> : <RiSparklingLine />
            }
            iconSide="left"
            disabled={!prompt.trim() || copyState === 'copying'}
            onClick={copyPrompt}
          />
        </div>
      </div>
      {expanded && (
        <div id={promptID}>
          <Textarea
            aria-label="Onboarding prompt"
            value={prompt}
            onChange={(value) => {
              setPrompt(value);
              setCopyState('idle');
            }}
            rows={6}
            className="bg-disabled text-basis border-transparent font-mono text-xs"
          />
        </div>
      )}
      {copyState === 'failed' && (
        <p role="status" className="text-muted text-xs">
          Couldn’t copy. Select and copy the prompt above.
        </p>
      )}
    </section>
  );
}
