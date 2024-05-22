import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { InlineCode } from '@inngest/components/InlineCode';
import { Modal } from '@inngest/components/Modal';
import { useMutation } from 'urql';

import { graphql } from '@/gql';

const Mutation = graphql(`
  mutation RotateSigningKey($envID: UUID!) {
    rotateSigningKey(envID: $envID) {
      createdAt
    }
  }
`);

type Props = {
  envID: string;
  isOpen: boolean;
  onClose: () => void;
};

export function RotateSigningKeyModal(props: Props) {
  const { envID, isOpen } = props;
  const [error, setError] = useState<string>();
  const [isFetching, setIsFetching] = useState(false);
  const [, rotateSigningKey] = useMutation(Mutation);

  function onClose() {
    setError(undefined);
    props.onClose();
  }

  async function onConfirm() {
    setIsFetching(true);
    try {
      const res = await rotateSigningKey(
        { envID },
        {
          // Bust cache
          additionalTypenames: ['SigningKey'],
        }
      );
      if (res.error) {
        throw res.error;
      }

      setError(undefined);
      onClose();
    } catch (error) {
      if (!(error instanceof Error)) {
        setError('Unknown error');
        return;
      }

      setError(error.message);
    } finally {
      setIsFetching(false);
    }
  }

  return (
    <Modal className="w-full max-w-3xl" isOpen={isOpen} onClose={onClose}>
      <Modal.Header>Rotate key</Modal.Header>

      <Modal.Body>
        <p className="mb-4">
          Before rotating, ensure that all of your apps have the correct{' '}
          <InlineCode value="INNGEST_SIGNING_KEY" /> and{' '}
          <InlineCode value="INNGEST_SIGNING_KEY_FALLBACK" /> environment variables.
        </p>

        <Alert severity="warning">
          This will permanently delete and replace the current key. It is irreversible.
        </Alert>
      </Modal.Body>

      <Modal.Footer>
        {error && (
          <Alert className="mb-6" severity="error">
            {error}
          </Alert>
        )}

        <div className="flex justify-end gap-2">
          <Button label="Close" appearance="outlined" btnAction={onClose} />
          <Button btnAction={onConfirm} disabled={isFetching} kind="danger" label="Rotate" />
        </div>
      </Modal.Footer>
    </Modal>
  );
}
