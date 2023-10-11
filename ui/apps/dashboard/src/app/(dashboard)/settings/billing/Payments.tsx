import Table from '@/components/Table';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
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

export default async function Invoices() {
  const response = await graphqlAPI.request(GetPaymentIntentsDocument);
  const payments = response.account.paymentIntents;

  const paymentTableData = payments.map((payment) => ({
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
  }));

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
