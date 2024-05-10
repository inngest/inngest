import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { AlertModal } from '@inngest/components/Modal';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { graphql } from '@/gql';

type Props = {
  appID: string;
  isArchived: boolean;
  isOpen: boolean;
  onClose: () => void;
};

export function ArchiveModal({ appID, isArchived, isOpen, onClose }: Props) {
  const [error, setError] = useState<Error>();
  const [isLoading, setIsLoading] = useState(false);
  const [, archiveApp] = useMutation(ArchiveAppDocument);
  const [, unarchiveApp] = useMutation(UnarchiveAppDocument);

  async function onConfirm() {
    setIsLoading(true);
    try {
      let error;
      let message: string;
      if (isArchived) {
        error = (await unarchiveApp({ appID })).error;
        message = 'Unarchived app';
      } else {
        error = (await archiveApp({ appID })).error;
        message = 'Archived app';
      }
      if (error) {
        throw error;
      }
      setError(undefined);
      toast.success(message);
      onClose();
    } catch (error) {
      if (error instanceof Error) {
        setError(error);
      } else {
        setError(new Error('unknown error'));
      }
    } finally {
      setIsLoading(false);
    }
  }

  if (isArchived) {
    return (
      <AlertModal
        isLoading={isLoading}
        isOpen={isOpen}
        onClose={onClose}
        onSubmit={onConfirm}
        title="Are you sure you want to unarchive this app?"
        className="w-[600px]"
      >
        <ul className="mt-4 list-inside list-disc">
          <li>New function runs can trigger.</li>
          <li>You may re-archive at any time.</li>
        </ul>

        {error && (
          <Alert className="mt-4" severity="error">
            {error.message}
          </Alert>
        )}
      </AlertModal>
    );
  }

  return (
    <AlertModal
      isLoading={isLoading}
      isOpen={isOpen}
      onClose={onClose}
      onSubmit={onConfirm}
      title="Are you sure you want to archive this app?"
      className="w-[600px]"
    >
      <ul className="mt-4 list-inside list-disc">
        <li>New function runs will not trigger.</li>
        <li>Existing function runs will continue until completion.</li>
        <li>Functions will still be visible, including their run history.</li>
        <li>You may unarchive at any time.</li>
      </ul>

      {error && (
        <Alert className="mt-4" severity="error">
          {error.message}
        </Alert>
      )}
    </AlertModal>
  );
}

const ArchiveAppDocument = graphql(`
  mutation AchiveApp($appID: UUID!) {
    archiveApp(id: $appID) {
      id
    }
  }
`);

const UnarchiveAppDocument = graphql(`
  mutation UnachiveApp($appID: UUID!) {
    unarchiveApp(id: $appID) {
      id
    }
  }
`);
