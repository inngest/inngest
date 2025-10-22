import { useCallback } from 'react'
import type { CancelRunResult } from '@inngest/components/SharedContext/useCancelRun'

import { useCancelRunMutation } from '@/store/generated'

export const useCancelRun = () => {
  const [cancelRun] = useCancelRunMutation()

  return useCallback(
    async ({ runID }: { runID?: string }): Promise<CancelRunResult> => {
      try {
        const res = await cancelRun({ runID })

        if ('error' in res) {
          throw res.error
        }

        return res
      } catch (error) {
        console.error('error cancelling function run', error)
        return {
          error:
            error instanceof Error
              ? error
              : new Error('Error cancelling function run'),
          data: undefined,
        }
      }
    },
    [cancelRun],
  )
}
