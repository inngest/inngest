'use client';

import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { AlertModal } from '@inngest/components/Modal';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
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
  keyID: string;
  isOpen: boolean;
  onClose: () => void;
  description?: string;
};

export default function DeleteKeyModal({
  keyID,
  isOpen,
  onClose,
  description,
}: DeleteKeyModalProps) {
  const env = useEnvironment();
  const [, deleteEventKey] = useMutation(DeleteEventKey);
  const router = useRouter();
  const currentContent = useManagePageTerminology();

  function handleDelete() {
    deleteEventKey({
      input: {
        id: keyID,
        workspaceID: env.id,
      },
    }).then((result) => {
      if (result.error) {
        toast.error(`${currentContent?.name} could not be deleted`);
      } else {
        toast.success(`${currentContent?.name} was successfully deleted`);
        router.push(`/env/${env.slug}/manage/${currentContent?.param}` as Route);
      }
    });
    onClose();
  }

  return (
    <AlertModal
      isOpen={isOpen}
      onClose={onClose}
      title={'Are you sure you want to delete this ' + currentContent?.name.toLowerCase() + '?'}
      description={description}
      onSubmit={handleDelete}
    />
  );
}
