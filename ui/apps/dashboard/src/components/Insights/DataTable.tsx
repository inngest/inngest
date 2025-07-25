'use client';

import NewTable from '@inngest/components/Table/NewTable';

import { EmptyState } from './EmptyState';
import type { InsightData } from './mockData';

const MOCK_SEE_EXAMPLES = () => {
  alert('TODO');
};

type DataTableProps = {
  className?: string;
  data: InsightData[];
  isLoading?: boolean;
};

export function DataTable({ className, data, isLoading = false }: DataTableProps) {
  return (
    <div className="border-subtle flex min-h-0 flex-1 flex-col border">
      <div className="border-subtle flex h-12 shrink-0 items-center justify-between border-b px-4">
        <div className="flex items-center gap-2">
          <h3 className="text-basis text-sm font-medium">Results</h3>
        </div>
      </div>

      <div className="bg-canvasBase flex min-h-0 flex-1 items-center justify-center overflow-y-auto">
        <div className="-translate-y-8">
          <NewTable
            blankState={<EmptyState onSeeExamples={MOCK_SEE_EXAMPLES} />}
            columns={[]}
            data={data}
            isLoading={isLoading}
          />
        </div>
      </div>
    </div>
  );
}
