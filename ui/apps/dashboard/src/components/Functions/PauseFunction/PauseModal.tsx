'use client';

import { useState } from 'react';
import { AlertModal } from '@inngest/components/Modal';
import { Select } from '@inngest/components/Select/Select';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { graphql } from '@/gql';

type CurrentRunHandlingOption = {
  name: string;
  id: string;
};
const CURRENT_RUN_HANDLING_STRATEGY_SUSPEND = 'suspend';
const CURRENT_RUN_HANDLING_STRATEGY_CANCEL = 'cancel';
const currentRunHandlingOptions = [
  {
    name: 'Pause immediately, then cancel after 7 days',
    id: CURRENT_RUN_HANDLING_STRATEGY_SUSPEND,
  },
  { name: 'Cancel immediately', id: CURRENT_RUN_HANDLING_STRATEGY_CANCEL },
] as const;
const defaultCurrentRunHandlingOption = currentRunHandlingOptions[0];

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

export function PauseFunctionModal({
  functionID,
  functionName,
  isPaused,
  isOpen,
  onClose,
}: PauseFunctionModalProps) {
  const [, pauseFunction] = useMutation(PauseFunctionDocument);
  const [, unpauseFunction] = useMutation(UnpauseFunctionDocument);
  const [currentRunHandlingStrategy, setCurrentRunHandlingStrategy] =
    useState<CurrentRunHandlingOption>(defaultCurrentRunHandlingOption);

  function onCloseWrapper() {
    setCurrentRunHandlingStrategy(defaultCurrentRunHandlingOption);
    onClose();
  }
  function handlePause() {
    pauseFunction({
      fnID: functionID,
      cancelRunning: currentRunHandlingStrategy.id == CURRENT_RUN_HANDLING_STRATEGY_CANCEL,
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

  let confirmButtonLabel = isPaused ? 'Resume Function' : 'Pause Function';
  if (!isPaused && currentRunHandlingStrategy.id === CURRENT_RUN_HANDLING_STRATEGY_CANCEL) {
    confirmButtonLabel = 'Pause & Cancel Runs';
  }

  return (
    <AlertModal
      isOpen={isOpen}
      onClose={onCloseWrapper}
      onSubmit={isPaused ? handleResume : handlePause}
      title={`${isPaused ? 'Resume' : 'Pause'} function “${functionName}”`}
      className="w-1/3"
      confirmButtonLabel={confirmButtonLabel}
      confirmButtonKind={isPaused ? 'primary' : 'danger'}
      cancelButtonLabel="Close"
    >
      {isPaused && (
        <div>
          <p className="p-6 pb-0 text-base">
            Are you sure you want to resume “<span className="font-semibold">{functionName}</span>”?
          </p>
          <p className="text-muted p-6 pb-0 pt-3 text-sm">
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
          <ul className="text-muted list-inside list-disc p-6 pb-0 pt-3 text-sm leading-6">
            <li>Functions can be resumed at any time.</li>
            <li>No new runs will be queued or invoked while the function is paused.</li>
            <li>
              No data will be lost. Events will continue being received, and you can process them
              later via a Replay.
            </li>
          </ul>
          <div className="p-6 pb-0">
            <hr className="border-muted" />
          </div>
          <label className="flex w-full flex-col gap-2 p-6 pb-5 pt-3 text-sm leading-6">
            <span className="text-muted">
              Choose what to do with currently-running function runs:
            </span>
            <Select
              onChange={setCurrentRunHandlingStrategy}
              isLabelVisible={false}
              label="Pause runs"
              multiple={false}
              value={currentRunHandlingStrategy}
            >
              <Select.Button isLabelVisible={false}>
                <div className="">
                  {currentRunHandlingStrategy.name ||
                    'How should currently-running runs be handled?'}
                </div>
              </Select.Button>
              <Select.Options>
                {currentRunHandlingOptions.map((option) => {
                  return (
                    <Select.Option key={option.id} option={option}>
                      {option.name}
                    </Select.Option>
                  );
                })}
              </Select.Options>
            </Select>
          </label>
        </div>
      )}
    </AlertModal>
  );
}
