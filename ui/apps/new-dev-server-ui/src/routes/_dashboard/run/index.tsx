import { createFileRoute } from '@tanstack/react-router'
import { RunDetailsV3 } from '@inngest/components/RunDetailsV3/RunDetailsV3'
import { useBooleanFlag } from '@inngest/components/SharedContext/useBooleanFlag'
import { useSearchParam } from '@inngest/components/hooks/useNewSearchParams'
import { cn } from '@inngest/components/utils/classNames'

import { useGetTrigger } from '@/hooks/useGetTrigger'

export const Route = createFileRoute('/_dashboard/run/')({
  component: RunComponent,
})

function RunComponent() {
  const { booleanFlag } = useBooleanFlag()
  const { value: pollingDisabled, isReady: pollingFlagReady } = booleanFlag(
    'polling-disabled',
    false,
  )
  const [runID] = useSearchParam('runID')
  const getTrigger = useGetTrigger()

  if (!runID) {
    throw new Error('missing runID in search params')
  }

  return (
    <div className={cn('bg-canvasBase overflow-y-auto pt-8')}>
      <RunDetailsV3
        standalone
        getTrigger={getTrigger}
        pollInterval={pollingFlagReady && pollingDisabled ? 0 : 2500}
        runID={runID}
        newStack={true}
      />
    </div>
  )
}
