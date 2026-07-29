import { useEffect, useState } from 'react';
import { RiCloseLine } from '@remixicon/react';
import { useNavigate } from '@tanstack/react-router';

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
  editableInput?: boolean;
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
  editableInput = true,
  redirect = true,
}: RerunModalType) => {
  const { rerun } = useRerunFromStep();
  const [newInput, setNewInput] = useState(input);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const navigate = useNavigate();

  const close = () => {
    setLoading(false);
    setError(null);
    setOpen(false);
  };

  useEffect(() => {
    setNewInput(input);
  }, [input]);

  return (
    <Modal
      className={`relative flex w-full flex-col p-6 ${
        editableInput ? 'max-w-[1200px]' : 'max-w-lg'
      }`}
      isOpen={open}
      onClose={close}
    >
      <RiCloseLine
        className="text-subtle absolute right-6 top-6 h-6 w-6 cursor-pointer"
        onClick={close}
      />
      <div className="mb-6 flex flex-row items-start justify-between gap-6">
        <div className="flex flex-col gap-2 pr-10">
          <span className="text-basis text-xl">Rerun from step </span>
          <span className="text-subtle text-sm">
            {editableInput
              ? 'Rerun from step using a different input. '
              : 'This step type does not support input overrides. '}
            A rerun step will appear as a new run. Previous steps will be auto-populated with
            current run&apos;s data and subsequent steps will be rerun.
          </span>
        </div>
      </div>
      {editableInput && (
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
                readOnly: !editableInput,
                handleChange: editableInput ? setNewInput : undefined,
              }}
            />
          </div>
        </div>
      )}
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
              fromStep: {
                stepID,
                ...(editableInput && newInput ? { input: patchInput(newInput) } : {}),
              },
            });

            setLoading(false);

            if (result.error) {
              console.error('rerun from step error', result.error);
              setError(result.error);
              return;
            }

            if (redirect && result.redirect) {
              navigate({ to: result.redirect });
            }

            close();
          }}
        />
      </div>
    </Modal>
  );
};
