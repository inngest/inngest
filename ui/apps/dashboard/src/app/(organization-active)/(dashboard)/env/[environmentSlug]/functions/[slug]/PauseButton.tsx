'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { PauseIcon, PlayIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import { AlertModal } from '@inngest/components/Modal';
import { toast } from 'sonner';
import { useMutation, useQuery } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
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
    <AlertModal
      isOpen={isOpen}
      onClose={onClose}
      primaryAction={{
        label: isPaused ? 'Resume' : 'Pause',
        btnAction: isPaused ? handleResume : handlePause,
      }}
      title={`Are you sure you want to ${isPaused ? 'resume' : 'pause'} this function?`}
      className="w-1/3"
    >
      {isPaused && (
        <p className="pt-4">
          This function will resume normal functionality and will be invoked as new events are
          received. Events received during pause will not be automatically replayed.
        </p>
      )}
      {!isPaused && (
        <ul className="list-disc p-4 pb-0 leading-8">
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

  const prevVersionObj = fn.previous.sort((a, b) => b!.version - a!.version)[0];
  const prevVersion = prevVersionObj?.version;
  const isPaused = !fn.current && !fn.archivedAt;

  return (
    <>
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
