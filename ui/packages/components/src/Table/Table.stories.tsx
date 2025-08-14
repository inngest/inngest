import { Table } from '@inngest/components/Table';
import type { Meta, StoryObj } from '@storybook/react';
import { createColumnHelper } from '@tanstack/react-table';

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
} satisfies Meta<typeof Table>;

export default meta;

type Story = StoryObj<typeof Table>;

export const Default: Story = {
  render: () => {
    return (
      <Table
        data={data}
        columns={defaultColumns}
        blankState={<p className="text-basis">No names</p>}
      />
    );
  },
};

export const Empty: Story = {
  render: () => {
    return (
      <Table
        data={[]}
        columns={defaultColumns}
        blankState={<p className="text-basis">No names</p>}
      />
    );
  },
};
