'use client';

import { useEffect, useState } from 'react';
import { useAuth } from '@clerk/nextjs';

export function useRestAPIRequest<T>({
  url,
  method,
}: {
  url: string | URL | null;
  method: string;
}): { data: T | undefined; error: Error | null } {
  const { getToken } = useAuth();
  const [data, setData] = useState<any>();
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    async function request() {
      if (!url) return;
      const sessionToken = await getToken();
      if (!sessionToken) return; // TODO - Handle no auth
      const response = await fetch(url, {
        method,
        headers: {
          Authorization: `Bearer ${sessionToken}`,
        },
      });
      if (!response.ok || response.status >= 400) {
        return setError(new Error(response.statusText));
      }
      const data = await response.json();

      setData(data);
    }
    request();
  }, [getToken, url, method]);

  return { data, error };
}
