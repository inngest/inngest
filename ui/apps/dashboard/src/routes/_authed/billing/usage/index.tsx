import { createFileRoute } from '@tanstack/react-router';

import { UsagePage } from '@/components/Billing/UsagePage';

type UsageSearchParams = {
  previous?: boolean;
};

export const Route = createFileRoute('/_authed/billing/usage/')({
  component: BillingUsagePage,
  validateSearch: (search: Record<string, unknown>): UsageSearchParams => {
    return {
      previous: search.previous === 'true' || search.previous === true,
    };
  },
});

function BillingUsagePage() {
  const { previous } = Route.useSearch();

  return <UsagePage previous={previous} />;
}
