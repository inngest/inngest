import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { RiCloseLine } from '@remixicon/react';

import { NewButton } from '../Button';
import { CodeBlock } from '../CodeBlock/CodeBlock';
import { Modal } from '../Modal/Modal';

export type RerunModalType = {
  open: boolean;
  setOpen: (open: boolean) => void;
  runID: string;
  stepID: string;
  input: string;
  rerunFromStep: (args: {
    runID: string;
    fromStep: { stepID: string; input: string };
  }) => Promise<RerunResult>;
};

export type RerunResult = {
  data?: {
    rerun: Record<string, unknown>;
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
  rerunFromStep,
}: RerunModalType) => {
  const [newInput, setNewInput] = useState(input);
  const router = useRouter();
  const [rerunning, setRerunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      setError(null);
    }
  }, [open]);

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
        <RiCloseLine
          className="text-subtle h-5 w-5 cursor-pointer"
          onClick={() => setOpen(false)}
        />
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
        <div>{error && <span className="text-error">{error}</span>}</div>
        <NewButton
          kind="secondary"
          appearance="ghost"
          label="Cancel"
          onClick={() => setOpen(false)}
        />
        <NewButton
          label="Rerun function"
          loading={rerunning}
          onClick={async () => {
            setRerunning(true);
            setError(null);

            const result = await rerunFromStep({
              runID,
              fromStep: { stepID, input: patchInput(newInput) },
            });

            if (result.error) {
              console.error('rerun from step error', result.error);
              setError('Rerun failed, please try again later.');
              setRerunning(false);
            }

            if (result.data?.rerun) {
              router.push(`/run?runID=${result.data.rerun}`);
            }
          }}
        />
      </div>
    </Modal>
  );
};
