'use client';

import { useMemo } from 'react';
import Table from '@inngest/components/Table/NewTable';

import { useInsightsQueryContext } from '../../context';
import type { InsightsEntry, InsightsResult, InsightsState } from '../types';
import { ResultsTableFooter, assertData } from './ResultsTableFooter';
import { useColumns } from './useColumns';
import { useOnScroll } from './useOnScroll';

export function ResultsTable() {
  const { data: dataRaw, fetchMore, state } = useInsightsQueryContext();
  const data = useMemo(() => withLoadingMoreRow(dataRaw, state), [dataRaw, state]);

  const { columns } = useColumns(data);
  const { onScroll } = useOnScroll(data, state, fetchMore);

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
  data: undefined | InsightsResult,
  state: InsightsState
): undefined | InsightsResult {
  if (data === undefined) return data;
  if (state !== 'fetchingMore') return data;

  const loadingRow: InsightsEntry = {
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
