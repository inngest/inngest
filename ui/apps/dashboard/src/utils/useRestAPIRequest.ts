'use client';

import { useEffect, useState } from 'react';
import { useAuth } from '@clerk/nextjs';
import {
  baseFetchSkipped,
  baseFetchSucceeded,
  baseInitialFetchFailed,
  baseInitialFetchLoading,
  type FetchResult,
} from '@inngest/components/types/fetch';

export function useRestAPIRequest<T>({
  url,
  method,
  pause = false,
}: {
  url: string | URL | null;
  method: string;
  pause?: boolean;
}): Omit<FetchResult<T, { skippable: true }>, 'refetch'> {
  const { getToken } = useAuth();
  const [data, setData] = useState<any>();
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<Error>();

  useEffect(() => {
    async function request() {
      if (!url || pause) return;
      setIsLoading(true);
      const sessionToken = await getToken();
      if (!sessionToken) {
        setIsLoading(false);

        // TODO: Does this need to be changed for Vercel Marketplace? Vercel
        // Marketplace users don't auth with Clerk.
        return; // TODO - Handle no auth
      }
      const response = await fetch(url, {
        method,
        headers: {
          Authorization: `Bearer ${sessionToken}`,
        },
      });
      if (!response.ok || response.status >= 400) {
        setData(null);
        setIsLoading(false);
        try {
          const data = await response.json();
          const error = data.error || response.statusText;
          return setError(new Error(error));
        } catch (err) {
          return setError(new Error(response.statusText));
        }
      }
      const data = await response.json();

      setData(data);
      setIsLoading(false);
    }
    request();
  }, [getToken, url, method, pause]);

  if (isLoading) {
    return {
      ...baseInitialFetchLoading,
      isLoading: true,
    };
  }

  if (error) {
    return {
      ...baseInitialFetchFailed,
      error,
    };
  }

  if (!!pause) {
    return {
      ...baseFetchSkipped,
    };
  }

  return {
    ...baseFetchSucceeded,
    data,
  };
}
