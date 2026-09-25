import { createFileRoute } from '@tanstack/react-router';

import InsightsPage from '@/components/Insights/InsightsPage';

export const Route = createFileRoute('/_dashboard/insights/')({
  component: InsightsPage,
});
