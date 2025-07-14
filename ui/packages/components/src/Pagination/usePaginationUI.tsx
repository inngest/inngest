import { useMemo } from 'react';

import { Pagination, type PaginationProps } from './Pagination';
import { usePagination, type UsePaginationConfig, type UsePaginationOutput } from './usePagination';

type PaginationPropsPassThrough = Omit<
  PaginationProps,
  'currentPage' | 'setCurrentPage' | 'totalPages'
>;
type UsePaginationUIConfig<T> = UsePaginationConfig<T> & PaginationPropsPassThrough;
type UsePaginationUIOutput<T> = UsePaginationOutput<T> & {
  BoundPagination: React.ComponentType<PaginationPropsPassThrough>;
};

export function usePaginationUI<T>({
  data,
  pageSize = 10,
}: UsePaginationUIConfig<T>): UsePaginationUIOutput<T> {
  const { currentPage, currentPageData, setCurrentPage, totalPages } = usePagination({
    data,
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
  }, [currentPage, totalPages, setCurrentPage]);

  return {
    BoundPagination,
    currentPage,
    currentPageData,
    setCurrentPage,
    totalPages,
  };
}
