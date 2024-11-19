'use client';

import { usePrettyJson } from '@inngest/components/hooks/usePrettyJson';

import { CodeBlock } from './CodeBlock';
import type { Result } from './types/functionRun';

type Props = {
  className?: string;
  result: Result;
  isSuccess?: boolean;
  stepAIEnabled?: boolean;
};

export function RunResult({ className, result, isSuccess, stepAIEnabled = false }: Props) {
  const prettyInput = usePrettyJson(result.input ?? '') || (result.input ?? '');
  const prettyOutput = usePrettyJson(result.data ?? '') || (result.data ?? '');

  return stepAIEnabled ? (
    <div className="flex flex-col">
      <div className="border-b-subtle border-t-subtle bg-canvasBase border-primary-moderate h-11 w-full border-b border-l border-t px-6 py-3 text-sm font-normal leading-tight">
        Content
      </div>
      <div className="flex h-full w-full flex-row">
        {result.input && (
          <CodeBlock
            className="w-full"
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
            className="w-full"
            header={{
              title: 'Output',
              status: isSuccess ? 'success' : undefined,
            }}
            tab={{
              content: prettyOutput,
            }}
          />
        )}
      </div>
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
  ) : (
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
