import { useEffect, useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';

import { ExperimentsTable } from '@inngest/components/Experiments';
import { Header } from '@inngest/components/Header/Header';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';

import { useExperimentsList } from '@/components/Experiments/useExperiments';

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
  const [isMounted, setIsMounted] = useState(false);
  useEffect(() => {
    setIsMounted(true);
  }, []);

  const { data, isPending, error, refetch } = useExperimentsList({
    enabled: isMounted,
  });

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
      />
    </>
  );
}
