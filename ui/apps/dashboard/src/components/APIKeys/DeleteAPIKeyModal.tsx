import { useEffect, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { AlertModal } from '@inngest/components/Modal';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { apiKeyErrorMessage } from './errorMessage';

const Mutation = graphql(`
  mutation DeleteAPIKey($id: UUID!) {
    deleteAPIKey(id: $id)
  }
`);

type Props = {
  isOpen: boolean;
  onClose: () => void;
  keyID: string | undefined;
  keyName: string | undefined;
};

export function DeleteAPIKeyModal({ isOpen, onClose, keyID, keyName }: Props) {
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [, del] = useMutation(Mutation);

  useEffect(() => {
    if (isOpen) {
      setError(null);
      setIsSubmitting(false);
    }
  }, [isOpen, keyID]);

  async function submit() {
    if (!keyID) return;
    setError(null);
    setIsSubmitting(true);
    try {
      const res = await del({ id: keyID }, { additionalTypenames: ['APIKey'] });
      if (res.error) {
        setError(apiKeyErrorMessage(res.error, 'Could not delete API key.'));
        return;
      }
      toast.success('API key deleted');
      onClose();
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <AlertModal
      className="w-full max-w-md"
      isOpen={isOpen}
      onClose={onClose}
      onSubmit={submit}
      title={keyName ? `Delete "${keyName}"?` : 'Delete API key?'}
      description="Any application using this key will immediately lose access. This cannot be undone."
      confirmButtonKind="danger"
      confirmButtonLabel="Delete"
      cancelButtonLabel="Cancel"
      isLoading={isSubmitting}
      autoClose={false}
    >
      {error && (
        <div className="px-6 pb-2">
          <Alert severity="error">{error}</Alert>
        </div>
      )}
    </AlertModal>
  );
}
