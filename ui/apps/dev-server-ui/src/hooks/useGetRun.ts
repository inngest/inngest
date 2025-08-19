import { useCallback } from 'react';
import type { GetRunPayload } from '@inngest/components/SharedContext/useGetRun';

import { client } from '@/store/baseApi';
import { GetRunDocument, type GetRunQuery } from '@/store/generated';

export function useGetRun() {
  return useCallback(async ({ runID, preview }: GetRunPayload) => {
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
      },
      loading: false,
      error: undefined,
    };
  }, []);
}
