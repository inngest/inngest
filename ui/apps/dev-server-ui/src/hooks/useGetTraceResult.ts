import { useCallback } from 'react'
import type { GetTraceResultPayload } from '@inngest/components/SharedContext/useGetTraceResult'

import { client } from '@/store/baseApi'
import {
  GetTraceResultDocument,
  type GetTraceResultQuery,
} from '@/store/generated'

export function useGetTraceResult() {
  return useCallback(async ({ traceID, preview }: GetTraceResultPayload) => {
    const data: GetTraceResultQuery = await client.request(
      GetTraceResultDocument,
      {
        traceID,
        preview,
      },
    )

    return data.runTraceSpanOutputByID
  }, [])
}
