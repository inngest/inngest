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
  const prettyInput = usePrettyJson(result.input ?? '') || (result.input ?? '');
  const prettyOutput = usePrettyJson(result.data ?? '') || (result.data ?? '');

  return (
    <div className={className}>
      {result.input && (
        <CodeBlock
          header={{
            title: 'Input',
          }}
          tab={{
            content: prettyInput,
          }}
        />
      )}

      {result.data && (
        <CodeBlock
          header={{
            title: 'Output',
            status: isSuccess ? 'success' : undefined,
          }}
          tab={{
            content: prettyOutput,
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
