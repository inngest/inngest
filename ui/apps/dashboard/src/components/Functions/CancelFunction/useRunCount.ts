'use client';

import { baseInitialFetchFailed } from '@inngest/components/types/fetch';
import { maybeDateToString } from '@inngest/components/utils/date';

import { graphql } from '@/gql';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query GetCancellationRunCount(
    $envID: ID!
    $functionSlug: String!
    $queuedAtMin: Time
    $queuedAtMax: Time!
  ) {
    environment: workspace(id: $envID) {
      function: workflowBySlug(slug: $functionSlug) {
        cancellationRunCount(input: { queuedAtMin: $queuedAtMin, queuedAtMax: $queuedAtMax })
      }
    }
  }
`);

export type RunCountInput = {
  envID: string;
  functionSlug: string;
  queuedAtMax: Date;
  queuedAtMin: Date | null;
};

export function useRunCount(input?: RunCountInput) {
  const res = useSkippableGraphQLQuery({
    query,
    skip: !input,
    variables: {
      envID: input?.envID ?? '',
      functionSlug: input?.functionSlug ?? '',
      queuedAtMin: maybeDateToString(input?.queuedAtMin) ?? null,
      queuedAtMax: maybeDateToString(input?.queuedAtMax) ?? '',
    },
  });

  if (res.data) {
    if (!res.data.environment.function) {
      return {
        ...baseInitialFetchFailed,
        error: new Error('function not found'),
      };
    }

    return {
      ...res,
      data: res.data.environment.function.cancellationRunCount,
    };
  }

  return res;
}
