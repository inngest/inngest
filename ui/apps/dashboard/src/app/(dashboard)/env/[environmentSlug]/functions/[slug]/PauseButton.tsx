'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { PauseIcon, PlayIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import * as Tooltip from '@radix-ui/react-tooltip';
import { toast } from 'sonner';
import { useMutation, useQuery } from 'urql';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import Modal from '@/components/Modal';
import { graphql } from '@/gql';

const FunctionVersionNumberDocument = graphql(`
  query GetFunctionVersionNumber($slug: String!, $environmentID: ID!) {
    workspace(id: $environmentID) {
      workflow: workflowBySlug(slug: $slug) {
        id
        archivedAt
        current {
          version
        }
        previous {
          version
        }
      }
    }
  }
`);

const PauseFunctionDocument = graphql(`
  mutation PauseFunction($input: EditWorkflowInput!) {
    editWorkflow(input: $input) {
      workflow {
        id
        name
      }
    }
  }
`);

type PauseFunctionModalProps = {
  functionID: string | undefined;
  functionName: string;
  currentVersion?: number | undefined;
  previousVersion?: number | undefined;
  isPaused: boolean;
  isOpen: boolean;
  onClose: () => void;
};

function PauseFunctionModal({
  functionID,
  functionName,
  currentVersion,
  previousVersion,
  isPaused,
  isOpen,
  onClose,
}: PauseFunctionModalProps) {
  const [, pauseFunctionMutation] = useMutation(PauseFunctionDocument);
  const router = useRouter();

  function handlePause() {
    if (functionID && currentVersion) {
      pauseFunctionMutation({
        input: {
          description: null,
          promote: null,
          workflowID: functionID,
          disable: new Date().toISOString(),
          version: currentVersion,
        },
      }).then((result) => {
        if (result.error) {
          toast.error(`${functionName} could not be paused: ${result.error.message}`);
        } else {
          toast.success(
            `${result.data?.editWorkflow?.workflow.name || functionName} was successfully paused`
          );
          router.refresh();
        }
      });
      onClose();
    }
  }

  function handleResume() {
    if (functionID && previousVersion) {
      pauseFunctionMutation({
        input: {
          disable: null,
          description: null,
          workflowID: functionID,
          promote: new Date().toISOString(),
          version: previousVersion,
        },
      }).then((result) => {
        if (result.error) {
          toast.error(`${functionName} could not be resumed: ${result.error.message}`);
        } else {
          toast.success(
            `${result.data?.editWorkflow?.workflow.name || functionName} was successfully resumed`
          );
          router.refresh();
        }
      });
      onClose();
    }
  }

  return (
    <Modal className="flex max-w-xl flex-col gap-4" isOpen={isOpen} onClose={onClose}>
      <p>{`Are you sure you want to ${isPaused ? 'resume' : 'pause'} this function?`}</p>
      {isPaused && (
        <p className="pb-4 text-sm">
          This function will resume normal functionality and will be invoked as new events are
          received. Events received during pause will not be automatically replayed.
        </p>
      )}
      {!isPaused && (
        <p className="pb-4 text-sm">
          Temporarily stop this function from being run. Events can still be sent, but this function
          will not be invoked. You can resume it at any time.
        </p>
      )}
      <div className="flex content-center justify-end">
        <Button appearance="outlined" btnAction={() => onClose()} label="No" />
        <Button
          kind="danger"
          appearance="text"
          btnAction={isPaused ? handleResume : handlePause}
          label="Yes"
        />
      </div>
    </Modal>
  );
}

type PauseFunctionProps = {
  functionSlug: string;
  disabled: boolean;
};

export default function PauseFunctionButton({ functionSlug, disabled }: PauseFunctionProps) {
  const [isPauseFunctionModalVisible, setIsPauseFunctionModalVisible] = useState<boolean>(false);
  const environment = useEnvironment();

  const [{ data: version, fetching: isFetchingVersions }] = useQuery({
    query: FunctionVersionNumberDocument,
    variables: {
      environmentID: environment.id,
      slug: functionSlug,
    },
  });

  const fn = version?.workspace.workflow;

  if (!fn) {
    return null;
  }

  const prevVersionObj = fn.previous.sort((a, b) => b!.version - a!.version)[0];
  const prevVersion = prevVersionObj?.version;
  const isPaused = !fn.current && !fn.archivedAt;

  return (
    <>
      <Tooltip.Provider>
        <Tooltip.Root delayDuration={0}>
          <Tooltip.Trigger asChild>
            <span tabIndex={0}>
              <Button
                icon={
                  isPaused ? (
                    <PlayIcon className=" text-green-600" />
                  ) : (
                    <PauseIcon className=" text-amber-500" />
                  )
                }
                btnAction={() => setIsPauseFunctionModalVisible(true)}
                disabled={disabled || isFetchingVersions}
                label={isPaused ? 'Resume' : 'Pause'}
              />
            </span>
          </Tooltip.Trigger>
          <Tooltip.Content className="align-center rounded-md bg-slate-800 px-2 text-xs text-slate-300">
            {isPaused
              ? 'Begin running this function after a temporary pause'
              : 'Temporarily stop a function from being run'}
            <Tooltip.Arrow className="fill-slate-800" />
          </Tooltip.Content>
        </Tooltip.Root>
      </Tooltip.Provider>
      <PauseFunctionModal
        functionID={fn.id}
        functionName={functionSlug}
        currentVersion={fn.current?.version}
        previousVersion={prevVersion}
        isPaused={isPaused}
        isOpen={isPauseFunctionModalVisible}
        onClose={() => setIsPauseFunctionModalVisible(false)}
      />
    </>
  );
}
