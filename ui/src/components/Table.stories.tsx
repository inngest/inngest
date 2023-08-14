import type { Meta, StoryObj } from '@storybook/react';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';

import Table from './Table';

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
    cell: (info) => <p className="text-slate-400">{info.getValue()}</p>,
  }),
  columnHelper.accessor('lastName', {
    header: () => <span>Last Name</span>,
    cell: (info) => <p className="text-slate-400">{info.getValue()}</p>,
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
  args: {
    options: {
      data: data,
      columns: defaultColumns,
      getCoreRowModel: getCoreRowModel(),
    },
    blankState: <p className="text-slate-400">No names</p>,
  },
};

export const Empty: Story = {
  args: {
    options: {
      data: [],
      columns: defaultColumns,
      getCoreRowModel: getCoreRowModel(),
    },
    blankState: <p className="text-slate-400">No names</p>,
  },
};
