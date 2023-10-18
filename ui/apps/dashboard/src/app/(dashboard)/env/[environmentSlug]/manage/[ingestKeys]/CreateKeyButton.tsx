'use client';

import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { PlusIcon } from '@heroicons/react/24/solid';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import Button from '@/components/Button';
import Modal from '@/components/Modal';
import { graphql } from '@/gql';
import { useEnvironment } from '@/queries';
import useManagePageTerminology from './useManagePageTerminology';

const CreateSourceKey = graphql(`
  mutation NewIngestKey($input: NewIngestKey!) {
    key: createIngestKey(input: $input) {
      id
    }
  }
`);

type NewKeyButtonProps = {
  environmentSlug: string;
};

export default function CreateKeyButton({ environmentSlug }: NewKeyButtonProps) {
  const currentContent = useManagePageTerminology();
  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug,
  });
  const [, createSourceKey] = useMutation(CreateSourceKey);
  const router = useRouter();
  const environmentID = environment?.id ?? '';

  if (!currentContent) {
    return null;
  }

  function handleClick() {
    if (currentContent) {
      createSourceKey({
        input: {
          filterList: null,
          workspaceID: environmentID,
          name: `My new ${currentContent.name}`,
          source: currentContent.type,
          metadata: {
            transform: undefined,
          },
        },
      }).then((result) => {
        if (result.error) {
          toast.error(`${currentContent.name} could not be created`);
        } else {
          toast.success(`${currentContent.name} was successfully created`);
          router.refresh();

          const newKeyID = result?.data?.key?.id;
          if (newKeyID) {
            router.push(
              `/env/${environmentSlug}/manage/${currentContent.param}/${newKeyID}` as Route
            );
          }
        }
      });
    }
  }

  return (
    <Button
      icon={<PlusIcon className="h-4" />}
      onClick={handleClick}
      disabled={!environment || isFetchingEnvironment || !currentContent}
    >
      Create {currentContent?.name}
    </Button>
  );
}
