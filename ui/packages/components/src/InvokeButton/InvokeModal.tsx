import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { CodeBlock } from '@inngest/components/CodeBlock';
import { Modal } from '@inngest/components/Modal';

import { parseCode } from './utils';

const initialCode = JSON.stringify({ data: {} }, null, 2);

type Props = {
  doesFunctionAcceptPayload: boolean;
  isOpen: boolean;
  onCancel: () => void;
  onConfirm: (payload: {
    data: Record<string, unknown>;
    user: Record<string, unknown> | null;
  }) => void;
};

export function InvokeModal({ doesFunctionAcceptPayload, isOpen, onCancel, onConfirm }: Props) {
  const [error, setError] = useState<string>();
  const [rawPayload, setRawPayload] = useState(initialCode);

  function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    try {
      let payload: ReturnType<typeof parseCode>;
      if (doesFunctionAcceptPayload) {
        payload = parseCode(rawPayload);
      } else {
        payload = { data: {}, user: null };
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
      <CodeBlock.Wrapper>
        <CodeBlock
          tab={{
            content: rawPayload,
            language: 'json',
            readOnly: false,
            handleChange: setRawPayload,
          }}
          minLines={10}
        />
      </CodeBlock.Wrapper>
    );
  } else {
    content = (
      <p className="text-basis">
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
          <Button kind="secondary" appearance="outlined" onClick={onCancel} label="Cancel" />
          <Button appearance="solid" kind="primary" label="Invoke Function" type="submit" />
        </Modal.Footer>
      </form>
    </Modal>
  );
}
