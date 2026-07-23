import { lazy } from 'react';
import { Header } from '@inngest/components/Header/Header';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';
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

function AIOverviewInfo() {
  return (
    <Info
      text="How your AI is performing across every run in this environment."
      action={
        <Link
          href="https://www.inngest.com/docs/features/inngest-functions/steps-workflows/ai"
          target="_blank"
        >
          Learn about AI Overview
        </Link>
      }
    />
  );
}

function AIOverviewComponent() {
  const { envSlug } = Route.useParams();

  return (
    <>
      <Header
        breadcrumb={[{ text: 'AI Overview' }]}
        infoIcon={<AIOverviewInfo />}
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
