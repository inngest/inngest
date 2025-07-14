import { useMemo, useState, type Dispatch, type SetStateAction } from 'react';

export type UsePaginationConfig<T> = {
  data: Array<T>;
  pageSize?: number;
};

export type UsePaginationOutput<T> = {
  currentPage: number;
  currentPageData: T[];
  setCurrentPage: Dispatch<SetStateAction<number>>;
  totalPages: number;
};

export function usePagination<T>({
  data,
  pageSize = 10,
}: UsePaginationConfig<T>): UsePaginationOutput<T> {
  const [currentPage, setCurrentPage] = useState(1);

  const totalPages = Math.ceil(data.length / pageSize);

  const currentPageData = useMemo(() => {
    return data.slice((currentPage - 1) * pageSize, currentPage * pageSize);
  }, [data, currentPage, pageSize]);

  return useMemo(
    () => ({ totalPages, currentPage, setCurrentPage, currentPageData }),
    [currentPage, currentPageData, totalPages, setCurrentPage]
  );
}
