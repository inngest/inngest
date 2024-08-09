'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { RiPauseLine, RiPlayFill } from '@remixicon/react';
import { useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { PauseFunctionModal } from './PauseModal';

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
