'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { ArchiveBoxIcon, PlayIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import * as Tooltip from '@radix-ui/react-tooltip';
import { toast } from 'sonner';
import { useMutation, useQuery } from 'urql';

import Modal from '@/components/Modal';
import { graphql } from '@/gql';
import UnarchiveIcon from '@/icons/unarchive.svg';
import { useEnvironment } from '@/queries';

const ArchiveFunctionDocument = graphql(`
  mutation ArchiveFunction($input: ArchiveWorkflowInput!) {
    archiveWorkflow(input: $input) {
      workflow {
        id
      }
    }
  }
`);

const GetFunctionArchivalDocument = graphql(`
  query GetFunctionArchival($slug: String!, $environmentID: ID!) {
    workspace(id: $environmentID) {
      workflow: workflowBySlug(slug: $slug) {
        id
        isArchived
        name
      }
    }
  }
`);

type ArchiveFunctionModalProps = {
  functionID: string | undefined;
  functionName: string;
  isArchived: boolean;
  isOpen: boolean;
  onClose: () => void;
};

function ArchiveFunctionModal({
  functionID,
  functionName,
  isArchived,
  isOpen,
  onClose,
}: ArchiveFunctionModalProps) {
  const [, archiveFunctionMutation] = useMutation(ArchiveFunctionDocument);
  const router = useRouter();

  function handleArchive() {
    if (functionID) {
      archiveFunctionMutation({
        input: {
          workflowID: functionID,
          archive: !isArchived,
        },
      }).then((result) => {
        if (result.error) {
          toast.error(
            `${functionName} could not be ${isArchived ? 'resumed' : 'archived'}: ${
              result.error.message
            }`
          );
        } else {
          toast.success(`${functionName} was successfully ${isArchived ? 'resumed' : 'archived'}`);
          router.refresh();
        }
      });
      onClose();
    }
  }

  return (
    <Modal className="flex max-w-xl flex-col gap-4" isOpen={isOpen} onClose={onClose}>
      <p>{`Are you sure you want to ${isArchived ? 'resume' : 'archive'} this function?`}</p>
      {isArchived && (
        <p className="pb-4 text-sm">
          Reactivate this function. This function will resume normal functionality and will be
          invoked as new events are received. Events received while archived will not be replayed.
        </p>
      )}
      {!isArchived && (
        <p className="pb-4 text-sm">
          Deactivate this function and prevent it from running again. Archived functions and their
          logs can be viewed at anytime. Functions can be unarchived if desired.
        </p>
      )}

      <div className="flex content-center justify-end">
        <Button appearance="outlined" btnAction={() => onClose()} label="No" />
        <Button kind="danger" appearance="text" btnAction={handleArchive} label="Yes" />
      </div>
    </Modal>
  );
}

type ArchiveFunctionProps = {
  environmentSlug: string;
  functionSlug: string;
};

export default function ArchiveFunctionButton({
  environmentSlug,
  functionSlug,
}: ArchiveFunctionProps) {
  const [isArchivedFunctionModalVisible, setIsArchivedFunctionModalVisible] = useState(false);
  const [{ data: environment }] = useEnvironment({ environmentSlug });

  const [{ data: version, fetching: isFetchingVersions }] = useQuery({
    query: GetFunctionArchivalDocument,
    variables: {
      environmentID: environment?.id!,
      slug: functionSlug,
    },
    pause: !environment?.id,
  });

  const fn = version?.workspace?.workflow;

  if (!fn) {
    return null;
  }

  const { isArchived } = fn;

  return (
    <>
      <Tooltip.Provider>
        <Tooltip.Root delayDuration={0}>
          <Tooltip.Trigger asChild>
            <span tabIndex={0}>
              <Button
                icon={
                  isArchived ? (
                    <UnarchiveIcon className=" text-slate-300" />
                  ) : (
                    <ArchiveBoxIcon className=" text-slate-300" />
                  )
                }
                btnAction={() => setIsArchivedFunctionModalVisible(true)}
                disabled={!version || isFetchingVersions}
                label={isArchived ? 'Unarchive' : 'Archive'}
              />
            </span>
          </Tooltip.Trigger>
          <Tooltip.Content className="align-center rounded-md bg-slate-800 px-2 text-xs text-slate-300">
            {isArchived
              ? 'Reactivate function'
              : 'Deactivate this function and archive for historic purposes'}
            <Tooltip.Arrow className="fill-slate-800" />
          </Tooltip.Content>
        </Tooltip.Root>
      </Tooltip.Provider>
      <ArchiveFunctionModal
        functionID={fn?.id}
        functionName={fn?.name ?? 'This function'}
        isOpen={isArchivedFunctionModalVisible}
        onClose={() => setIsArchivedFunctionModalVisible(false)}
        isArchived={isArchived}
      />
    </>
  );
}
