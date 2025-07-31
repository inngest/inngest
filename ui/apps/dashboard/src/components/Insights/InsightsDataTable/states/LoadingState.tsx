'use client';

import { memo } from 'react';
import Table from '@inngest/components/Table/NewTable';
import { type ColumnDef } from '@tanstack/react-table';

const LOADING_COLUMNS: ColumnDef<any, any>[] = [
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
