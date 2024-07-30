import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';
import { useMutation } from 'urql';

import { graphql } from '@/gql';

const mutation = graphql(`
  mutation DeleteCancellation($envID: UUID!, $cancellationID: ULID!) {
    deleteCancellation(envID: $envID, cancellationID: $cancellationID)
  }
`);

type Props = {
  onClose: () => void;
  pendingDelete:
    | {
        id: string;
        envID: string;
      }
    | undefined;
};

export function DeleteCancellationModal(props: Props) {
  const { pendingDelete } = props;
  const isOpen = Boolean(pendingDelete);
  const [error, setError] = useState<string>();
  const [isFetching, setIsFetching] = useState(false);
  const [, deleteCancellation] = useMutation(mutation);

  function onClose() {
    setError(undefined);
    props.onClose();
  }

  async function onConfirm() {
    if (!pendingDelete) {
      // Unreachable
      return;
    }

    setIsFetching(true);
    try {
      const res = await deleteCancellation(
        { cancellationID: pendingDelete.id, envID: pendingDelete.envID },
        {
          // Bust cache
          additionalTypenames: ['Cancellation'],
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
      <Modal.Header>Delete cancellation</Modal.Header>

      <Modal.Body>
        <p className="mb-4">
          Are you sure you want to delete this cancellation? This will not affect function runs that
          were already cancelled.
        </p>
      </Modal.Body>

      <Modal.Footer>
        {error && (
          <Alert className="mb-6" severity="error">
            {error}
          </Alert>
        )}

        <div className="flex justify-end gap-2">
          <Button label="Close" appearance="outlined" btnAction={onClose} />
          <Button btnAction={onConfirm} disabled={isFetching} kind="danger" label="Delete" />
        </div>
      </Modal.Footer>
    </Modal>
  );
}
