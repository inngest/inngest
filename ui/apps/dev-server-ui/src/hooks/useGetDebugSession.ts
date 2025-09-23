import { useCallback } from 'react';
import type { GetDebugSessionPayload } from '@inngest/components/SharedContext/useGetDebugSession';

import { client } from '@/store/baseApi';
import {
  GetDebugSessionDocument,
  GetTraceResultDocument,
  type GetDebugSessionQuery,
  type GetTraceResultQuery,
} from '@/store/generated';

export function useGetDebugSession() {
  return useCallback(async ({ functionSlug, debugSessionID, runID }: GetDebugSessionPayload) => {
    const { debugSession }: GetDebugSessionQuery = await client.request<GetDebugSessionQuery>(
      GetDebugSessionDocument,
      {
        query: {
          functionSlug,
          debugSessionID,
          runID,
        },
      }
    );

    return { data: { debugRuns: debugSession?.debugRuns || [] }, loading: false, error: undefined };
  }, []);
}
