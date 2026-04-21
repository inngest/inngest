import { useCallback } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { AnyVariables, Client, TypedDocumentNode } from 'urql';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import type {
  ExperimentDetail,
  ExperimentListItem,
  ExperimentScoringConfig,
  ExperimentScoringMetric,
  TimeRangePreset,
} from '@inngest/components/Experiments';

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

async function runMutation<Result, Variables extends AnyVariables>(
  client: Client,
  doc: TypedDocumentNode<Result, Variables>,
  variables: Variables,
): Promise<Result> {
  const result = await client.mutation(doc, variables).toPromise();
  if (result.error) throw result.error;
  if (!result.data) throw new Error('No data returned');
  return result.data;
}

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
    const data = await runQuery(client, experimentsQuery, {
      workspaceID: environment.id,
    });
    return data.experiments.map((exp) => ({
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
    staleTime: EXPERIMENTS_CACHE_MS,
    gcTime: EXPERIMENTS_CACHE_MS,
  });
}

const experimentDetailQuery = graphql(`
  query GetExperimentDetail(
    $workspaceID: ID!
    $experimentName: String!
    $timeRange: TimeRangeInput
    $variantFilter: String
  ) {
    experimentDetail(
      workspaceID: $workspaceID
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

const experimentScoringConfigQuery = graphql(`
  query GetExperimentScoringConfig($workspaceID: ID!, $experimentName: String!) {
    experimentScoringConfig(workspaceID: $workspaceID, experimentName: $experimentName) {
      experimentName
      updatedAt
      metrics {
        key
        enabled
        points
        minValue
        maxValue
        invert
        labelBest
        labelWorst
        displayName
      }
    }
  }
`);

const experimentInsightsQueryDoc = graphql(`
  query GetExperimentInsightsQuery(
    $workspaceID: ID!
    $experimentName: String!
    $timeRange: TimeRangeInput
  ) {
    experimentInsightsQuery(
      workspaceID: $workspaceID
      experimentName: $experimentName
      timeRange: $timeRange
    )
  }
`);

export function useExperimentInsightsQuery(
  experimentName: string,
  preset: TimeRangePreset,
) {
  const client = useClient();
  const environment = useEnvironment();

  const queryFn = useCallback(async (): Promise<string> => {
    const { from, to } = presetToRange(preset);
    const data = await runQuery(client, experimentInsightsQueryDoc, {
      workspaceID: environment.id,
      experimentName,
      timeRange: { from: from.toISOString(), to: to.toISOString() },
    });
    return data.experimentInsightsQuery;
  }, [client, environment.id, experimentName, preset]);

  return useQuery<string>({
    queryKey: [
      'experiment-insights-query',
      environment.id,
      experimentName,
      preset,
    ],
    queryFn,
    staleTime: EXPERIMENTS_CACHE_MS,
    gcTime: EXPERIMENTS_CACHE_MS,
  });
}

const updateExperimentScoringConfigMutation = graphql(`
  mutation UpdateExperimentScoringConfig(
    $workspaceID: ID!
    $experimentName: String!
    $metrics: [ExperimentScoringMetricInput!]!
  ) {
    updateExperimentScoringConfig(
      workspaceID: $workspaceID
      experimentName: $experimentName
      metrics: $metrics
    ) {
      experimentName
      updatedAt
      metrics {
        key
        enabled
        points
        minValue
        maxValue
        invert
        labelBest
        labelWorst
        displayName
      }
    }
  }
`);

function presetToRange(preset: TimeRangePreset): { from: Date; to: Date } {
  const to = new Date();
  const hours = preset === '24h' ? 24 : preset === '7d' ? 24 * 7 : 24 * 30;
  return { from: new Date(to.getTime() - hours * 60 * 60 * 1000), to };
}

export function useExperimentDetail(
  experimentName: string,
  preset: TimeRangePreset,
  variantFilter: string | null,
) {
  const client = useClient();
  const environment = useEnvironment();

  const queryFn = useCallback(async (): Promise<ExperimentDetail | null> => {
    const { from, to } = presetToRange(preset);
    const result = await client
      .query(
        experimentDetailQuery,
        {
          workspaceID: environment.id,
          experimentName,
          timeRange: { from: from.toISOString(), to: to.toISOString() },
          variantFilter: variantFilter || null,
        },
        { requestPolicy: 'network-only' },
      )
      .toPromise();

    // The server returns null when an experiment has no runs in the selected
    // time range. The GraphQL field is non-nullable, so urql surfaces this as
    // a "null which the schema does not allow" error. Treat it as empty data
    // rather than propagating as an error to the UI.
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
  }, [client, environment.id, experimentName, preset, variantFilter]);

  return useQuery<ExperimentDetail | null>({
    queryKey: [
      'experiment-detail',
      environment.id,
      experimentName,
      preset,
      variantFilter,
    ],
    queryFn,
    staleTime: EXPERIMENTS_CACHE_MS,
    gcTime: EXPERIMENTS_CACHE_MS,
  });
}

export function useExperimentScoringConfig(experimentName: string) {
  const client = useClient();
  const environment = useEnvironment();

  const queryFn = useCallback(async (): Promise<ExperimentScoringConfig> => {
    const data = await runQuery(client, experimentScoringConfigQuery, {
      workspaceID: environment.id,
      experimentName,
    });
    const c = data.experimentScoringConfig;
    return {
      experimentName: c.experimentName,
      updatedAt: new Date(c.updatedAt),
      metrics: c.metrics,
    };
  }, [client, environment.id, experimentName]);

  return useQuery<ExperimentScoringConfig>({
    queryKey: ['experiment-scoring', environment.id, experimentName],
    queryFn,
    staleTime: EXPERIMENTS_CACHE_MS,
    gcTime: EXPERIMENTS_CACHE_MS,
  });
}

export function useUpdateExperimentScoringConfig(experimentName: string) {
  const client = useClient();
  const environment = useEnvironment();
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: ['experiment-scoring-update', environment.id, experimentName],
    mutationFn: async (
      metrics: ExperimentScoringMetric[],
    ): Promise<ExperimentScoringConfig> => {
      const data = await runMutation(
        client,
        updateExperimentScoringConfigMutation,
        { workspaceID: environment.id, experimentName, metrics },
      );
      const c = data.updateExperimentScoringConfig;
      return {
        experimentName: c.experimentName,
        updatedAt: new Date(c.updatedAt),
        metrics: c.metrics,
      };
    },
    onSuccess: (data) => {
      queryClient.setQueryData(
        ['experiment-scoring', environment.id, experimentName],
        data,
      );
    },
  });
}
