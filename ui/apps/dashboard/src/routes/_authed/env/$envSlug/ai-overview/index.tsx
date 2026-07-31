import { lazy } from 'react';
import { Header } from '@inngest/components/Header/Header';
import { RefreshButton } from '@inngest/components/Refresh/RefreshButton';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import FeedbackFloatingButton from '@/components/Feedback/FeedbackFloatingButton';

const AIOverviewDashboard = lazy(() =>
  import('@/components/AIOverview/Dashboard').then((m) => ({
    default: m.AIOverviewDashboard,
  })),
);

export const Route = createFileRoute('/_authed/env/$envSlug/ai-overview/')({
  component: AIOverviewComponent,
});

function AIOverviewComponent() {
  const { envSlug } = Route.useParams();

  return (
    <>
      <Header
        breadcrumb={[{ text: 'AI Overview' }]}
        action={<RefreshButton />}
      />
      <div className="bg-canvasBase mx-auto flex h-full w-full flex-col">
        <ClientOnly>
          <AIOverviewDashboard envSlug={envSlug} />
        </ClientOnly>
      </div>
      <FeedbackFloatingButton />
    </>
  );
}
