import { createFileRoute } from '@tanstack/react-router';

import { ExperimentsPage } from '@/components/AI/ExperimentsPage';

export const Route = createFileRoute('/_dashboard/ai/experiments/')({
  component: ExperimentsPage,
});
