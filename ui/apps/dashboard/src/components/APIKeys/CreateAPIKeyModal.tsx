import { useRef, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Modal } from '@inngest/components/Modal';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import {
  EnvironmentSelect,
  useEnvironmentSelection,
} from './EnvironmentSelect';
import { apiKeyErrorMessage } from './errorMessage';
import { RevealKeyCard } from './RevealKeyCard';
import { validateAPIKeyName } from './validation';

const Mutation = graphql(`
  mutation CreateAPIKey($input: CreateAPIKeyInput!) {
    createAPIKey(input: $input) {
      plaintextKey
      apiKey {
        id
        name
        createdAt
        maskedKey
        env {
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
  const [plaintextKey, setPlaintextKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Tracks whether the user closed the modal while a mutation was in flight,
  // so we can drop the plaintext response on the floor instead of stashing it
  // in state (which would leak the key into the next modal-open).
  const cancelledRef = useRef(false);

  const { selectedEnv, setSelectedEnv, envGroups } =
    useEnvironmentSelection(isOpen);
  const [, create] = useMutation(Mutation);

  async function submit() {
    setError(null);
    const nameErr = validateAPIKeyName(name);
    if (nameErr) {
      setError(nameErr);
      return;
    }
    if (!selectedEnv) {
      setError('Select an environment.');
      return;
    }
    const trimmed = name.trim();

    cancelledRef.current = false;
    setIsSubmitting(true);
    try {
      const res = await create(
        { input: { name: trimmed, workspaceID: selectedEnv.id } },
        { additionalTypenames: ['APIKey'] },
      );
      if (cancelledRef.current) {
        return;
      }
      if (res.error) {
        setError(apiKeyErrorMessage(res.error, 'Could not create API key.'));
        return;
      }
      const pt = res.data?.createAPIKey?.plaintextKey;
      if (!pt) {
        setError('Unexpected response from server.');
        return;
      }
      setPlaintextKey(pt);
    } finally {
      if (!cancelledRef.current) {
        setIsSubmitting(false);
      }
    }
  }

  function close() {
    cancelledRef.current = true;
    setName('');
    setSelectedEnv(null);
    setPlaintextKey(null);
    setError(null);
    setIsSubmitting(false);
    onClose();
  }

  const inRevealStep = plaintextKey !== null;

  return (
    <Modal
      className="w-full max-w-xl overflow-visible"
      isOpen={isOpen}
      onClose={close}
    >
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
              <label
                htmlFor="api-key-name"
                className="text-basis text-sm font-medium"
              >
                API key name
              </label>
              <Input
                id="api-key-name"
                placeholder="eg. my-api-key"
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={isSubmitting}
              />
            </div>

            <div className="flex flex-col gap-2">
              <label className="text-basis text-sm font-medium">
                Environment
              </label>
              <EnvironmentSelect
                groups={envGroups}
                value={selectedEnv}
                onChange={setSelectedEnv}
              />
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
