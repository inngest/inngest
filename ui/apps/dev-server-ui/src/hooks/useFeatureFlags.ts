import { useEffect, useState } from 'react';
import { createDevServerURL } from '@/utils/devServer';

interface FeatureFlags {
  FEATURE_CEL_SEARCH?: boolean;
  FEATURE_EVENTS?: boolean;
}

export function useFeatureFlags() {
  const [featureFlags, setFeatureFlags] = useState<FeatureFlags>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    async function fetchFeatureFlags() {
      try {
        const response = await fetch(createDevServerURL('/dev'));
        if (!response.ok) {
          throw new Error('Failed to fetch feature flags');
        }
        const data = await response.json();
        setFeatureFlags(data.features || {});
      } catch (err) {
        setError(err instanceof Error ? err : new Error('An error occurred'));
      } finally {
        setLoading(false);
      }
    }

    fetchFeatureFlags();
  }, []);

  return { featureFlags, loading, error };
}
