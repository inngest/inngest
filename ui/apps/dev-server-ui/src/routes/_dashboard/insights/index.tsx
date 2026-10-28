import { createFileRoute } from '@tanstack/react-router';

import { DuckDBGuard } from '@/components/DuckDBGuard';
import InsightsPage from '@/components/Insights/InsightsPage';

export const Route = createFileRoute('/_dashboard/insights/')({
  component: GuardedInsightsPage,
});

function GuardedInsightsPage() {
  return (
    <DuckDBGuard>
      <InsightsPage />
    </DuckDBGuard>
  );
}
