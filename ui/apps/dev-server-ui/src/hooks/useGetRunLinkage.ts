import { useCallback, useState } from 'react';
import type { GetRunLinkagePayload } from '@inngest/components/SharedContext/useGetRunLinkage';

import { client } from '@/store/baseApi';
import {
  GetRunLinkageDocument,
  type GetRunLinkageQuery,
} from '@/store/generated';

export function useGetRunLinkage() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error>();

  return useCallback(async ({ runID }: GetRunLinkagePayload) => {
    setLoading(true);
    setError(undefined);
    const data: GetRunLinkageQuery = await client.request(
      GetRunLinkageDocument,
      {
        runID,
      },
    );
    const run = data.run;

    if (!run) {
      throw new Error('missing run');
    }

    return {
      data: {
        defers: run.defers,
        deferredFrom: run.deferredFrom,
        invokedFrom: run.invokedFrom,
      },
      loading,
      error,
    };
  }, []);
}
