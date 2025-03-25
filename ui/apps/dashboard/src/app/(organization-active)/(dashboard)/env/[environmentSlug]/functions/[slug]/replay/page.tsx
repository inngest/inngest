'use client';

import React from 'react';
import { useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import NewReplayButton from '@/components/Replay/NewReplayButton';
import { graphql } from '@/gql';
import { ReplayList } from './ReplayList';

const GetFunctionPauseStateDocument = graphql(`
  query GetFunctionPauseState($environmentID: ID!, $functionSlug: String!) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        id
        isPaused
      }
    }
  }
`);

type FunctionReplayPageProps = {
  params: {
    slug: string;
  };
};
export default function FunctionReplayPage({ params }: FunctionReplayPageProps) {
  const env = useEnvironment();
  const functionSlug = decodeURIComponent(params.slug);
  const [{ data }] = useQuery({
    query: GetFunctionPauseStateDocument,
    variables: {
      environmentID: env.id,
      functionSlug,
    },
  });
  const functionIsPaused = data?.environment.function?.isPaused || false;

  return (
    <>
      {!env.isArchived && !functionIsPaused && (
        <div className="flex items-center justify-end px-5">
          <NewReplayButton functionSlug={functionSlug} />
        </div>
      )}
      <div className="overflow-y-auto">
        <ReplayList functionSlug={functionSlug} />
      </div>
    </>
  );
}
