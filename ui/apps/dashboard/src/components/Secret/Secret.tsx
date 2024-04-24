import { useState } from 'react';
import { cn } from '@inngest/components/utils/classNames';

import { CopyButton } from './CopyButton';
import { RevealButton } from './RevealButton';

type SecretKind = 'signing-key';

type Props = {
  className?: string;
  kind?: SecretKind;
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
        'flex rounded-md border border-slate-300 bg-slate-50 text-slate-500',
        className
      )}
    >
      <div className="flex grow items-center truncate border-r border-slate-300 p-2 text-slate-500">
        <span className="grow truncate font-mono text-sm">{value}</span>
        <RevealButton isRevealed={isRevealed} onClick={() => setIsRevealed((prev) => !prev)} />
      </div>

      <CopyButton value={secret} />
    </div>
  );
}

function maskSecret(value: string, kind: SecretKind | undefined): string {
  if (kind === 'signing-key') {
    return value.replace(
      /^(signkey-[A-Za-z0-9]+-).+$/,
      (match, p1) => p1 + 'X'.repeat(match.length - p1.length)
    );
  }

  return value.replaceAll(/./g, 'X');
}
