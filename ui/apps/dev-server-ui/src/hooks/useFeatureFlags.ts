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

/**
 * Creates a Dev Server URL from a path. If Dev Server host is unknown, it
 * returns the path.
 */
function createDevServerURL(path: string) {
  const host = process.env.NEXT_PUBLIC_API_BASE_URL;
  if (!host) {
    return path;
  }
  return new URL(path, host).toString();
}
