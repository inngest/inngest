'use client';

import { CodeBlock } from './CodeBlock';
import type { Result } from './types/functionRun';

type Props = {
  className?: string;
  result: Result;
};

export function RunResult({ className, result }: Props) {
  return (
    <div className={className}>
      {result.data && (
        <CodeBlock
          tabs={[
            {
              label: 'Output',
              content: result.data,
            },
          ]}
        />
      )}

      {result.error?.stack && (
        <CodeBlock
          header={{
            color: 'bg-rose-50',
            description: result.error.message,
            title: result.error.name ?? 'Error',
          }}
          tabs={[
            {
              label: 'Stack',
              content: result.error.stack,
            },
          ]}
        />
      )}
    </div>
  );
}
