import { gql } from 'urql';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';
import { type AccountPaymentStatus } from './types';

type PaymentStatusResult = {
  account: {
    id: string;
    // Null when the account is in good standing.
    paymentStatus: AccountPaymentStatus | null;
  };
};

// Plain urql `gql` document (not the codegen `graphql()` tag) so referencing a
// field that isn't in the introspected schema yet can't break codegen for the
// whole app. Swap to `graphql()` + generated types once `account.paymentStatus`
// ships.
const paymentStatusQuery = gql<PaymentStatusResult, Record<string, never>>`
  query PaymentStatus {
    account {
      id
      paymentStatus {
        severity
        stage
        amountDueLabel
        daysPastDue
        hasFailedPayment
        actionDate
        pendingAction
        resolveURL
        overdueInvoices {
          id
          amountLabel
          dueAt
          daysPastDue
          status
          invoiceURL
          failureReason
        }
      }
    }
  }
`;

// Shared between the global layout banner and the /billing detail banner. They
// issue the same client-side query so urql dedupes them into a single request
// and both surfaces stay in sync.
export function usePaymentStatus(): AccountPaymentStatus | null {
  // Gated so no requests fire before the API field ships; flip the flag on once
  // `account.paymentStatus` is live.
  const { value: enabled } = useBooleanFlag('overdue-invoice-banner');

  const res = useSkippableGraphQLQuery({
    query: paymentStatusQuery,
    variables: {},
    skip: !enabled,
    // Poll every 5 minutes so a resolved payment clears the banner without a
    // manual refresh, even for users sitting on a single page.
    pollIntervalInMilliseconds: 5 * 60_000,
  });

  // Payment banners are optional UI: while loading, skipped, or errored there's
  // no data, so we degrade to rendering nothing.
  if (res.data) {
    return res.data.account.paymentStatus;
  }
  return null;
}
