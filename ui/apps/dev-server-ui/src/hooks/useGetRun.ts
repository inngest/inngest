import { useCallback } from 'react';

import { client } from '@/store/baseApi';
import { GetRunDocument, type GetRunQuery } from '@/store/generated';

export function useGetRun() {
  return useCallback(async (runID: string) => {
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
      ...run,
      app,
      id: runID,
      fn,
      trace,
    };
  }, []);
}
