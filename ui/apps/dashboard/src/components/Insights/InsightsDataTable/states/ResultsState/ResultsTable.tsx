'use client';

import { Table } from '@inngest/components/Table';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { ResultsTableFooter, assertData } from './ResultsTableFooter';
import { useColumns } from './useColumns';

export function ResultsTable() {
  const { data } = useInsightsStateMachineContext();

  const { columns } = useColumns(data);

  if (!assertData(data)) return null;

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex-1 overflow-auto" id="insights-table-container">
        <Table
          cellClassName="[&:not(:first-child)]:border-l [&:not(:first-child)]:border-light box-border max-w-[350px]"
          columns={columns}
          data={data.rows}
          enableHeaderTruncation
          headerClassName="max-w-[350px]"
        />
      </div>
      <ResultsTableFooter />
    </div>
  );
}
