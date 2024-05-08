import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { graphql } from '@/gql';

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
        setError(new Error('Unknown error'));
      }
    } finally {
      setIsLoading(false);
    }
  }

  if (isArchived) {
    return (
      <Modal className="w-[800px]" isOpen={isOpen} onClose={onClose}>
        <Modal.Header>Unarchive app</Modal.Header>
        <Modal.Body>
          Are you sure you want to unarchive this app? Its functions will become triggerable again.
        </Modal.Body>
        <Modal.Footer className="flex justify-end gap-2">
          <Button appearance="outlined" btnAction={onClose} disabled={isLoading} label="Cancel" />
          <Button btnAction={onConfirm} disabled={isLoading} kind="danger" label="Unarchive" />
        </Modal.Footer>
      </Modal>
    );
  }

  return (
    <Modal className="w-[800px]" isOpen={isOpen} onClose={onClose}>
      <Modal.Header>Archive app</Modal.Header>
      <Modal.Body>
        Are you sure you want to archive this app? Its functions will no longer trigger. An archived
        app may be unarchived at any time.
        {error && (
          <Alert className="mt-4" severity="error">
            {error.message}
          </Alert>
        )}
      </Modal.Body>
      <Modal.Footer className="flex justify-end gap-2">
        <Button appearance="outlined" btnAction={onClose} disabled={isLoading} label="Cancel" />
        <Button btnAction={onConfirm} disabled={isLoading} kind="danger" label="Archive" />
      </Modal.Footer>
    </Modal>
  );
}
