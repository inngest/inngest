import { useState } from 'react';
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
  }) => Promise<unknown>;
};
export const RerunModal = ({
  open,
  setOpen,
  runID,
  stepID,
  input,
  rerunFromStep,
}: RerunModalType) => {
  const [newInput, setNewInput] = useState('');
  const router = useRouter();

  return (
    <Modal
      className="flex max-w-[1200px] flex-col p-6"
      isOpen={open}
      onClose={() => setOpen(false)}
    >
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
      <div className="mt-6 flex flex-row justify-end gap-2">
        <NewButton
          kind="secondary"
          appearance="ghost"
          label="Cancel"
          onClick={() => setOpen(false)}
        />
        <NewButton
          label="Rerun function"
          onClick={async () => {
            const result = await rerunFromStep({ runID, fromStep: { stepID, input: newInput } });
            if (result) {
              router.push(`/run?runID=${result}`);
            }
          }}
        />
      </div>
    </Modal>
  );
};
