import { useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import type { ExperimentListItem } from '@inngest/components/Experiments';

// ---------- Experiment list ----------

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

// ---------- Experiment detail ----------

export type ExperimentMetadataField = {
  key: string;
  label: string;
  valueType: string;
};

export type ExperimentDimensionValue = {
  key: string;
  value: string;
};

export type ExperimentInsightsRow = {
  dimensions: ExperimentDimensionValue[];
  runCount: number;
  failureRate: number;
  percentOfTotal: number;
};

export type ExperimentDetail = {
  summary: ExperimentListItem;
  availableFields: ExperimentMetadataField[];
  selectedFields: ExperimentMetadataField[];
  rows: ExperimentInsightsRow[];
};

type ExperimentDetailQueryResponse = {
  experimentDetail: {
    summary: {
      name: string;
      functionID: string;
      selectionStrategy: string;
      variants: string[];
      totalRuns: number;
      variantCount: number;
      firstSeen: string;
      lastSeen: string;
    };
    availableFields: ExperimentMetadataField[];
    selectedFields: ExperimentMetadataField[];
    rows: ExperimentInsightsRow[];
  };
};

const experimentDetailQuery = `
  query GetExperimentDetail($workspaceID: ID!, $experimentName: String!, $fields: [String!]) {
    experimentDetail(
      workspaceID: $workspaceID
      experimentName: $experimentName
      fields: $fields
    ) {
      summary {
        name
        functionID
        selectionStrategy
        variants
        totalRuns
        variantCount
        firstSeen
        lastSeen
      }
      availableFields {
        key
        label
        valueType
      }
      selectedFields {
        key
        label
        valueType
      }
      rows {
        dimensions {
          key
          value
        }
        runCount
        failureRate
        percentOfTotal
      }
    }
  }
`;

export function useExperimentDetail({
  experimentName,
  fields,
  enabled = true,
}: {
  experimentName: string;
  fields: string[];
  enabled?: boolean;
}) {
  const client = useClient();
  const environment = useEnvironment();

  const queryFn = useCallback(async (): Promise<ExperimentDetail> => {
    const result = await client
      .query<ExperimentDetailQueryResponse>(
        experimentDetailQuery,
        {
          workspaceID: environment.id,
          experimentName,
          fields: fields.length > 0 ? fields : undefined,
        },
        { requestPolicy: 'network-only' },
      )
      .toPromise();

    if (result.error) throw result.error;
    if (!result.data) throw new Error('No data returned');

    const s = result.data.experimentDetail.summary;
    return {
      summary: {
        experimentName: s.name,
        functionId: s.functionID,
        selectionStrategy: s.selectionStrategy,
        variants: s.variants,
        totalRuns: s.totalRuns,
        variantCount: s.variantCount,
        firstSeen: new Date(s.firstSeen),
        lastSeen: new Date(s.lastSeen),
      },
      availableFields: result.data.experimentDetail.availableFields,
      selectedFields: result.data.experimentDetail.selectedFields,
      rows: result.data.experimentDetail.rows,
    };
  }, [client, environment.id, experimentName, fields]);

  return useQuery<ExperimentDetail>({
    queryKey: ['experiment-detail', environment.id, experimentName, ...fields],
    queryFn,
    enabled,
    staleTime: 30_000,
  });
}
