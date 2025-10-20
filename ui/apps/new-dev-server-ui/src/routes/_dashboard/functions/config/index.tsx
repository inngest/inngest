import { useSearchParam } from '@inngest/components/hooks/useNewSearchParams'
import { createFileRoute, useNavigate } from '@tanstack/react-router'

import { Suspense } from 'react'

import { SlideOver } from '@inngest/components/SlideOver/SlideOver'
import { FunctionConfigurationContainer } from '@/components/Function/FunctionConfigurationContainer'

export const Route = createFileRoute('/_dashboard/functions/config/')({
  component: FunctionComponent,
})

const FunctionConfig = () => {
  const navigate = useNavigate()
  const [functionSlug] = useSearchParam('slug')

  const closeSlideOver = () => {
    navigate({ to: '/functions' })
  }

  if (!functionSlug) return

  return (
    <SlideOver size="fixed-500" onClose={closeSlideOver}>
      <FunctionConfigurationContainer
        onClose={closeSlideOver}
        functionSlug={functionSlug}
      />
    </SlideOver>
  )
}

function FunctionComponent() {
  return (
    <Suspense>
      <FunctionConfig />
    </Suspense>
  )
}
