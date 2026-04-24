import { useMemo, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Modal } from '@inngest/components/Modal';
import { Select, type Option } from '@inngest/components/Select/Select';
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
  const [selectedEnv, setSelectedEnv] = useState<Option | null>(null);
  const [plaintextKey, setPlaintextKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const [{ data: envs }] = useEnvironments();
  const [, create] = useMutation(Mutation);

  const envOptions: Option[] = useMemo(
    () =>
      (envs ?? [])
        .filter((e) => !e.isArchived)
        .map((e) => ({ id: e.id, name: e.name })),
    [envs],
  );

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
    if (!selectedEnv) {
      setError('Select an environment.');
      return;
    }

    setIsSubmitting(true);
    try {
      const res = await create(
        { input: { name: trimmed, workspaceID: selectedEnv.id } },
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
    setSelectedEnv(null);
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
                Environment
              </label>
              <Select
                label="Environment"
                isLabelVisible={false}
                value={selectedEnv}
                onChange={(opt) => setSelectedEnv(opt)}
              >
                <Select.Button>
                  <span
                    className={selectedEnv ? 'text-basis' : 'text-disabled'}
                  >
                    {selectedEnv?.name ?? 'Select an environment'}
                  </span>
                </Select.Button>
                <Select.Options>
                  {envOptions.map((opt) => (
                    <Select.Option key={opt.id} option={opt}>
                      {opt.name}
                    </Select.Option>
                  ))}
                </Select.Options>
              </Select>
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
