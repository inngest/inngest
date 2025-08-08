import { useCallback } from 'react';
import { useClient } from 'urql';

import { graphql } from '@/gql';
import { InsightsColumnType, type InsightsQuery } from '@/gql/graphql';
import type { InsightsFetchResult } from './types';

export interface FetchInsightsParams {
  after?: string | null;
  first: number;
  query: string;
}

const insightsQuery = graphql(`
  query Insights($query: String!, $first: Int!, $after: String) {
    insights(query: $query, first: $first, after: $after) {
      columns {
        name
        columnType
      }
      edges {
        cursor
        node {
          id
          values
        }
      }
      pageInfo {
        endCursor
        hasNextPage
      }
      totalCount
    }
  }
`);

export function useFetchInsights() {
  const client = useClient();

  const fetchInsights = useCallback(
    async ({ query, first, after = null }: FetchInsightsParams): Promise<InsightsFetchResult> => {
      const res = await client.query(insightsQuery, { after, first, query }).toPromise();
      if (res.error) throw res.error;
      if (!res.data) throw new Error('No data');

      return transformInsightsResponse(res.data.insights);
    },
    [client]
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
    entries: insights.edges.map((edge) => ({
      id: edge.node.id,
      values: transformValuesByColumns(edge.node.values, insights.columns),
      isLoadingRow: undefined,
    })),
    pageInfo: {
      endCursor: insights.pageInfo.endCursor,
      hasNextPage: insights.pageInfo.hasNextPage,
    },
    totalCount: insights.totalCount,
  };
}
