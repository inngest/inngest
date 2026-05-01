import { createFileRoute } from '@tanstack/react-router';
import { useCallback, useMemo } from 'react';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';

import NotFound from '@/components/Error/NotFound';
import { ExperimentDetailPage } from '@/components/Experiments/ExperimentDetailPage';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import {
  getExperimentUrlState,
  hasExperimentTimeRangeSearch,
  setExperimentTimeRangeSearch,
  setExperimentVariantsSearch,
  validateExperimentDetailSearch,
} from '@/lib/experiments/urlState';
import { useFunction } from '@/queries/functions';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/experiments/$functionSlug/$experimentName/',
)({
  component: ExperimentDetailRoute,
  validateSearch: validateExperimentDetailSearch,
});

function ExperimentDetailRoute() {
  const { functionSlug, experimentName } = Route.useParams();
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  const experimentsEnabled = useBooleanFlag('experimentation-steps');
  const urlState = useMemo(() => getExperimentUrlState(search), [search]);
  const hasTimeRangeSearch = hasExperimentTimeRangeSearch(search);

  const updateTimeRange = useCallback(
    (range: RangeChangeProps) => {
      navigate({
        search: (prev) => setExperimentTimeRangeSearch(prev, range),
        replace: true,
      });
    },
    [navigate],
  );

  const updateSelectedVariants = useCallback(
    (variants: string[]) => {
      navigate({
        search: (prev) => setExperimentVariantsSearch(prev, variants),
        replace: true,
      });
    },
    [navigate],
  );

  const decodedFunctionSlug = decodeURIComponent(functionSlug);
  const [{ data, fetching }] = useFunction({
    functionSlug: decodedFunctionSlug,
  });
  const fn = data?.workspace.workflow;

  if (experimentsEnabled.isReady && !experimentsEnabled.value) {
    return <NotFound />;
  }

  if (!fetching && !fn) {
    return <NotFound />;
  }

  if (!fn) {
    return null;
  }

  return (
    <ExperimentDetailPage
      experimentName={decodeURIComponent(experimentName)}
      functionID={fn.id}
      functionName={fn.name}
      functionSlug={decodedFunctionSlug}
      timeRange={urlState.timeRange}
      hasTimeRangeSearch={hasTimeRangeSearch}
      onTimeRangeChange={updateTimeRange}
      selectedVariants={urlState.selectedVariants}
      onSelectedVariantsChange={updateSelectedVariants}
    />
  );
}
