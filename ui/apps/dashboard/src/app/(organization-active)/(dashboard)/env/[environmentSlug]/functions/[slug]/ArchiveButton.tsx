'use client';

import { Button } from '@inngest/components/Button';
import * as Tooltip from '@radix-ui/react-tooltip';
import { RiArchive2Line, RiInboxUnarchiveLine } from '@remixicon/react';
import { useQuery } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
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
    <>
      <Tooltip.Provider>
        <Tooltip.Root delayDuration={0}>
          <Tooltip.Trigger asChild>
            <span tabIndex={0}>
              <Button
                icon={
                  isArchived ? (
                    <RiInboxUnarchiveLine className=" text-slate-300" />
                  ) : (
                    <RiArchive2Line className=" text-slate-300" />
                  )
                }
                onClick={() =>
                  console.error('manual function archival has been replaced with app archival')
                }
                disabled
                label={isArchived ? 'Unarchive' : 'Archive'}
              />
            </span>
          </Tooltip.Trigger>
          <Tooltip.Content className="align-center rounded-md bg-slate-800 px-2 text-xs text-slate-300">
            Manual function archival has been replaced with app archival
            <Tooltip.Arrow className="fill-slate-800" />
          </Tooltip.Content>
        </Tooltip.Root>
      </Tooltip.Provider>
    </>
  );
}
