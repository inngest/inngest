'use client';

import { RunDetails } from '@inngest/components/RunDetailsV2';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import LoadingIcon from '@/icons/LoadingIcon';
import { useCancelRun } from '@/queries/useCancelRun';
import { useRun } from './useRun';

type Props = {
  params: {
    runID: string;
  };
};

export default function Page({ params }: Props) {
  const runID = decodeURIComponent(params.runID);
  const env = useEnvironment();
  const cancelRun = useCancelRun({ envID: env.id, runID });

  const res = useRun({ envID: env.id, runID });
  if (res.error) {
    throw res.error;
  }
  if (res.isLoading && !res.data) {
    return <Loading />;
  }
  const { run, trace } = res.data;

  async function getOutput() {
    return null;
  }

  return (
    <div className="overflow-y-auto">
      <RunDetails
        app={run.function.app}
        cancelRun={cancelRun}
        fn={run.function}
        getOutput={getOutput}
        run={{
          id: params.runID,
          output: null,
          trace,
        }}
      />
    </div>
  );
}

function Loading() {
  return (
    <div className="flex h-full w-full items-center justify-center">
      <div className="flex flex-col items-center justify-center gap-2">
        <LoadingIcon />
        <div>Loading</div>
      </div>
    </div>
  );
}
