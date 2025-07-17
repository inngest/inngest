'use client';

import { useCallback, useEffect, useMemo, useState, type SetStateAction } from 'react';

export type UsePaginationConfig<T> = {
  data: Array<T>;
  id: string;
  pageSize?: number;
};

export type UsePaginationOutput<T> = {
  currentPage: number;
  currentPageData: T[];
  setCurrentPage: (action: SetStateAction<number>) => void;
  totalPages: number;
};

export function usePagination<T>({
  data,
  id,
  pageSize = 10,
}: UsePaginationConfig<T>): UsePaginationOutput<T> {
  const [currentPage, setCurrentPage] = useState(1);

  const totalPages = Math.ceil(data.length / pageSize);

  const setCurrentPageSafe = useCallback(
    (action: SetStateAction<number>) => {
      setCurrentPage((prev: number) => {
        const newPage = typeof action === 'function' ? action(prev) : action;

        if (newPage < 1) {
          console.warn(`usePagination: newPage is less than 1; setting to 1.`);
          return 1;
        }

        if (newPage > totalPages) {
          console.warn(
            `usePagination: newPage is greater than totalPages; setting to ${totalPages}.`
          );
          return totalPages;
        }

        return newPage;
      });
    },
    [totalPages]
  );

  const currentPageData = useMemo(() => {
    return data.slice((currentPage - 1) * pageSize, currentPage * pageSize);
  }, [data, currentPage, pageSize]);

  useEffect(() => {
    if (currentPage !== 1 && (currentPage > totalPages || currentPage < 1)) {
      setCurrentPage(1);
    }
  }, [currentPage, totalPages]);

  // Reset the current page when the id (e.g. searchParam) changes
  useEffect(() => {
    if (currentPage !== 1) setCurrentPage(1);
  }, [id]);

  return useMemo(
    () => ({
      currentPage,
      currentPageData,
      setCurrentPage: setCurrentPageSafe,
      totalPages,
    }),
    [currentPage, currentPageData, totalPages, setCurrentPageSafe]
  );
}
