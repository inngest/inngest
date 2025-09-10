'use client';

import { Table } from '@inngest/components/Table';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { ResultsTableFooter, assertData } from './ResultsTableFooter';
import mockInsightsData from './mockData';
import { useColumns } from './useColumns';

export function ResultsTable() {
  const { data } = useInsightsStateMachineContext();

  // Temporary toggle to use local mock data for development
  const USE_MOCK_DATA = true;
  const effectiveData = USE_MOCK_DATA ? mockInsightsData : data;

  const { columns } = useColumns(effectiveData);

  if (!assertData(effectiveData)) return null;

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex-1 overflow-auto" id="insights-table-container">
        <Table
          columns={columns}
          data={effectiveData.rows}
          cellClassName="[&:not(:first-child)]:border-l [&:not(:first-child)]:border-light box-border"
        />
      </div>
      <ResultsTableFooter />
    </div>
  );
}
