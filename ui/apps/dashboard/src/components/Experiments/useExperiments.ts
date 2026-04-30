import { useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import type { AnyVariables, Client, TypedDocumentNode } from 'urql';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import type {
  ExperimentDetail,
  ExperimentListItem,
} from '@inngest/components/Experiments';

export type ExperimentTimeRange = { from: Date; to: Date };

const EXPERIMENTS_CACHE_MS = 5 * 60 * 1000;

async function runQuery<Result, Variables extends AnyVariables>(
  client: Client,
  doc: TypedDocumentNode<Result, Variables>,
  variables: Variables,
): Promise<Result> {
  const result = await client
    .query(doc, variables, { requestPolicy: 'network-only' })
    .toPromise();
  if (result.error) throw result.error;
  if (!result.data) throw new Error('No data returned');
  return result.data;
}

const experimentsQuery = graphql(`
  query GetExperiments($workspaceID: ID!) {
    experiments(workspaceID: $workspaceID) {
      name
      functionID
      functionSlug
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
    const data = await runQuery(client, experimentsQuery, {
      workspaceID: environment.id,
    });
    return data.experiments.map((exp) => ({
      experimentName: exp.name,
      functionId: exp.functionID,
      functionSlug: exp.functionSlug,
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
    staleTime: EXPERIMENTS_CACHE_MS,
    gcTime: EXPERIMENTS_CACHE_MS,
  });
}

const experimentDetailQuery = graphql(`
  query GetExperimentDetail(
    $workspaceID: ID!
    $functionID: ID!
    $experimentName: String!
    $timeRange: TimeRangeInput
    $variantFilter: String
  ) {
    experimentDetail(
      workspaceID: $workspaceID
      functionID: $functionID
      experimentName: $experimentName
      timeRange: $timeRange
      variantFilter: $variantFilter
    ) {
      name
      firstSeen
      lastSeen
      selectionStrategy
      variantWeights {
        variantName
        weight
      }
      variants {
        variantName
        runCount
        metrics {
          key
          avg
          min
          max
        }
      }
    }
  }
`);

const experimentInsightsQueryDoc = graphql(`
  query GetExperimentInsightsQuery(
    $workspaceID: ID!
    $functionID: ID!
    $experimentName: String!
    $timeRange: TimeRangeInput
  ) {
    experimentInsightsQuery(
      workspaceID: $workspaceID
      functionID: $functionID
      experimentName: $experimentName
      timeRange: $timeRange
    )
  }
`);

export function useExperimentInsightsQuery(
  functionID: string,
  experimentName: string,
  range: ExperimentTimeRange,
  options: { enabled?: boolean } = {},
) {
  const client = useClient();
  const environment = useEnvironment();
  const fromIso = range.from.toISOString();
  const toIso = range.to.toISOString();

  const queryFn = useCallback(async (): Promise<string> => {
    const data = await runQuery(client, experimentInsightsQueryDoc, {
      workspaceID: environment.id,
      functionID,
      experimentName,
      timeRange: { from: fromIso, to: toIso },
    });
    return data.experimentInsightsQuery;
  }, [client, environment.id, functionID, experimentName, fromIso, toIso]);

  return useQuery<string>({
    queryKey: [
      'experiment-insights-query',
      environment.id,
      functionID,
      experimentName,
      fromIso,
      toIso,
    ],
    queryFn,
    enabled: options.enabled ?? true,
    staleTime: EXPERIMENTS_CACHE_MS,
    gcTime: EXPERIMENTS_CACHE_MS,
  });
}

export function useExperimentDetail(
  functionID: string,
  experimentName: string,
  range: ExperimentTimeRange,
  variantFilter: string | null,
  options: { enabled?: boolean } = {},
) {
  const client = useClient();
  const environment = useEnvironment();
  const fromIso = range.from.toISOString();
  const toIso = range.to.toISOString();

  const queryFn = useCallback(async (): Promise<ExperimentDetail | null> => {
    const result = await client
      .query(
        experimentDetailQuery,
        {
          workspaceID: environment.id,
          functionID,
          experimentName,
          timeRange: { from: fromIso, to: toIso },
          variantFilter: variantFilter || null,
        },
        { requestPolicy: 'network-only' },
      )
      .toPromise();

    // Empty experiment ranges currently surface as this GraphQL error.
    const isNoDataInRange = result.error?.graphQLErrors.some((e) =>
      e.message.includes('null which the schema does not allow'),
    );
    if (isNoDataInRange) return null;

    if (result.error) throw result.error;
    if (!result.data) throw new Error('No data returned');

    const d = result.data.experimentDetail;
    return {
      name: d.name,
      firstSeen: new Date(d.firstSeen),
      lastSeen: new Date(d.lastSeen),
      selectionStrategy: d.selectionStrategy,
      variantWeights: d.variantWeights,
      variants: d.variants.map((v) => ({
        variantName: v.variantName,
        runCount: v.runCount,
        metrics: v.metrics,
      })),
    };
  }, [
    client,
    environment.id,
    functionID,
    experimentName,
    fromIso,
    toIso,
    variantFilter,
  ]);

  return useQuery<ExperimentDetail | null>({
    queryKey: [
      'experiment-detail',
      environment.id,
      functionID,
      experimentName,
      fromIso,
      toIso,
      variantFilter,
    ],
    queryFn,
    enabled: options.enabled ?? true,
    staleTime: EXPERIMENTS_CACHE_MS,
    gcTime: EXPERIMENTS_CACHE_MS,
  });
}
