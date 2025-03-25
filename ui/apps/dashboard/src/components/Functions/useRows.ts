import { useEffect, useState } from 'react';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { getFunctionUsagesPage, useFunctionsPage } from '@/queries';
import type { FunctionTableRow } from './FunctionTable';

type Output = {
  error: Error | undefined;
  hasMore: boolean | undefined;
  isLoading: boolean;
  loadMore: () => void;
  rows: FunctionTableRow[];
};

export function useRows({ archived, search }: { archived: boolean; search: string }): Output {
  const client = useClient();
  const env = useEnvironment();
  const [page, setPage] = useState(1);

  // Primary function data. This doesn't include usage data since we'll
  // lazy-load it.
  const [functionsData, setFunctionsData] = useState<{
    lastPage: number;
    rows: FunctionTableRow[];
  }>({
    lastPage: 0,
    rows: [],
  });

  // Reset function data when switching between archived and active.
  useEffect(() => {
    setFunctionsData({
      lastPage: 0,
      rows: [],
    });
  }, [archived, search]);

  const functionsRes = useFunctionsPage({
    archived,
    search,
    envID: env.id,
    page,
  });

  // Append new function data.
  useEffect(() => {
    if (!functionsRes.data || functionsRes.isLoading) {
      return;
    }

    if (functionsRes.data.page.page > functionsData.lastPage) {
      setFunctionsData((prev) => {
        return {
          lastPage: functionsRes.data.page.page,
          rows: [...prev.rows, ...functionsRes.data.functions],
        };
      });

      try {
        getFunctionUsagesPage({
          archived,
          client,
          envID: env.id,
          page: functionsRes.data.page.page,
        }).then((res) => {
          setFunctionsData((prev) => {
            // Merge function
            const rows = prev.rows.map((row) => {
              const usage = res.data.functions.find((usageItem) => {
                return usageItem.slug === row.slug;
              });

              return {
                ...row,
                ...usage,
              };
            });

            return {
              ...prev,
              rows: rows,
            };
          });
        });
      } catch (err) {
        console.error(`failed to fetch function usage data: ${err}`);
      }
    }
  }, [archived, client, env.id, functionsRes, functionsData.lastPage]);

  return {
    error: functionsRes.error,
    hasMore: functionsRes.data?.page.hasNextPage,
    isLoading: functionsRes.isLoading,
    loadMore: () => setPage((prev) => prev + 1),
    rows: functionsData.rows,
  };
}
