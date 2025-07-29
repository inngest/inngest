'use client';

import Table from '@inngest/components/Table/NewTable';

import { useInsightsQueryContext } from '../../context';
import { NoResults } from './NoResults';
import { ResultsTableFooter } from './ResultsTableFooter';
import { useColumns } from './useColumns';
import { useOnScroll } from './useOnScroll';

export function ResultsTable() {
  const { data, fetchMore, state } = useInsightsQueryContext();
  const { columns } = useColumns(data);
  const { onScroll } = useOnScroll(data, state, fetchMore);

  if (data === undefined) return <NoResults />;

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex-1 overflow-auto" id="insights-table-container" onScroll={onScroll}>
        <Table columns={columns} data={data.entries} isLoading={false} />
      </div>
      <ResultsTableFooter data={data} state={state} />
    </div>
  );
}
