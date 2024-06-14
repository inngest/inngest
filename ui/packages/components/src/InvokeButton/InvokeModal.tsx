import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { CodeBlock } from '@inngest/components/CodeBlock';
import { Modal } from '@inngest/components/Modal';

const initialCode = JSON.stringify({ data: {} }, null, 2);

type Props = {
  doesFunctionAcceptPayload: boolean;
  isOpen: boolean;
  onCancel: () => void;
  onConfirm: (payload: { data: Record<string, unknown> }) => void;
};

export function InvokeModal({ doesFunctionAcceptPayload, isOpen, onCancel, onConfirm }: Props) {
  const [error, setError] = useState<string>();
  const [rawPayload, setRawPayload] = useState(initialCode);

  function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    try {
      let payload;
      if (doesFunctionAcceptPayload) {
        payload = parseCode(rawPayload);
      } else {
        payload = { data: {} };
      }

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
  if (doesFunctionAcceptPayload) {
    content = (
      <CodeBlock
        tabs={[
          {
            content: rawPayload,
            language: 'json',
            readOnly: false,
            handleChange: setRawPayload,
          },
        ]}
        minLines={10}
      />
    );
  } else {
    content = (
      <p className="dark:text-white">
        Cron functions without event triggers cannot include payload data.
      </p>
    );
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

        <Modal.Footer className="flex justify-end gap-2">
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

function parseCode(code: string): { data: Record<string, unknown> } {
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

  const supportedKeys = ['data'];
  for (const key of Object.keys(payload)) {
    if (!supportedKeys.includes(key)) {
      throw new Error(`Property "${key}" is not supported when invoking a function`);
    }
  }

  return { data };
}
