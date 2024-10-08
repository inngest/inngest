import { useEffect, useState } from 'react';

interface FeatureFlags {
  FEATURE_CEL_SEARCH?: boolean;
}

export function useFeatureFlags() {
  const [featureFlags, setFeatureFlags] = useState<FeatureFlags>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    async function fetchFeatureFlags() {
      try {
        const response = await fetch('http://localhost:8288/dev');
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
