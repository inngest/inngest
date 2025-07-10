import { useCallback, useState } from 'react';
import type { GetRunPayload } from '@inngest/components/SharedContext/useGetRun';

import { client } from '@/store/baseApi';
import { GetRunDocument, type GetRunQuery } from '@/store/generated';

export function useRun() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error>();

  return useCallback(async ({ runID }: GetRunPayload) => {
    setLoading(true);
    setError(undefined);
    const data: GetRunQuery = await client.request(GetRunDocument, { runID });
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
      },
      loading,
      error,
    };
  }, []);
}
