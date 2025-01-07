import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';
import { useMutation } from 'urql';

import { graphql } from '@/gql';

const Mutation = graphql(`
  mutation DeleteSigningKey($signingKeyID: UUID!) {
    deleteSigningKey(id: $signingKeyID) {
      createdAt
    }
  }
`);

type Props = {
  isOpen: boolean;
  onClose: () => void;
  signingKeyID: string;
};

export function DeleteSigningKeyModal(props: Props) {
  const { isOpen, signingKeyID } = props;
  const [error, setError] = useState<string>();
  const [isFetching, setIsFetching] = useState(false);
  const [, deleteSigningKey] = useMutation(Mutation);

  function onClose() {
    setError(undefined);
    props.onClose();
  }

  async function onConfirm() {
    setIsFetching(true);
    try {
      const res = await deleteSigningKey(
        { signingKeyID },
        {
          // Bust cache
          additionalTypenames: ['SigningKey'],
        }
      );
      if (res.error) {
        throw res.error;
      }

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
      <Modal.Header>Permanently delete key</Modal.Header>

      <Modal.Body>
        <p className="mb-4">Are you sure you want to permanently delete this key?</p>

        <Alert severity="info">
          This key is inactive, so deletion will not affect communication between Inngest and your
          apps.
        </Alert>
      </Modal.Body>

      <Modal.Footer>
        {error && (
          <Alert className="mb-6" severity="error">
            {error}
          </Alert>
        )}

        <div className="flex justify-end gap-2">
          <Button label="Close" appearance="outlined" kind="secondary" onClick={onClose} />
          <Button
            onClick={onConfirm}
            disabled={isFetching}
            kind="danger"
            label="Permanently delete"
          />
        </div>
      </Modal.Footer>
    </Modal>
  );
}
