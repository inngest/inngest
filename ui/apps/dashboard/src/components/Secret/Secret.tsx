'use client';

import { useState } from 'react';
import { cn } from '@inngest/components/utils/classNames';

import { CopyButton } from './CopyButton';
import { RevealButton } from './RevealButton';

export type SecretKind = 'event-key' | 'signing-key' | 'webhook-path';

type Props = {
  className?: string;
  kind: SecretKind;
  secret: string;
};

export function Secret({ className, kind, secret }: Props) {
  const [isRevealed, setIsRevealed] = useState(false);

  let value = maskSecret(secret, kind);
  if (isRevealed) {
    value = secret;
  }

  return (
    <div
      className={cn(
        'flex overflow-hidden rounded-md border border-slate-300 bg-slate-50 text-slate-500',
        className
      )}
    >
      <div className="flex grow items-center truncate p-2 font-mono text-sm text-slate-500">
        <span className="grow truncate">{value}</span>
      </div>

      <RevealButton
        className="border-r border-slate-300"
        isRevealed={isRevealed}
        onClick={() => setIsRevealed((prev) => !prev)}
      />

      <CopyButton value={secret} />
    </div>
  );
}

function maskSecret(value: string, kind: SecretKind): string {
  if (value.length < 8) {
    // Invalid secret
    return value.replaceAll(/./g, 'X');
  }

  if (kind === 'event-key') {
    // Mask everything after the 8th character
    return value.substring(0, 7) + 'X'.repeat(value.length - 8);
  }

  if (kind === 'signing-key') {
    // Mask everything after the prefix (e.g. "signkey-prod-")
    return value.replace(
      /^(signkey-[A-Za-z0-9]+-).+$/,
      (match, p1) => p1 + 'X'.repeat(match.length - p1.length)
    );
  }

  // Mask everything after the 8th character of the path (e.g. "/e/12345678")
  return value.replace(/^(\/e\/.{8}).+$/, (match, p1) => p1 + 'X'.repeat(match.length - p1.length));
}
