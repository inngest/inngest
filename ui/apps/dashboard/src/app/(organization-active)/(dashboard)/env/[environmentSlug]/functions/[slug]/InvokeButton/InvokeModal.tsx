import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';

import CodeEditor from '@/components/Textarea/CodeEditor';

const initialCode = { data: {} };

type Props = {
  isOpen: boolean;
  onCancel: () => void;
  onConfirm: (payload: { data: Record<string, unknown> }) => void;
};

export function InvokeModal({ isOpen, onCancel, onConfirm }: Props) {
  const [error, setError] = useState<string>();

  function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);

    try {
      const payload = parseCode(formData.get('code'));
      onConfirm(payload);
      setError(undefined);
    } catch (error) {
      if (!(error instanceof Error)) {
        setError('Unknown error');
        return;
      }

      setError(error.message);
    }
  }

  return (
    <Modal
      className="w-[800px]"
      description="Invoke this function, triggering a function run"
      isOpen={isOpen}
      onClose={onCancel}
      title={<h2 className="mb-4 text-lg font-medium">Invoke Function</h2>}
    >
      <form onSubmit={onSubmit}>
        <div className="border-b border-slate-200 p-6">
          <CodeEditor
            className="rounded-lg bg-slate-900 px-4"
            initialCode={JSON.stringify(initialCode, null, 2)}
            language="json"
            name="code"
          />

          {error && <div className="pt-4 text-red-500">{error}</div>}
        </div>

        <div className="flex flex-row justify-end gap-4 p-6">
          <Button appearance="outlined" btnAction={onCancel} label="Cancel" />

          <Button appearance="solid" kind="primary" label="Invoke Function" type="submit" />
        </div>
      </form>
    </Modal>
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function parseCode(code: unknown): { data: Record<string, unknown> } {
  if (typeof code !== 'string') {
    throw new Error("The payload form field isn't a string");
  }

  let payload: Record<string, unknown>;
  const parsed: unknown = JSON.parse(code);
  if (!isRecord(parsed)) {
    throw new Error('Parsed JSON is not an object');
  }

  payload = parsed;

  let { data } = payload;
  if (data === null) {
    data = {};
  }
  if (!isRecord(data)) {
    throw new Error('The "data" field must be an object or null');
  }

  return { data };
}
