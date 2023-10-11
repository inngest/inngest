'use client';

import Button from '@/components/Button';

export default function PaymentsButton() {
  function scrollToPayments() {
    document.getElementById('payments')?.scrollIntoView();
  }
  return (
    <Button onClick={scrollToPayments} variant="secondary" className="mt-4">
      View All Payments
    </Button>
  );
}
