import { useEffect, useState } from 'react';
import { CopyButton } from '@inngest/components/CopyButton';

// Keep onboarding's inline feedback local; other copy controls retain their notifications.
export function QuietCopyButton({
  content,
  onCopy,
}: {
  content?: string;
  onCopy?: () => void;
}) {
  const [state, setState] = useState<'idle' | 'copying' | 'copied' | 'failed'>(
    'idle',
  );

  useEffect(() => {
    if (state !== 'copied') return;
    const timeout = setTimeout(() => setState('idle'), 1000);
    return () => clearTimeout(timeout);
  }, [state]);

  async function copy(code: string) {
    setState('copying');
    onCopy?.();
    try {
      await navigator.clipboard.writeText(code);
      setState('copied');
    } catch {
      setState('failed');
    }
  }

  return (
    <>
      <CopyButton
        size="small"
        code={content}
        disabled={state === 'copying'}
        isCopying={state === 'copied'}
        handleCopyClick={copy}
        appearance="outlined"
      />
      {state === 'failed' && (
        <span role="status" className="text-muted ml-2 text-xs">
          Couldn’t copy. Select the text to copy it.
        </span>
      )}
    </>
  );
}
