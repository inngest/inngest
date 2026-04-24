import { Alert } from '@inngest/components/Alert';

import { CopyButton } from '@/components/Secret/CopyButton';

type Props = {
  plaintextKey: string;
};

export function RevealKeyCard({ plaintextKey }: Props) {
  return (
    <div className="flex flex-col gap-3">
      <Alert severity="warning">
        This is the only time you will see this key. Copy it now — if you lose
        it, delete it and create a new one.
      </Alert>
      <div className="border-subtle bg-canvasSubtle flex items-center gap-2 overflow-hidden rounded-md border">
        <code className="text-basis flex-1 break-all p-3 font-mono text-sm">
          {plaintextKey}
        </code>
        <CopyButton value={plaintextKey} />
      </div>
    </div>
  );
}
