'use client';

import type React from 'react';
import { useQuery } from 'urql';

import Placeholder from '@/components/Placeholder';
import Table from '@/components/Table';
import { graphql } from '@/gql';
import { day } from '@/utils/date';
import PaymentIcon from './PaymentIcons';

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
  status: React.ReactNode;
  description: React.ReactNode | string;
  createdAt: React.ReactNode | string;
  amount: React.ReactNode;
  url: React.ReactNode;
};

const loadingPlaceholder = (
  <div className="flex">
    <Placeholder className="mx-1 my-1 h-2 w-full max-w-[120px] bg-slate-200" />
  </div>
);

const loadingRows: TableRow[] = [1, 2, 3].map(() => ({
  status: <></>,
  description: loadingPlaceholder,
  createdAt: loadingPlaceholder,
  amount: loadingPlaceholder,
  url: loadingPlaceholder,
}));

export default function Payments() {
  const [{ data, fetching }] = useQuery({
    query: GetPaymentIntentsDocument,
  });
  const payments = data?.account.paymentIntents || [];

  const paymentTableData = fetching
    ? loadingRows
    : payments.map(
        (payment): TableRow => ({
          status: <PaymentIcon status={payment.status} />,
          description: payment.description,
          createdAt: day(payment.createdAt),
          amount:
            payment.status === 'canceled' ? (
              <span className="text-slate-400">{payment.amountLabel}</span>
            ) : (
              payment.amountLabel
            ),
          url: payment.invoiceURL ? (
            <a
              href={payment.invoiceURL}
              target="_blank"
              className="font-semibold text-indigo-500 hover:text-indigo-800 hover:underline"
            >
              View
            </a>
          ) : null,
        })
      );

  return (
    <Table
      columns={[
        { key: 'status', className: 'w-14' },
        { key: 'description', label: 'Description' },
        { key: 'amount', label: 'Amount' },
        { key: 'createdAt', label: 'Created At' },
        { key: 'url', label: 'Invoice', className: 'w-20' },
      ]}
      data={paymentTableData}
      empty="You have no prior payments"
    />
  );
}
