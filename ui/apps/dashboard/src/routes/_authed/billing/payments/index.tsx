import { createFileRoute } from '@tanstack/react-router';

import Payments from '@/components/Billing/Payments/Payments';

export const Route = createFileRoute('/_authed/billing/payments/')({
  component: BillingPaymentsPage,
});

function BillingPaymentsPage() {
  return <Payments />;
}
