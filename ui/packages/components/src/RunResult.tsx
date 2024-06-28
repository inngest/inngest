'use client';

import { usePrettyJson } from '@inngest/components/hooks/usePrettyJson';

import { CodeBlock } from './CodeBlock';
import type { Result } from './types/functionRun';

type Props = {
  className?: string;
  result: Result;
  isSuccess?: boolean;
};

export function RunResult({ className, result, isSuccess }: Props) {
  const prettyResult = result.data && usePrettyJson(result.data);

  return (
    <div className={className}>
      {result.data && (
        <CodeBlock
          header={{
            title: 'Output',
            status: isSuccess ? 'success' : undefined,
          }}
          tab={{
            content: prettyResult || result.data,
          }}
        />
      )}

      {result.error && (
        <CodeBlock
          header={{
            title:
              (result.error.name || 'Error') +
              (result.error.message ? ': ' + result.error.message : ''),
            status: 'error',
          }}
          tab={{
            content: result.error.stack ?? '',
          }}
        />
      )}
    </div>
  );
}
