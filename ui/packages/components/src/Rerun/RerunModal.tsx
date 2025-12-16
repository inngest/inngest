import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { RiCloseLine } from '@remixicon/react';

import { Button } from '../Button';
import { CodeBlock } from '../CodeBlock/CodeBlock';
import { Modal } from '../Modal/Modal';
import { useRerunFromStep } from '../SharedContext/useRerunFromStep';

export type RerunModalType = {
  open: boolean;
  setOpen: (open: boolean) => void;
  runID: string;
  debugRunID?: string;
  debugSessionID?: string;
  stepID: string;
  input: string;
  redirect?: boolean;
};

export type RerunResult = {
  data?: {
    rerun: unknown;
  };
  error?: unknown;
};

//
// patching in support fo step.ai.infer input bodies
const patchInput = (newInput: string) => {
  try {
    const parsed = JSON.parse(newInput);
    return parsed instanceof Array ? JSON.stringify(parsed) : JSON.stringify([...[parsed]]);
  } catch (e) {
    console.warn('Unable to parse rerun input as JSON');
    return newInput;
  }
};

export const RerunModal = ({
  open,
  setOpen,
  runID,
  stepID,
  input,
  debugRunID,
  debugSessionID,
  redirect = true,
}: RerunModalType) => {
  const { rerun } = useRerunFromStep();
  const [newInput, setNewInput] = useState(input);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const router = useRouter();

  const close = () => {
    setLoading(false);
    setError(null);
    setOpen(false);
  };

  useEffect(() => {
    setNewInput(input);
  }, [input]);

  return (
    <Modal className="flex max-w-[1200px] flex-col p-6" isOpen={open} onClose={close}>
      <div className="mb-6 flex flex-row items-center justify-between gap-6">
        <div className="flex flex-col gap-2">
          <span className="text-basis text-xl">Rerun from step </span>
          <span className="text-subtle text-sm">
            Rerun from step using a different input. A rerun step will appear as a new run. Previous
            steps will be auto-populated with current run&apos;s data and subsequent steps will be
            rerun.
          </span>
        </div>
        <RiCloseLine className="text-subtle h-5 w-5 cursor-pointer" onClick={close} />
      </div>

      <div className="bg-canvasSubtle flex h-full w-full flex-row items-start">
        <div className="h-full w-full">
          <CodeBlock
            actions={[]}
            header={{
              title: 'Previous Input',
            }}
            tab={{
              content: input,
            }}
          />
        </div>

        <div className="h-full w-full">
          <CodeBlock
            header={{
              title: 'Input',
            }}
            tab={{
              content: input,
              readOnly: false,
              handleChange: setNewInput,
            }}
          />
        </div>
      </div>
      <div className="mt-6 flex flex-row items-center justify-end gap-2">
        <div>{error && <span className="text-error">{error.message}</span>}</div>
        <Button kind="secondary" appearance="ghost" label="Cancel" onClick={() => setOpen(false)} />
        <Button
          label="Rerun function"
          loading={loading}
          disabled={loading}
          onClick={async () => {
            setLoading(true);
            const result = await rerun({
              runID,
              debugRunID,
              debugSessionID,
              fromStep: { stepID, ...(newInput ? { input: patchInput(newInput) } : {}) },
            });

            setLoading(false);

            if (result.error) {
              console.error('rerun from step error', result.error);
              setError(result.error);
              return;
            }

            if (redirect && result.redirect) {
              router.push(result.redirect);
            }

            close();
          }}
        />
      </div>
    </Modal>
  );
};
