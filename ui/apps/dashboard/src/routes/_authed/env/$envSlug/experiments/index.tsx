import { useCallback, useEffect, useState } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';

import { InlineCode } from '@inngest/components/Code';
import { ExperimentsTable } from '@inngest/components/Experiments';
import { Header } from '@inngest/components/Header/Header';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';

import NotFound from '@/components/Error/NotFound';
import { useExperimentsList } from '@/components/Experiments/useExperiments';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute('/_authed/env/$envSlug/experiments/')({
  component: ExperimentsComponent,
});

function ExperimentsInfo() {
  return (
    <Info
      text="View and compare experiment variants across your functions."
      action={
        <Link
          href="https://www.inngest.com/docs/features/step-experimentation"
          target="_blank"
        >
          Learn about experiments
        </Link>
      }
    />
  );
}

export default function ExperimentsComponent() {
  const { envSlug } = Route.useParams();
  const navigate = useNavigate();
  const [isMounted, setIsMounted] = useState(false);
  useEffect(() => {
    setIsMounted(true);
  }, []);

  const experimentsEnabled = useBooleanFlag('experimentation-steps');

  const { data, isPending, error, refetch } = useExperimentsList({
    enabled: isMounted && experimentsEnabled.value,
  });

  const handleRowClick = useCallback(
    (experimentName: string) => {
      navigate({
        to: pathCreator.experiment({ envSlug, experimentName }),
      });
    },
    [navigate, envSlug],
  );

  if (experimentsEnabled.isReady && !experimentsEnabled.value) {
    return <NotFound />;
  }

  return (
    <>
      <Header
        breadcrumb={[{ text: 'All experiments' }]}
        infoIcon={<ExperimentsInfo />}
      />
      <ExperimentsTable
        key={envSlug}
        data={data}
        isPending={isPending || !isMounted}
        error={error}
        refetch={refetch}
        onRowClick={handleRowClick}
        emptyDescription={
          <>
            To define an experiment, use{' '}
            <InlineCode>group.experiment()</InlineCode> on your Inngest
            function.
          </>
        }
      />
    </>
  );
}
