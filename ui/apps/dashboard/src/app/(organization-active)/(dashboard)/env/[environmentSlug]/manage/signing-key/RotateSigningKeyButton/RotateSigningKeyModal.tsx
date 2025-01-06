import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { InlineCode } from '@inngest/components/Code';
import { AlertModal } from '@inngest/components/Modal';
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
    <AlertModal
      className="w-full max-w-3xl"
      isOpen={isOpen}
      onClose={onClose}
      onSubmit={onConfirm}
      title="Rotate key"
      isLoading={isFetching}
      confirmButtonLabel="Rotate"
    >
      <div className="p-6">
        <p className="mb-4">
          Before rotating, ensure that all of your apps have the correct{' '}
          <InlineCode>INNGEST_SIGNING_KEY</InlineCode> and{' '}
          <InlineCode>INNGEST_SIGNING_KEY_FALLBACK</InlineCode> environment variables.
        </p>

        <Alert severity="warning" className="text-sm">
          This will permanently delete and replace the current key. It is irreversible.
        </Alert>
        {error && (
          <Alert className="mb-6 text-sm" severity="error">
            {error}
          </Alert>
        )}
      </div>
    </AlertModal>
  );
}
