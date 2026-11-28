import { useMemo, useRef } from 'react';
import { Link } from '@inngest/components/Link';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { Table, TextCell } from '@inngest/components/Table';
import { formatDayString } from '@inngest/components/utils/date';
import { createColumnHelper } from '@tanstack/react-table';
import { useQuery } from 'urql';

import PaymentStatusPill from '@/components/Billing/Payments/PaymentStatusPill';
import { graphql } from '@/gql';

const GetInvoicesDocument = graphql(`
  query GetInvoices {
    account {
      invoices {
        number
        status
        createdAt
        amountLabel
        invoiceURL
      }
    }
  }
`);

type TableRow = {
  status: string;
  number: string;
  createdAt: string;
  amount: React.ReactNode;
  url: React.ReactNode;
};

const columnHelper = createColumnHelper<TableRow>();

const columns = [
  columnHelper.accessor('status', {
    header: () => <span>Status</span>,
    cell: (props) => <PaymentStatusPill status={props.getValue()} />,
    enableSorting: false,
  }),
  columnHelper.accessor('number', {
    header: () => <span>Invoice</span>,
    cell: (props) => <TextCell>{props.getValue()}</TextCell>,
  }),
  columnHelper.accessor('amount', {
    header: () => <span>Amount</span>,
    cell: (props) => {
      const isVoid = props.row.original.status === 'void';
      return (
        <TextCell>
          <span className={isVoid ? 'text-muted' : ''}>{props.getValue()}</span>
        </TextCell>
      );
    },
    enableSorting: false,
  }),
  columnHelper.accessor('createdAt', {
    header: () => <span>Created at</span>,
    cell: (props) => <TextCell>{props.getValue()}</TextCell>,
    enableSorting: false,
  }),
  columnHelper.accessor('url', {
    header: () => <span />,
    cell: (props) => {
      const url = props.getValue();
      const isOpen = props.row.original.status === 'open';
      if (url) {
        return (
          <Link href={url as string} size="small" target="_blank">
            {isOpen ? 'Pay invoice' : 'View invoice'}
          </Link>
        );
      }
      return null;
    },
    enableSorting: false,
  }),
];

export default function Payments() {
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const [{ data, fetching }] = useQuery({
    query: GetInvoicesDocument,
  });

  const invoices = useMemo(() => data?.account.invoices || [], [data]);

  const tableColumns = useMemo(
    () =>
      fetching
        ? columns.map((column) => ({
            ...column,
            cell: () => <Skeleton className="my-1 block h-4" />,
          }))
        : columns,
    [fetching],
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

    return invoices.map(
      (invoice): TableRow => ({
        status: invoice.status,
        number: invoice.number,
        createdAt: formatDayString(new Date(invoice.createdAt)),
        amount: invoice.amountLabel,
        url: invoice.invoiceURL,
      }),
    );
  }, [fetching, invoices]);

  return (
    <main
      className="border-muted min-h-0 overflow-y-auto rounded-md border [&>table]:border-b-0"
      ref={tableContainerRef}
    >
      <Table
        isLoading={fetching}
        columns={tableColumns}
        data={paymentTableData}
        blankState={!fetching && <p>You have no invoices</p>}
      />
    </main>
  );
}
