import { useMemo } from 'react';

import { Pagination, type PaginationProps } from './Pagination';
import { usePagination, type UsePaginationConfig, type UsePaginationOutput } from './usePagination';

type PaginationPropsPassThrough = Omit<
  PaginationProps,
  'currentPage' | 'setCurrentPage' | 'totalPages'
>;
type UsePaginationUIConfig<T> = UsePaginationConfig<T>;
type UsePaginationUIOutput<T> = UsePaginationOutput<T> & {
  BoundPagination: React.ComponentType<PaginationPropsPassThrough>;
};

export function usePaginationUI<T>({
  data,
  id,
  pageSize = 10,
}: UsePaginationUIConfig<T>): UsePaginationUIOutput<T> {
  const { currentPage, currentPageData, setCurrentPage, totalPages } = usePagination({
    data,
    id,
    pageSize,
  });

  const BoundPagination = useMemo(() => {
    return function PaginationEnhanced(props: PaginationPropsPassThrough) {
      return (
        <Pagination
          currentPage={currentPage}
          setCurrentPage={setCurrentPage}
          totalPages={totalPages}
          {...props}
        />
      );
    };
  }, [currentPage, setCurrentPage, totalPages]);

  return useMemo(
    () => ({
      BoundPagination,
      currentPage,
      currentPageData,
      setCurrentPage,
      totalPages,
    }),
    [BoundPagination, currentPage, currentPageData, setCurrentPage, totalPages]
  );
}
