'use client';

import { useCallback, useMemo, type UIEventHandler } from 'react';
import Table from '@inngest/components/Table/NewTable';

import { useInsightsQueryContext } from '../../context';
import { NoResults } from './NoResults';
import { ResultsTableFooter } from './ResultsTableFooter';
import { getTableColumns, handleScroll } from './utils';

export function ResultsTable() {
  const { data, fetchMore, state } = useInsightsQueryContext();

  const tableColumns = useMemo(() => {
    return getTableColumns(data?.columns || []);
  }, [data?.columns]);

  // TODO: Handle case where data doesn't fill viewport on tall monitors
  // When content is short and doesn't trigger scroll, users can't access more data.
  // Solutions: auto-fetch until scrollable OR show "Load More" button when !isScrollable

  const onScroll: UIEventHandler<HTMLDivElement> = useCallback(
    (event) => {
      handleScroll(event, {
        hasEntries: Boolean(data?.entries.length),
        hasNextPage: Boolean(data?.pageInfo.hasNextPage),
        state,
        fetchMore,
      });
    },
    [fetchMore, data?.pageInfo.hasNextPage, data?.entries.length, state]
  );

  if (!data?.entries.length) {
    return <NoResults />;
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex-1 overflow-auto" id="insights-table-container" onScroll={onScroll}>
        <Table columns={tableColumns} data={data.entries} isLoading={false} />
      </div>
      <ResultsTableFooter data={data} state={state} />
    </div>
  );
}
