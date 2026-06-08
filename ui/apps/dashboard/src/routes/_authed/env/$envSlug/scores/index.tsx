import { Header } from '@inngest/components/Header/Header';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';
import { createFileRoute } from '@tanstack/react-router';

import NotFound from '@/components/Error/NotFound';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { ScoresDashboard } from '@/components/Scores/Dashboard';

export const Route = createFileRoute('/_authed/env/$envSlug/scores/')({
  component: ScoresComponent,
});

function ScoresInfo() {
  return (
    <Info
      text="View score variants across your functions."
      action={
        <Link
          href="https://www.inngest.com/docs/features/scoring" // TODO: actual docs URL
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
  // TODO: this likely needs its own flag
  const scoresEnabled = useBooleanFlag('experimentation-steps');

  if (scoresEnabled.isReady && !scoresEnabled.value) {
    return <NotFound />;
  }

  return (
    <>
      <Header breadcrumb={[{ text: 'Scores' }]} infoIcon={<ScoresInfo />} />
      <div id="chart-tooltip" className="z-[1000]" />
      <div className="bg-canvasBase mx-auto flex h-full w-full flex-col">
        <ScoresDashboard envSlug={envSlug} />
      </div>
    </>
  );
}
