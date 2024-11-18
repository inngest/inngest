'use client';

import { NewButton } from '@inngest/components/Button';

export default function PaymentsButton() {
  return (
    <NewButton
      href="/billing/payments?ref=app-billing-usage"
      appearance="outlined"
      className="mt-4"
      label="View All Payments"
    />
  );
}
