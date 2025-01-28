'use client';

import { useCallback, useEffect, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { AlertModal } from '@inngest/components/Modal/AlertModal';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { graphql } from '@/gql';

const ArchiveEnvironmentDocument = graphql(`
  mutation ArchiveEnvironment($id: ID!) {
    archiveEnvironment(id: $id) {
      id
    }
  }
`);

const UnarchiveEnvironmentDocument = graphql(`
  mutation UnarchiveEnvironment($id: ID!) {
    unarchiveEnvironment(id: $id) {
      id
    }
  }
`);

type Props = {
  envID: string;
  isArchived: boolean;
  isBranchEnv: boolean;
  isOpen: boolean;
  onCancel: () => void;
  onSuccess: () => void;
};

export function EnvironmentArchiveModal(props: Props) {
  const { envID, isBranchEnv, isOpen, onCancel, onSuccess } = props;
  const [error, setError] = useState<string>();
  const [isLoading, setIsLoading] = useState(false);
  const [, archive] = useMutation(ArchiveEnvironmentDocument);
  const [, unarchive] = useMutation(UnarchiveEnvironmentDocument);

  // Use an internal isArchived to prevent text changes after
  // confirmation.
  const [isArchived, setIsArchived] = useState(props.isArchived);
  useEffect(() => {
    if (isOpen) {
      setIsArchived(props.isArchived);
    }
  }, [isOpen, props.isArchived]);

  const onSubmit = useCallback(async () => {
    setIsLoading(true);

    try {
      let res;
      if (isArchived) {
        res = await unarchive({ id: envID });
      } else {
        res = await archive({ id: envID });
      }

      if (res.error) {
        throw res.error;
      }

      onSuccess();
      toast.success(`Environment ${isArchived ? 'unarchived' : 'archived'}`);
      setError(undefined);
    } catch (error) {
      if (!(error instanceof Error)) {
        setError('Unknown error');
        return;
      }

      setError(error.message);
    } finally {
      setIsLoading(false);
    }
  }, [archive, envID, isArchived, onSuccess, unarchive]);

  return (
    <AlertModal
      className="max-w-xl"
      isOpen={isOpen}
      onClose={onCancel}
      title={`${isArchived ? 'Unarchive' : 'Archive'} environment`}
      confirmButtonLabel={isArchived ? 'Unarchive' : 'Archive'}
      onSubmit={onSubmit}
      isLoading={isLoading}
      description={`Are you sure you want to ${
        isArchived ? 'unarchive' : 'archive'
      } this environment?`}
    >
      <div className="p-6 pb-0">
        {isArchived && (
          <p className="pb-4 text-sm">
            Any active functions within this environment will become triggerable.
          </p>
        )}

        {!isArchived && (
          <p className="pb-4 text-sm">
            Functions within this environment will no longer be triggerable. Nothing will be deleted
            and you can unarchive at any time.
          </p>
        )}

        {!isArchived && isBranchEnv && (
          <p className="pb-4 text-sm">
            Since this is a branch environment, any future app syncs will unarchive the environment.
          </p>
        )}
        {error && <Alert severity="error">{error}</Alert>}
      </div>
    </AlertModal>
  );
}
