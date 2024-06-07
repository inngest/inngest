'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { AlertModal } from '@inngest/components/Modal';
import * as Tooltip from '@radix-ui/react-tooltip';
import { RiPauseLine, RiPlayFill } from '@remixicon/react';
import { toast } from 'sonner';
import { useMutation, useQuery } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { graphql } from '@/gql';

const FunctionVersionNumberDocument = graphql(`
  query GetFunctionVersionNumber($slug: String!, $environmentID: ID!) {
    workspace(id: $environmentID) {
      workflow: workflowBySlug(slug: $slug) {
        id
        isPaused
        name
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
  mutation PauseFunction($fnID: ID!) {
    pauseFunction(fnID: $fnID) {
      id
    }
  }
`);

const UnpauseFunctionDocument = graphql(`
  mutation UnpauseFunction($fnID: ID!) {
    unpauseFunction(fnID: $fnID) {
      id
    }
  }
`);

type PauseFunctionModalProps = {
  functionID: string;
  functionName: string;
  isPaused: boolean;
  isOpen: boolean;
  onClose: () => void;
};

function PauseFunctionModal({
  functionID,
  functionName,
  isPaused,
  isOpen,
  onClose,
}: PauseFunctionModalProps) {
  const [, pauseFunction] = useMutation(PauseFunctionDocument);
  const [, unpauseFunction] = useMutation(UnpauseFunctionDocument);

  function handlePause() {
    pauseFunction({ fnID: functionID }).then((result) => {
      if (result.error) {
        toast.error(`${functionName} could not be paused: ${result.error.message}`);
      } else {
        toast.success(`${functionName} was successfully paused`);
      }
    });
    onClose();
  }

  function handleResume() {
    unpauseFunction({ fnID: functionID }).then((result) => {
      if (result.error) {
        toast.error(`${functionName} could not be resumed: ${result.error.message}`);
      } else {
        toast.success(`${functionName} was successfully resumed`);
      }
    });
    onClose();
  }

  return (
    <AlertModal
      isOpen={isOpen}
      onClose={onClose}
      onSubmit={isPaused ? handleResume : handlePause}
      title={`Are you sure you want to ${isPaused ? 'resume' : 'pause'} this function?`}
      className="w-1/3"
    >
      {isPaused && (
        <p className="p-6 pb-0">
          This function will resume normal functionality and will be invoked as new events are
          received. Events received during pause will not be automatically replayed.
        </p>
      )}
      {!isPaused && (
        <ul className="list-inside list-disc p-6 pb-0 leading-8">
          <li>Existing runs will continue to run to completion.</li>
          <li>No new runs will be queued or invoked.</li>
          <li>Events will continue to be received, but they will not trigger new runs.</li>
          <li>Paused functions will be unpaused when you resync your app.</li>
          <li>Functions can be resumed at any time.</li>
        </ul>
      )}
    </AlertModal>
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

  const { isPaused } = fn;

  return (
    <>
      <Tooltip.Provider>
        <Tooltip.Root delayDuration={0}>
          <Tooltip.Trigger asChild>
            <span tabIndex={0}>
              <Button
                icon={
                  isPaused ? (
                    <RiPlayFill className=" text-green-600" />
                  ) : (
                    <RiPauseLine className=" text-amber-500" />
                  )
                }
                onClick={() => setIsPauseFunctionModalVisible(true)}
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
        functionName={fn.name}
        isPaused={isPaused}
        isOpen={isPauseFunctionModalVisible}
        onClose={() => setIsPauseFunctionModalVisible(false)}
      />
    </>
  );
}
