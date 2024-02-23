import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';

import { Alert } from '@/components/Alert';
import CodeEditor from '@/components/Textarea/CodeEditor';

const initialCode = { data: {} };

type Props = {
  doesFunctionAcceptPayload: boolean;
  isOpen: boolean;
  onCancel: () => void;
  onConfirm: (payload: { data: Record<string, unknown> }) => void;
};

export function InvokeModal({
  doesFunctionAcceptPayload: hasEventTrigger,
  isOpen,
  onCancel,
  onConfirm,
}: Props) {
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

  let content;
  if (hasEventTrigger) {
    content = (
      <CodeEditor
        className="rounded-lg bg-slate-900 px-4"
        initialCode={JSON.stringify(initialCode, null, 2)}
        language="json"
        name="code"
      />
    );
  } else {
    content = <p>Cron functions without event triggers cannot include payload data.</p>;
  }

  return (
    <Modal className="w-full max-w-3xl" isOpen={isOpen} onClose={onCancel}>
      <Modal.Header description="Invoke this function, triggering a function run">
        Invoke Function
      </Modal.Header>

      <form onSubmit={onSubmit}>
        <Modal.Body>
          {content}

          {error && (
            <Alert className="mt-6" severity="error">
              {error}
            </Alert>
          )}
        </Modal.Body>

        <Modal.Footer className="flex flex-row justify-end gap-4">
          <Button appearance="outlined" btnAction={onCancel} label="Cancel" />
          <Button appearance="solid" kind="primary" label="Invoke Function" type="submit" />
        </Modal.Footer>
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
