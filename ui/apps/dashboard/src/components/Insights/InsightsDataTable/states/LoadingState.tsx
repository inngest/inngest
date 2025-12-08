import { memo } from 'react';
import { Table } from '@inngest/components/Table';
import { type ColumnDef } from '@tanstack/react-table';

type LoadingRowData = {
  col1?: unknown;
  col2?: unknown;
  col3?: unknown;
};

const LOADING_COLUMNS: ColumnDef<LoadingRowData, unknown>[] = [
  { id: 'col1', header: undefined, accessorKey: 'col1' },
  { id: 'col2', header: undefined, accessorKey: 'col2' },
  { id: 'col3', header: undefined, accessorKey: 'col3' },
];

// This component will freeze the UI if it's not memoized and
// the user causes a re-render by typing in the SQL editor.
// TODO: Debug this issue with the underlying Table that causes this.
export const LoadingState = memo(function LoadingState() {
  return <Table columns={LOADING_COLUMNS} data={undefined} isLoading />;
});
