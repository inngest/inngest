'use client';

import { useState } from 'react';
import { usePrettyJson } from '@inngest/components/hooks/usePrettyJson';

import { Button } from './Button';
import { CodeBlock } from './CodeBlock';
import { RerunModal } from './Rerun/RerunModal';
import type { Result } from './types/functionRun';

type Props = {
  className?: string;
  result: Result;
  runID: string;
  stepID?: string | null;
  isSuccess?: boolean;
};

export function RunResult({ result, isSuccess, runID, stepID }: Props) {
  const prettyInput = usePrettyJson(result.input ?? '') || (result.input ?? '');
  const prettyOutput = usePrettyJson(result.data ?? '') || (result.data ?? '');
  const [rerunModalOpen, setRerunModalOpen] = useState(false);

  return (
    <div className="flex flex-col">
      {result.input && (
        <div className="bg-canvasBase border-l-primary-moderate border-subtle border-r-hidden h-11 w-full border border-l px-6 py-3 text-sm font-normal leading-tight">
          Content
        </div>
      )}
      <div className="bg-canvasSubtle flex">
        {result.input && (
          <div className="border-r-subtle flex w-full flex-col justify-between border-r">
            <CodeBlock
              header={{
                title: 'Input',
              }}
              tab={{
                content: prettyInput,
              }}
            />

            {runID && stepID && (
              <>
                <Button
                  className="m-2 w-40"
                  label="Rerun with new prompt"
                  onClick={() => setRerunModalOpen(true)}
                />
                <RerunModal
                  open={rerunModalOpen}
                  setOpen={setRerunModalOpen}
                  runID={runID}
                  stepID={stepID}
                  input={prettyInput}
                />
              </>
            )}
          </div>
        )}
        {result.data && (
          <div className="w-full">
            <CodeBlock
              header={{
                title: 'Output',
                status: isSuccess ? 'success' : undefined,
              }}
              tab={{
                content: prettyOutput,
              }}
            />
          </div>
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
  );
}
