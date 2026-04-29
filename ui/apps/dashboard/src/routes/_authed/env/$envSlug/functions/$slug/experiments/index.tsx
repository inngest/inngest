import { useCallback, useEffect, useState } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';

import { InlineCode } from '@inngest/components/Code';
import {
  ExperimentsTable,
  type ExperimentListItem,
} from '@inngest/components/Experiments';

import NotFound from '@/components/Error/NotFound';
import { useExperimentsList } from '@/components/Experiments/useExperiments';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useFunction } from '@/queries/functions';
import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/functions/$slug/experiments/',
)({
  component: FunctionExperimentsComponent,
});

// Per-function experiments tab. Mirrors the function detail page's other
// tabs (Runs, Replays, Cancellations) and shows only the experiments that
// belong to THIS function — different from /env/$envSlug/experiments which
// is the cross-function overview.
function FunctionExperimentsComponent() {
  const { envSlug, slug } = Route.useParams();
  const navigate = useNavigate();
  const [isMounted, setIsMounted] = useState(false);
  useEffect(() => {
    setIsMounted(true);
  }, []);

  const experimentsEnabled = useBooleanFlag('experimentation-steps');

  // Load the function so we can filter the experiments list by its UUID.
  // The parent route already issued this query; urql dedupes.
  const [{ data: fnData }] = useFunction({
    functionSlug: decodeURIComponent(slug),
  });
  const functionID = fnData?.workspace.workflow?.id ?? null;

  const {
    data: allExperiments,
    isPending,
    error,
    refetch,
  } = useExperimentsList({
    enabled: isMounted && experimentsEnabled.value,
  });

  const data =
    functionID && allExperiments
      ? allExperiments.filter((e) => e.functionId === functionID)
      : allExperiments;

  const handleRowClick = useCallback(
    (row: ExperimentListItem) => {
      navigate({
        to: pathCreator.functionExperiment({
          envSlug,
          functionSlug: row.functionSlug,
          experimentName: row.experimentName,
        }),
      });
    },
    [navigate, envSlug],
  );

  if (experimentsEnabled.isReady && !experimentsEnabled.value) {
    return <NotFound />;
  }

  return (
    <ExperimentsTable
      key={`${envSlug}-${slug}`}
      data={data}
      isPending={isPending || !isMounted}
      error={error}
      refetch={refetch}
      onRowClick={handleRowClick}
      emptyDescription={
        <>
          To define an experiment for this function, use{' '}
          <InlineCode>group.experiment()</InlineCode> in its handler.
        </>
      }
    />
  );
}
