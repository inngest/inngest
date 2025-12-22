import { createFileRoute } from '@tanstack/react-router';

import { UsagePage } from '@/components/Billing/UsagePage';
import type { UsageDimension } from '@/components/Billing/Usage/types';

type UsageSearchParams = {
  previous?: boolean;
  dimension?: UsageDimension;
};

export const Route = createFileRoute('/_authed/billing/usage/')({
  component: BillingUsagePage,
  validateSearch: (search: Record<string, unknown>): UsageSearchParams => {
    return {
      previous: search.previous === 'true' || search.previous === true,
      dimension: search.dimension as UsageDimension,
    };
  },
});

function BillingUsagePage() {
  const { previous, dimension } = Route.useSearch();

  return <UsagePage previous={previous} dimension={dimension} />;
}
