import { useEffect, useState } from 'react'
import type { BooleanFlag } from '@inngest/components/SharedContext/useBooleanFlag.js'

import { createDevServerURL } from './useFeatureFlags'

export const useBooleanFlag = (
  flag: string,
  defaultValue: boolean = false,
): BooleanFlag => {
  const [featureFlags, setFeatureFlags] = useState<Record<string, boolean>>({})
  const [loading, setLoading] = useState(true)

  const loadFeatureFlags = async () => {
    try {
      const response = await fetch(createDevServerURL('/dev'))
      setLoading(false)
      if (!response.ok) {
        throw new Error('feature flag response not ok')
      }
      const data = await response.json()
      setFeatureFlags(data.features || {})
    } catch (err) {
      console.error('error fetching feature flags', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => void loadFeatureFlags(), [])

  return { isReady: !loading, value: featureFlags[flag] ?? defaultValue }
}
