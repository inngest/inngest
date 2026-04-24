import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Modal } from '@inngest/components/Modal';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { useEnvironments } from '@/queries/environments';
import { RevealKeyCard } from './RevealKeyCard';

const Mutation = graphql(`
  mutation CreateAPIKey($input: CreateAPIKeyInput!) {
    createAPIKey(input: $input) {
      plaintextKey
      apiKey {
        id
        name
        createdAt
        maskedKey
        workspace {
          id
          name
        }
      }
    }
  }
`);

type Props = {
  isOpen: boolean;
  onClose: () => void;
};

export function CreateAPIKeyModal({ isOpen, onClose }: Props) {
  const [name, setName] = useState('');
  const [workspaceID, setWorkspaceID] = useState<string>('');
  const [plaintextKey, setPlaintextKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const [{ data: envs }] = useEnvironments();
  const [, create] = useMutation(Mutation);

  async function submit() {
    setError(null);
    const trimmed = name.trim();
    if (!trimmed) {
      setError('Name is required.');
      return;
    }
    if (trimmed.length > 128) {
      setError('Name must be 128 characters or fewer.');
      return;
    }
    if (!workspaceID) {
      setError('Select a workspace.');
      return;
    }

    setIsSubmitting(true);
    try {
      const res = await create(
        { input: { name: trimmed, workspaceID } },
        { additionalTypenames: ['APIKey'] },
      );
      if (res.error) {
        setError(res.error.message);
        return;
      }
      const pt = res.data?.createAPIKey?.plaintextKey;
      if (!pt) {
        setError('Unexpected response from server.');
        return;
      }
      setPlaintextKey(pt);
    } finally {
      setIsSubmitting(false);
    }
  }

  function close() {
    setName('');
    setWorkspaceID('');
    setPlaintextKey(null);
    setError(null);
    onClose();
  }

  const inRevealStep = plaintextKey !== null;

  return (
    <Modal className="w-full max-w-xl" isOpen={isOpen} onClose={close}>
      <Modal.Header>
        {inRevealStep ? 'Copy your API key' : 'Create API key'}
      </Modal.Header>

      <Modal.Body>
        {inRevealStep ? (
          <RevealKeyCard plaintextKey={plaintextKey} />
        ) : (
          <div className="flex flex-col gap-5">
            <p className="text-subtle text-sm">
              Generate an API key to give your applications secure access to
              Inngest. You can remove keys at any time.
            </p>

            <div className="flex flex-col gap-2">
              <label className="text-basis text-sm font-medium">
                API Key Name
              </label>
              <Input
                placeholder="eg. my-api-key"
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={isSubmitting}
              />
            </div>

            <div className="flex flex-col gap-2">
              <label className="text-basis text-sm font-medium">
                Workspace
              </label>
              <select
                className="border-muted text-basis bg-canvasBase focus:border-active h-8 rounded border px-2 text-sm outline-none"
                value={workspaceID}
                onChange={(e) => setWorkspaceID(e.target.value)}
                disabled={isSubmitting}
              >
                <option value="" disabled>
                  Select a workspace
                </option>
                {(envs ?? [])
                  .filter((e) => !e.isArchived)
                  .map((env) => (
                    <option key={env.id} value={env.id}>
                      {env.name}
                    </option>
                  ))}
              </select>
            </div>

            {error && <Alert severity="error">{error}</Alert>}
          </div>
        )}
      </Modal.Body>

      <Modal.Footer>
        <div className="flex justify-end gap-2">
          {inRevealStep ? (
            <Button kind="primary" label="Done" onClick={close} />
          ) : (
            <>
              <Button
                appearance="outlined"
                kind="secondary"
                label="Cancel"
                onClick={close}
                disabled={isSubmitting}
              />
              <Button
                kind="primary"
                label="Generate key"
                onClick={submit}
                loading={isSubmitting}
                disabled={isSubmitting}
              />
            </>
          )}
        </div>
      </Modal.Footer>
    </Modal>
  );
}
