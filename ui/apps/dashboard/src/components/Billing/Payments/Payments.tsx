'use client';

import { useMemo, useRef } from 'react';
import { type Route } from 'next';
import { Link } from '@inngest/components/Link';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { Table, TextCell } from '@inngest/components/Table';
import { formatDayString } from '@inngest/components/utils/date';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';
import { useQuery } from 'urql';

import PaymentStatusPill from '@/components/Billing/Payments/PaymentStatusPill';
import { graphql } from '@/gql';

const GetPaymentIntentsDocument = graphql(`
  query GetPaymentIntents {
    account {
      paymentIntents {
        status
        createdAt
        amountLabel
        description
        invoiceURL
      }
    }
  }
`);

type TableRow = {
  status: string;
  description: string;
  createdAt: string;
  amount: React.ReactNode;
  url: React.ReactNode;
};

const columnHelper = createColumnHelper<TableRow>();

const columns = [
  columnHelper.accessor('status', {
    header: () => <span>Status</span>,
    cell: (props) => <PaymentStatusPill status={props.getValue()} />,
  }),
  columnHelper.accessor('description', {
    header: () => <span>Description</span>,
    cell: (props) => <TextCell>{props.getValue()}</TextCell>,
  }),
  columnHelper.accessor('amount', {
    header: () => <span>Amount</span>,
    cell: (props) => {
      const isCanceled = props.row.original.status === 'canceled';
      return (
        <TextCell>
          <span className={isCanceled ? 'text-muted' : ''}>{props.getValue()}</span>
        </TextCell>
      );
    },
  }),
  columnHelper.accessor('createdAt', {
    header: () => <span>Created at</span>,
    cell: (props) => <TextCell>{props.getValue()}</TextCell>,
  }),
  columnHelper.accessor('url', {
    header: () => <span />,
    cell: (props) => {
      const url = props.getValue();
      const requiresConfirmation = props.row.original.status === 'requires_confirmation';
      if (url) {
        return (
          <Link href={url as Route} size="small" target="_blank">
            {requiresConfirmation ? 'Pay invoice' : 'View invoice'}
          </Link>
        );
      }
      return null;
    },
  }),
];

export default function Payments() {
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const [{ data, fetching }] = useQuery({
    query: GetPaymentIntentsDocument,
  });

  const payments = useMemo(() => data?.account.paymentIntents || [], [data]);

  const tableColumns = useMemo(
    () =>
      fetching
        ? columns.map((column) => ({
            ...column,
            cell: () => <Skeleton className="my-1 block h-4" />,
          }))
        : columns,
    [fetching]
  );

  const paymentTableData = useMemo(() => {
    if (fetching) {
      return Array(columns.length)
        .fill(null)
        .map((_, index) => {
          return {
            // Need an ID to avoid "missing key" errors when rendering rows
            id: index,
          };
        }) as unknown as TableRow[]; // Casting is bad but we need to do this for the loading skeleton
    }

    return payments.map(
      (payment): TableRow => ({
        status: payment.status,
        description: payment.description,
        createdAt: formatDayString(new Date(payment.createdAt)),
        amount: payment.amountLabel,
        url: payment.invoiceURL,
      })
    );
  }, [fetching, payments]);

  return (
    <main
      className="border-muted min-h-0 overflow-y-auto rounded-md border [&>table]:border-b-0"
      ref={tableContainerRef}
    >
      <Table
        tableContainerRef={tableContainerRef}
        isVirtualized={false}
        options={{
          data: paymentTableData,
          columns: tableColumns,
          getCoreRowModel: getCoreRowModel(),
          enableSorting: false,
        }}
        blankState={!fetching && <p>You have no prior payments</p>}
      />
    </main>
  );
}
