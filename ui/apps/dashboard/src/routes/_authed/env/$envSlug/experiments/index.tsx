import { useCallback, useEffect, useRef, useState } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';

import { InlineCode } from '@inngest/components/Code';
import {
  ExperimentsEmptyState,
  ExperimentsTable,
  type ExperimentListItem,
} from '@inngest/components/Experiments';
import { Header } from '@inngest/components/Header/Header';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';

import FeedbackFloatingButton from '@/components/Feedback/FeedbackFloatingButton';
import { useExperimentsList } from '@/components/Experiments/useExperiments';
import {
  trackExperimentDocsLinkOpened,
  trackExperimentEmptyStateDocsLinkOpened,
  trackExperimentEmptyStateExampleCopied,
  trackExperimentEmptyStatePromptCopied,
  trackExperimentEmptyStateViewed,
  trackExperimentsListViewed,
} from '@/components/Experiments/tracking';
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
          href="https://www.inngest.com/docs/features/inngest-functions/steps-workflows/step-experiments"
          target="_blank"
          onClick={() => trackExperimentDocsLinkOpened()}
        >
          Learn about experiments
        </Link>
      }
    />
  );
}

function ExperimentsComponent() {
  const { envSlug } = Route.useParams();
  const navigate = useNavigate();
  const [isMounted, setIsMounted] = useState(false);
  useEffect(() => {
    setIsMounted(true);
  }, []);

  const { data, isPending, error, refetch } = useExperimentsList({
    enabled: isMounted,
  });

  const hasTrackedListViewed = useRef(false);
  useEffect(() => {
    if (hasTrackedListViewed.current) return;
    if (isPending || error || !Array.isArray(data)) return;

    hasTrackedListViewed.current = true;
    trackExperimentsListViewed({
      experimentCount: data.length,
      functionCount: new Set(data.map((item) => item.functionId)).size,
    });
  }, [data, isPending, error]);

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

  const showEmptyState =
    !isPending && !error && Array.isArray(data) && data.length === 0;

  if (showEmptyState) {
    return (
      <>
        <Header
          breadcrumb={[{ text: 'All experiments' }]}
          infoIcon={<ExperimentsInfo />}
        />
        <ExperimentsEmptyState
          onViewed={trackExperimentEmptyStateViewed}
          onDocsLinkClick={trackExperimentEmptyStateDocsLinkOpened}
          onPromptCopy={trackExperimentEmptyStatePromptCopied}
          onExampleCopy={trackExperimentEmptyStateExampleCopied}
        />
        <FeedbackFloatingButton />
      </>
    );
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
      <FeedbackFloatingButton />
    </>
  );
}
