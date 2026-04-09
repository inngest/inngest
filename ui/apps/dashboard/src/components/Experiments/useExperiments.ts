import { useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import type { ExperimentListItem } from '@inngest/components/Experiments';

const experimentsQuery = graphql(`
  query GetExperiments($workspaceID: ID!) {
    experiments(workspaceID: $workspaceID) {
      name
      functionID
      selectionStrategy
      totalRuns
      variantCount
      firstSeen
      lastSeen
    }
  }
`);

export function useExperimentsList({
  enabled = true,
}: { enabled?: boolean } = {}) {
  const client = useClient();
  const environment = useEnvironment();

  const queryFn = useCallback(async (): Promise<ExperimentListItem[]> => {
    const result = await client
      .query(
        experimentsQuery,
        { workspaceID: environment.id },
        { requestPolicy: 'network-only' },
      )
      .toPromise();

    if (result.error) throw result.error;
    if (!result.data) throw new Error('No data returned');

    return result.data.experiments.map((exp) => ({
      experimentName: exp.name,
      functionId: exp.functionID,
      selectionStrategy: exp.selectionStrategy,
      totalRuns: exp.totalRuns,
      variantCount: exp.variantCount,
      firstSeen: new Date(exp.firstSeen),
      lastSeen: new Date(exp.lastSeen),
    }));
  }, [client, environment.id]);

  return useQuery<ExperimentListItem[]>({
    queryKey: ['experiments-list', environment.id],
    queryFn,
    enabled,
    staleTime: 30_000,
  });
}
