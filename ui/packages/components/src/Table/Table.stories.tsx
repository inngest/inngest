import { createRef } from 'react';
import type { Meta, StoryObj } from '@storybook/react';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';

import { Table } from './Table';

type Table = {
  firstName: string;
  lastName: string;
};

const data = [
  {
    firstName: 'Tony',
    lastName: 'Stark',
  },
];

const columnHelper = createColumnHelper<Table>();

const defaultColumns = [
  columnHelper.accessor('firstName', {
    header: () => <span>First Name</span>,
    cell: (info) => <p className="text-basis">{info.getValue()}</p>,
  }),
  columnHelper.accessor('lastName', {
    header: () => <span>Last Name</span>,
    cell: (info) => <p className="text-basis">{info.getValue()}</p>,
  }),
];

const meta = {
  title: 'Components/Table',
  component: Table,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof Table>;

export default meta;

type Story = StoryObj<typeof Table>;

export const Default: Story = {
  render: () => {
    const tableContainerRef = createRef<HTMLDivElement>();
    return (
      <Table
        options={{
          data: data,
          columns: defaultColumns,
          getCoreRowModel: getCoreRowModel(),
        }}
        tableContainerRef={tableContainerRef}
        blankState={<p className="text-basis">No names</p>}
      />
    );
  },
};

export const Empty: Story = {
  render: () => {
    const tableContainerRef = createRef<HTMLDivElement>();
    return (
      <Table
        options={{
          data: [],
          columns: defaultColumns,
          getCoreRowModel: getCoreRowModel(),
        }}
        tableContainerRef={tableContainerRef}
        blankState={<p className="text-basis">No names</p>}
      />
    );
  },
};
