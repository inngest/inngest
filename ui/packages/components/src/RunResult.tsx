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
          header={{
            title: 'Output',
          }}
          tab={{
            content: result.data,
          }}
        />
      )}

      {result.error?.stack && (
        <CodeBlock
          header={{
            title:
              result.error.name ??
              'Error' + (result.error.message ? ': ' + result.error.message : ''),
            status: 'error',
          }}
          tab={{
            content: result.error.stack,
          }}
        />
      )}
    </div>
  );
}
