'use client';

import Table from '@inngest/components/Table/NewTable';
import { type ColumnDef } from '@tanstack/react-table';

const LOADING_COLUMNS: ColumnDef<any, any>[] = [
  { id: 'col1', header: undefined, accessorKey: 'col1' },
  { id: 'col2', header: undefined, accessorKey: 'col2' },
  { id: 'col3', header: undefined, accessorKey: 'col3' },
];

export function LoadingState() {
  return <Table columns={LOADING_COLUMNS} data={undefined} isLoading />;
}
