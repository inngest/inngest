import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';

import { CopyButton } from '@/components/Secret/CopyButton';
import { RevealButton } from '@/components/Secret/RevealButton';

type Props = {
  plaintextKey: string;
};

// maskKey renders the visible prefix plus fixed-width dots, matching the
// maskedKey format used in the list ("sk-inn-api••••<preview>"). Hiding the
// full body by default matches how event- and signing-key surfaces behave.
function maskKey(plaintext: string): string {
  const prefixMatch = plaintext.match(/^(sk-inn-[a-z]+-)/);
  const prefix = prefixMatch?.[1] ?? 'sk-inn-api-';
  return prefix + '••••••••••••••••';
}

export function RevealKeyCard({ plaintextKey }: Props) {
  const [isRevealed, setIsRevealed] = useState(false);
  const display = isRevealed ? plaintextKey : maskKey(plaintextKey);

  return (
    <div className="flex flex-col gap-3">
      <Alert severity="warning">
        Keep a record of the key below. You won’t be able to view it again.
      </Alert>
      <div className="border-subtle bg-canvasSubtle flex items-center gap-2 overflow-hidden rounded-md border">
        <code className="text-basis flex-1 break-all p-3 font-mono text-sm">
          {display}
        </code>
        <RevealButton
          className="border-subtle border-l"
          isRevealed={isRevealed}
          onClick={() => setIsRevealed((prev) => !prev)}
        />
        <CopyButton value={plaintextKey} />
      </div>
    </div>
  );
}
