'use client';

import { useState } from 'react';
import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { PlusIcon } from '@heroicons/react/24/solid';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import Input from '@/components/Forms/Input';
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
  const [inputValue, setInputValue] = useState<string>('');
  const [isModalOpen, setModalOpen] = useState(false);
  const currentContent = useManagePageTerminology();
  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug,
  });
  const [{ fetching }, createSourceKey] = useMutation(CreateSourceKey);
  const router = useRouter();
  const environmentID = environment?.id ?? '';

  if (!currentContent) {
    return null;
  }

  function handleClick() {
    if (currentContent && inputValue) {
      createSourceKey({
        input: {
          filterList: null,
          workspaceID: environmentID,
          name: inputValue,
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
    <>
      <Button
        icon={<PlusIcon />}
        btnAction={() => setModalOpen(true)}
        disabled={!environment || isFetchingEnvironment || !currentContent}
        kind="primary"
        label={`Create ${currentContent?.name}`}
      />
      <Modal
        isOpen={isModalOpen}
        className={'w-1/4'}
        onClose={() => setModalOpen(false)}
        title={`Create a New ${currentContent?.name}`}
        footer={
          <div className="flex justify-end gap-2">
            <Button
              appearance="outlined"
              label="Cancel"
              btnAction={() => {
                setModalOpen(false);
              }}
            />
            <Button
              kind="primary"
              label="Create"
              loading={fetching}
              btnAction={() => {
                handleClick();
                setModalOpen(false);
              }}
              disabled={!inputValue}
            />
          </div>
        }
      >
        <div className="p-6">
          <Input
            name="keyName"
            placeholder={`${currentContent?.name} Name`}
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
          />
        </div>
      </Modal>
    </>
  );
}
