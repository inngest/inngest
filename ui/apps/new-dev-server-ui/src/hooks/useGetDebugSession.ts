import type { GetDebugSessionPayload } from '@inngest/components/SharedContext/useGetDebugSession'
import { useCallback } from 'react'

import { client } from '@/store/baseApi'
import {
  GetDebugSessionDocument,
  type GetDebugSessionQuery,
} from '@/store/generated'

export function useGetDebugSession() {
  return useCallback(
    async ({ functionSlug, debugSessionID, runID }: GetDebugSessionPayload) => {
      const { debugSession }: GetDebugSessionQuery =
        await client.request<GetDebugSessionQuery>(GetDebugSessionDocument, {
          query: {
            functionSlug,
            debugSessionID,
            runID,
          },
        })

      return {
        data: { debugRuns: debugSession?.debugRuns || [] },
        loading: false,
        error: undefined,
      }
    },
    [],
  )
}
