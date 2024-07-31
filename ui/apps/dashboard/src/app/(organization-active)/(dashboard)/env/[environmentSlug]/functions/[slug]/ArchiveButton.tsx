'use client';

import { Button } from '@inngest/components/Button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { RiArchive2Line, RiInboxUnarchiveLine } from '@remixicon/react';
import { useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

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

type ArchiveFunctionProps = {
  functionSlug: string;
};

/**
 * @deprecated Delete this component any time after 2024-05-17
 */
export default function ArchiveFunctionButton({ functionSlug }: ArchiveFunctionProps) {
  const environment = useEnvironment();

  const [{ data: version }] = useQuery({
    query: GetFunctionArchivalDocument,
    variables: {
      environmentID: environment.id,
      slug: functionSlug,
    },
  });

  const fn = version?.workspace.workflow;

  if (!fn) {
    return null;
  }

  const { isArchived } = fn;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span tabIndex={0}>
          <Button
            icon={
              isArchived ? (
                <RiInboxUnarchiveLine className=" text-slate-300" />
              ) : (
                <RiArchive2Line className=" text-slate-300" />
              )
            }
            btnAction={() =>
              console.error('manual function archival has been replaced with app archival')
            }
            disabled
            label={isArchived ? 'Unarchive' : 'Archive'}
          />
        </span>
      </TooltipTrigger>
      <TooltipContent className="align-center max-w-sm rounded-md px-2 text-xs">
        Manual function archival has been replaced with app archival
      </TooltipContent>
    </Tooltip>
  );
}
