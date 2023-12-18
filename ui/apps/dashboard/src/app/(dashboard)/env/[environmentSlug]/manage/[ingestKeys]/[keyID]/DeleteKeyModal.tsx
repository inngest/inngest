'use client';

import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { AlertModal } from '@inngest/components/Modal';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import useManagePageTerminology from './../useManagePageTerminology';

const DeleteEventKey = graphql(`
  mutation DeleteEventKey($input: DeleteIngestKey!) {
    deleteIngestKey(input: $input) {
      ids
    }
  }
`);

type DeleteKeyModalProps = {
  environmentSlug: string;
  environmentID: string;
  keyID: string;
  isOpen: boolean;
  onClose: () => void;
};

export default function DeleteKeyModal({
  environmentID,
  environmentSlug,
  keyID,
  isOpen,
  onClose,
}: DeleteKeyModalProps) {
  const input = {
    environmentID,
    keyID,
  };

  const [, deleteEventKey] = useMutation(DeleteEventKey);
  const router = useRouter();
  const currentContent = useManagePageTerminology();

  function handleDelete() {
    deleteEventKey({
      input: {
        id: keyID,
        workspaceID: environmentID,
      },
    }).then((result) => {
      if (result.error) {
        toast.error(`${currentContent?.name} could not be deleted`);
      } else {
        toast.success(`${currentContent?.name} was successfully deleted`);
        router.refresh();
        router.push(`/env/${environmentSlug}/manage/${currentContent?.param}` as Route);
      }
    });
    onClose();
  }

  return (
    <AlertModal
      isOpen={isOpen}
      onClose={onClose}
      title={'Are you sure you want to delete this ' + currentContent?.name?.toLowerCase() + '?'}
      primaryAction={{ label: 'Yes', btnAction: () => handleDelete() }}
    />
  );
}
