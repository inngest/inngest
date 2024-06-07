'use client';

import { Button } from '@inngest/components/Button';

export default function PaymentsButton() {
  function scrollToPayments() {
    document.getElementById('payments')?.scrollIntoView();
  }
  return (
    <Button
      onClick={scrollToPayments}
      appearance="outlined"
      className="mt-4"
      label="View All Payments"
    />
  );
}
