import { useCallback, useState } from 'react'
import type { GetRunTracePayload } from '@inngest/components/SharedContext/useGetRunTrace'

import { client } from '@/store/baseApi'
import { GetRunTraceDocument, type GetRunTraceQuery } from '@/store/generated'

export const useGetRunTrace = () => {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<Error>()

  return useCallback(async ({ runID }: GetRunTracePayload) => {
    setLoading(true)
    setError(undefined)
    const data: GetRunTraceQuery = await client.request(GetRunTraceDocument, {
      runID,
    })

    return {
      data: data.runTrace,
      loading,
      error,
    }
  }, [])
}
