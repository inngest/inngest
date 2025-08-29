import { useCallback } from 'react';
import type { GetDebugRunPayload } from '@inngest/components/SharedContext/useGetDebugRun';

import { client } from '@/store/baseApi';
import { GetDebugRunDocument, type GetDebugRunQuery } from '@/store/generated';

export function useGetDebugRun() {
  return useCallback(async ({ functionSlug, debugRunID, runID }: GetDebugRunPayload) => {
    const { debugRun }: GetDebugRunQuery = await client.request<GetDebugRunQuery>(
      GetDebugRunDocument,
      {
        query: {
          functionSlug,
          debugRunID,
          runID,
        },
      }
    );

    return { data: debugRun ?? undefined, loading: false, error: undefined };
  }, []);
}
