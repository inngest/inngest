import { createFileRoute } from '@tanstack/react-router';

import { ScoresPage } from '@/components/AI/ScoresPage';

export const Route = createFileRoute('/_dashboard/ai/scores/')({
  component: ScoresPage,
});
