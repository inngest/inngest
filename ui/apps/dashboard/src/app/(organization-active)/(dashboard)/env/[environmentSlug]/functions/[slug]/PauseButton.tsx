'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { AlertModal } from '@inngest/components/Modal';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { RiPauseLine, RiPlayFill } from '@remixicon/react';
import { toast } from 'sonner';
import { useMutation, useQuery } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { SelectInput } from '@/components/Forms/SelectInput';
import { graphql } from '@/gql';

type CurrentRunHandlingOption = {
  label: string;
  value: string;
};
const CURRENT_RUN_HANDLING_STRATEGY_SUSPEND = 'suspend';
const CURRENT_RUN_HANDLING_STRATEGY_CANCEL = 'cancel';
const currentRunHandlingOptions: CurrentRunHandlingOption[] = [
  {
    label: 'Pause immediately, then cancel after 7 days',
    value: CURRENT_RUN_HANDLING_STRATEGY_SUSPEND,
  },
  { label: 'Cancel immediately', value: CURRENT_RUN_HANDLING_STRATEGY_CANCEL },
];
type CurrentRunHandlingStrategy = (typeof currentRunHandlingOptions)[number]['value'];

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
  mutation PauseFunction($fnID: ID!, $cancelRunning: Boolean) {
    pauseFunction(fnID: $fnID, cancelRunning: $cancelRunning) {
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
  const [currentRunHandlingStrategy, setCurrentRunHandlingStrategy] =
    useState<CurrentRunHandlingStrategy>(CURRENT_RUN_HANDLING_STRATEGY_SUSPEND);

  function onCloseWrapper() {
    setCurrentRunHandlingStrategy(CURRENT_RUN_HANDLING_STRATEGY_SUSPEND);
    onClose();
  }
  function handlePause() {
    pauseFunction({
      fnID: functionID,
      cancelRunning: currentRunHandlingStrategy == CURRENT_RUN_HANDLING_STRATEGY_CANCEL,
    }).then((result) => {
      if (result.error) {
        toast.error(`“${functionName}” could not be paused: ${result.error.message}`);
      } else {
        toast.success(`“${functionName}” was successfully paused`);
      }
    });
    onCloseWrapper();
  }
  function handleResume() {
    unpauseFunction({ fnID: functionID }).then((result) => {
      if (result.error) {
        toast.error(`“${functionName}” could not be resumed: ${result.error.message}`);
      } else {
        toast.success(`“${functionName}” was successfully resumed`);
      }
    });
    onCloseWrapper();
  }

  return (
    <AlertModal
      isOpen={isOpen}
      onClose={onCloseWrapper}
      onSubmit={isPaused ? handleResume : handlePause}
      title={`${isPaused ? 'Resume' : 'Pause'} function “${functionName}”`}
      className="w-1/3"
      confirmButtonLabel={isPaused ? 'Resume' : 'Pause'}
      confirmButtonKind={isPaused ? 'success' : 'danger'}
      cancelButtonLabel="Cancel"
    >
      {isPaused && (
        <div>
          <p className="p-6 pb-0 text-base">
            Are you sure you want to resume “<span className="font-semibold">{functionName}</span>”?
          </p>
          <p className="text-subtle p-6 pb-0 pt-3 text-sm">
            This function will resume normal functionality and will be invoked as new events are
            received. Events received during pause will not be automatically replayed.
          </p>
        </div>
      )}
      {!isPaused && (
        <div>
          <p className="p-6 pb-0 text-base">
            Are you sure you want to pause “<span className="font-semibold">{functionName}</span>”?
          </p>
          <ul className="text-subtle list-inside list-disc p-6 pb-0 pt-3 text-sm leading-6">
            <li>Functions can be resumed at any time.</li>
            <li>No new runs will be queued or invoked.</li>
            <li>Events will continue to be received, but they will not trigger new runs.</li>
          </ul>
          <div className="p-6 pb-0">
            <hr className="border-muted" />
          </div>
          <label className="flex w-full flex-col gap-2 p-6 pb-5 pt-3 text-sm leading-6">
            <span className="text-subtle">
              Choose what to do with currently-running function runs:
            </span>
            <SelectInput
              value={currentRunHandlingStrategy}
              options={currentRunHandlingOptions}
              onChange={setCurrentRunHandlingStrategy}
              placeholder="How should currently-running runs be handled?"
            />
          </label>
        </div>
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
      <Tooltip delayDuration={0}>
        <TooltipTrigger asChild>
          <span tabIndex={0}>
            <Button
              icon={
                isPaused ? (
                  <RiPlayFill className=" text-green-600" />
                ) : (
                  <RiPauseLine className=" text-amber-500" />
                )
              }
              btnAction={() => setIsPauseFunctionModalVisible(true)}
              disabled={disabled || isFetchingVersions}
              label={isPaused ? 'Resume' : 'Pause'}
            />
          </span>
        </TooltipTrigger>
        <TooltipContent className="align-center rounded-md px-2 text-xs">
          {isPaused
            ? 'Begin running this function after a temporary pause'
            : 'Temporarily stop a function from being run'}
        </TooltipContent>
      </Tooltip>
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
