'use client';

import { useMemo } from 'react';
import { Table } from '@inngest/components/Table';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type {
  InsightsFetchResult,
  InsightsStatus,
} from '@/components/Insights/InsightsStateMachineContext/types';
import { ResultsTableFooter, assertData } from './ResultsTableFooter';
import { useColumns } from './useColumns';
import { useOnScroll } from './useOnScroll';

export function ResultsTable() {
  const { data: dataRaw, fetchMore, status } = useInsightsStateMachineContext();
  const data = useMemo(() => withLoadingMoreRow(dataRaw, status), [dataRaw, status]);

  const { columns } = useColumns(data);
  const { onScroll } = useOnScroll(data, status, fetchMore);

  if (!assertData(data)) return null;

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex-1 overflow-auto" id="insights-table-container" onScroll={onScroll}>
        <Table
          columns={columns}
          data={data.entries}
          isLoading={false}
          cellClassName="[&:not(:first-child)]:border-l [&:not(:first-child)]:border-light box-border"
        />
      </div>
      <ResultsTableFooter />
    </div>
  );
}

function withLoadingMoreRow(
  data: undefined | InsightsFetchResult,
  status: InsightsStatus
): undefined | InsightsFetchResult {
  if (data === undefined) return data;
  if (status !== 'fetchingMore') return data;

  const loadingRow: InsightsFetchResult['entries'][number] = {
    id: `__loading_row__`,
    isLoadingRow: true,
    values: data.columns.reduce((acc, col) => {
      acc[col.name] = null;
      return acc;
    }, {} as Record<string, any>),
  };

  return {
    ...data,
    entries: [...data.entries, loadingRow],
  };
}
