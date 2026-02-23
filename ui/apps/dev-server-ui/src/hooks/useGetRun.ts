import { useCallback, useState } from 'react';
import type { GetRunPayload } from '@inngest/components/SharedContext/useGetRun';

import { client } from '@/store/baseApi';
import { GetRunDocument, type GetRunQuery } from '@/store/generated';

export function useGetRun() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error>();

  return useCallback(async ({ runID, preview }: GetRunPayload) => {
    setLoading(true);
    setError(undefined);
    const data: GetRunQuery = await client.request(GetRunDocument, { runID, preview });
    const run = data.run;

    if (!run) {
      throw new Error('missing run');
    }

    const fn = run.function;

    const app = {
      ...fn.app,
      externalID: fn.app.name,
    };

    const trace = run.trace;
    if (!trace) {
      throw new Error('missing trace');
    }

    return {
      data: {
        ...run,
        app,
        id: runID,
        fn,
        trace,
        deferredRuns: run.deferredRuns ?? undefined,
        deferGroupName: run.deferGroupName ?? undefined,
        parentRunID: run.parentRunID ?? undefined,
      },
      loading,
      error,
    };
  }, []);
}
