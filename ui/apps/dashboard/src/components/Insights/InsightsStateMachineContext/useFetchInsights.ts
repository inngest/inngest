'use client';

import { useCallback } from 'react';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { InsightsColumnType, type InsightsQuery } from '@/gql/graphql';
import { UNTITLED_QUERY } from '../InsightsTabManager/constants';
import type { InsightsFetchResult } from './types';

export interface FetchInsightsParams {
  query: string;
  queryName: string;
}

type FetchInsightsCallback = (query: string, name: undefined | string) => void;

const insightsQuery = graphql(`
  query Insights($query: String!, $workspaceID: ID!) {
    insights(query: $query, workspaceID: $workspaceID) {
      columns {
        name
        columnType
      }
      rows {
        values
      }
    }
  }
`);

export function useFetchInsights() {
  const client = useClient();
  const environment = useEnvironment();

  const fetchInsights = useCallback(
    async (
      { query, queryName }: FetchInsightsParams,
      cb: FetchInsightsCallback
    ): Promise<InsightsFetchResult> => {
      const res = await client
        .query(
          insightsQuery,
          { query, workspaceID: environment.id },
          { requestPolicy: 'network-only' }
        )
        .toPromise();
      if (res.error) throw res.error;
      if (!res.data) throw new Error('No data');

      cb(query, queryName === UNTITLED_QUERY ? undefined : queryName);
      return transformInsightsResponse(res.data.insights);
    },
    [client, environment.id]
  );

  return { fetchInsights };
}

function mapColumnType(columnType: InsightsColumnType): 'date' | 'number' | 'string' {
  switch (columnType) {
    case InsightsColumnType.Date:
      return 'date';
    case InsightsColumnType.Number:
      return 'number';
    case InsightsColumnType.String:
    case InsightsColumnType.Unknown:
    default:
      return 'string';
  }
}

function parseValueByType(
  value: string,
  columnType: InsightsColumnType
): string | number | Date | null {
  switch (columnType) {
    case InsightsColumnType.Number:
      return parseFloat(value);
    case InsightsColumnType.Date:
      return new Date(value);
    case InsightsColumnType.String:
    case InsightsColumnType.Unknown:
    default:
      return String(value);
  }
}

function transformValuesByColumns(
  values: string[],
  columns: Array<{ name: string; columnType: InsightsColumnType }>
): Record<string, string | number | Date | null> {
  return columns.reduce((acc, column, index) => {
    const value = values[index];
    if (value === undefined) {
      acc[column.name] = null;
      return acc;
    }

    acc[column.name] = parseValueByType(value, column.columnType);
    return acc;
  }, {} as Record<string, string | number | Date | null>);
}

function transformInsightsResponse(insights: InsightsQuery['insights']): InsightsFetchResult {
  return {
    columns: insights.columns.map((col) => ({
      name: col.name,
      type: mapColumnType(col.columnType),
    })),
    rows: insights.rows.map((row, index) => ({
      id: `row-${index}`,
      values: transformValuesByColumns(row.values, insights.columns),
    })),
  };
}
