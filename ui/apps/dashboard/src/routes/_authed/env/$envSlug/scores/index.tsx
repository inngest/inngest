import { lazy } from 'react';
import { Header } from '@inngest/components/Header/Header';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';
import { RefreshButton } from '@inngest/components/Refresh/RefreshButton';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

const ScoresDashboard = lazy(() =>
  import('@/components/Scores/Dashboard').then((m) => ({
    default: m.ScoresDashboard,
  })),
);

export const Route = createFileRoute('/_authed/env/$envSlug/scores/')({
  component: ScoresComponent,
});

function ScoresInfo() {
  return (
    <Info
      text="View score variants across your functions."
      action={
        <Link
          href="https://www.inngest.com/docs/features/inngest-functions/steps-workflows/scoring"
          target="_blank"
        >
          Learn about scores
        </Link>
      }
    />
  );
}

function ScoresComponent() {
  const { envSlug } = Route.useParams();

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Scores' }]}
        infoIcon={<ScoresInfo />}
        action={<RefreshButton />}
      />
      <div id="chart-tooltip" className="z-[1000]" />
      <div className="bg-canvasBase mx-auto flex h-full w-full flex-col">
        <ClientOnly>
          <ScoresDashboard envSlug={envSlug} />
        </ClientOnly>
      </div>
    </>
  );
}
