import { useEffect, useState } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { RiCloseLine } from '@remixicon/react';

import { Button } from '../Button';
import { CodeBlock } from '../CodeBlock/CodeBlock';
import { Modal } from '../Modal/Modal';
import { useRerunFromStep } from '../Shared/useRerunFromStep';

export type RerunModalType = {
  open: boolean;
  setOpen: (open: boolean) => void;
  runID: string;
  stepID: string;
  input: string;
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

export const RerunModal = ({ open, setOpen, runID, stepID, input }: RerunModalType) => {
  const { rerun, loading, error } = useRerunFromStep();
  const [newInput, setNewInput] = useState(input);
  const [rerunning, setRerunning] = useState(false);
  const router = useRouter();

  const pathname = usePathname();
  const parts = pathname.trim().split('/').slice(1);

  useEffect(() => {
    if (!open) {
      setRerunning(false);
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
        <div>{error && <span className="text-error">{error.message}</span>}</div>
        <Button kind="secondary" appearance="ghost" label="Cancel" onClick={() => setOpen(false)} />
        <Button
          label="Rerun function"
          loading={loading || rerunning}
          disabled={loading || rerunning}
          onClick={async () => {
            setRerunning(true);
            const result = await rerun({
              runID,
              fromStep: { stepID, input: patchInput(newInput) },
            });

            if (error) {
              console.error('rerun from step error', error);
            }

            if (result?.data?.rerun) {
              router.push(
                parts[0] === 'env'
                  ? `/${parts[0]}/${parts[1]}/runs/${result.data.rerun}`
                  : `/run?runID=${result.data.rerun}`
              );
            }
          }}
        />
      </div>
    </Modal>
  );
};
