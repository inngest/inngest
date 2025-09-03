import { useCallback } from 'react';
import type { GetDebugRunPayload } from '@inngest/components/SharedContext/useGetDebugRun';

import { client } from '@/store/baseApi';
import { GetDebugRunDocument, type GetDebugRunQuery } from '@/store/generated';

export function useGetDebugRun() {
  return useCallback(async ({ functionSlug, debugRunID, runID }: GetDebugRunPayload) => {
    const { debugRun } = await client.request<GetDebugRunQuery>(GetDebugRunDocument, {
      query: {
        functionSlug,
        debugRunID,
        runID,
      },
    });

    return {
      data: debugRun,
      loading: false,
      error: undefined,
    };
  }, []);
}
