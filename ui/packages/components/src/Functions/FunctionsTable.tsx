'use client';

import { useCallback, useEffect, useMemo, useRef, useState, type UIEventHandler } from 'react';
import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button/Button';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import { Search } from '@inngest/components/Forms/Search';
import TableBlankState from '@inngest/components/Functions/TableBlankState';
import { Table } from '@inngest/components/Table';
import useDebounce from '@inngest/components/hooks/useDebounce';
import { type Function, type PageInfo } from '@inngest/components/types/function';
import { useInfiniteQuery } from '@tanstack/react-query';

import { useSearchParam } from '../hooks/useSearchParam';
import FunctionsStatusFilter from './StatusMenu';
import { useColumns } from './columns';

export function FunctionsTable({
  getFunctions,
  getFunctionVolume,
  pathCreator,
  emptyActions,
}: {
  emptyActions: React.ReactNode;
  pathCreator: {
    function: (params: { functionSlug: string }) => Route;
    eventType: (params: { eventName: string }) => Route;
    app: (params: { externalAppID: string }) => Route;
  };
  getFunctions: ({
    cursor,
    archived,
  }: {
    cursor: number | null;
    nameSearch: string | null;
    archived: boolean;
  }) => Promise<{ functions: Omit<Function, 'usage'>[]; pageInfo: PageInfo }>;
  getFunctionVolume: ({
    functionID,
  }: {
    functionID: string;
  }) => Promise<Pick<Function, 'usage' | 'failureRate'>>;
}) {
  const router = useRouter();
  const columns = useColumns({ pathCreator, getFunctionVolume });

  const [filteredStatus, setFilteredStatus, removeFilteredStatus] = useSearchParam('archived');
  const archived = filteredStatus === 'true';
  const [isScrollable, setIsScrollable] = useState(false);
  const [nameSearch = null, setNameSearch, removeNameSearch] = useSearchParam('nameSearch');
  const [searchInput, setSearchInput] = useState<string>(nameSearch || '');
  const containerRef = useRef<HTMLDivElement>(null);

  const scrollToTop = useCallback(
    (smooth = false) => {
      if (containerRef.current) {
        containerRef.current.scrollTo({
          top: 0,
          behavior: smooth ? 'smooth' : 'auto',
        });
      }
    },
    [containerRef.current]
  );

  const debouncedSearch = useDebounce(() => {
    if (searchInput === '') {
      removeNameSearch();
    } else {
      setNameSearch(searchInput);
    }
    scrollToTop();
  }, 300);

  const onStatusFilterChange = useCallback(
    (value: boolean) => {
      if (value) {
        setFilteredStatus('true');
      } else {
        removeFilteredStatus();
      }
      scrollToTop();
    },
    [setFilteredStatus, removeFilteredStatus]
  );

  const {
    isPending, // first load, no data
    error,
    fetchNextPage,
    hasNextPage,
    data: functionsData,
    isFetching,
    refetch,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['functions', { archived, nameSearch }],
    queryFn: ({ pageParam = 1 }: { pageParam: number }) =>
      getFunctions({ cursor: pageParam, archived, nameSearch }),
    refetchOnWindowFocus: false,
    getNextPageParam: (lastPage) => {
      const { currentPage, totalPages } = lastPage.pageInfo;
      if (typeof totalPages === 'number' && currentPage < totalPages) {
        return currentPage + 1;
      }
    },
    initialPageParam: 1,
  });

  const mergedData = useMemo(() => {
    return (
      functionsData?.pages.flatMap((page) =>
        page.functions.map((e) => ({
          ...e,
          usage: undefined,
        }))
      ) ?? []
    );
  }, [functionsData]);

  const hasFunctionsData = mergedData && mergedData.length > 0;

  useEffect(() => {
    const el = containerRef.current;
    if (el) {
      setIsScrollable(el.scrollHeight > el.clientHeight);
    }
  }, [mergedData]);

  const onScroll: UIEventHandler<HTMLDivElement> = useCallback(
    (event) => {
      if (hasFunctionsData && hasNextPage) {
        const { scrollHeight, scrollTop, clientHeight } = event.target as HTMLDivElement;

        // Check if scrolled to the bottom
        const reachedBottom = scrollHeight - scrollTop - clientHeight < 200;
        if (reachedBottom && !isFetching && !isFetchingNextPage) {
          fetchNextPage();
        }
      }
    },
    [fetchNextPage, hasNextPage, isFetchingNextPage, hasFunctionsData, isFetching]
  );

  if (error) {
    return <ErrorCard error={error} reset={() => refetch()} />;
  }

  return (
    <div className="bg-canvasBase text-basis no-scrollbar flex-1 overflow-hidden focus-visible:outline-none">
      <div className="bg-canvasBase sticky top-0 z-10 mx-3 flex h-11 items-center gap-1.5">
        <Search
          name="search"
          placeholder="Search by function name"
          value={searchInput}
          className="w-[182px]"
          onUpdate={(value) => {
            setSearchInput(value);
            debouncedSearch();
          }}
        />
        <FunctionsStatusFilter archived={archived} onStatusChange={onStatusFilterChange} />
      </div>
      <div className="h-[calc(100%-58px)] overflow-y-auto" onScroll={onScroll} ref={containerRef}>
        <Table
          columns={columns}
          data={mergedData || []}
          isLoading={isPending || (isFetching && !isFetchingNextPage)}
          blankState={
            <TableBlankState
              actions={emptyActions}
              title={
                nameSearch
                  ? `No results found for "${nameSearch}"`
                  : archived
                  ? 'No archived functions found'
                  : undefined
              }
            />
          }
          onRowClick={(row) =>
            router.push(pathCreator.function({ functionSlug: row.original.slug }))
          }
          getRowHref={(row) => pathCreator.function({ functionSlug: row.original.slug })}
        />
        {!hasNextPage && hasFunctionsData && isScrollable && !isFetchingNextPage && !isFetching && (
          <div className="flex flex-col items-center pb-4 pt-8">
            <p className="text-muted text-sm">No additional functions found.</p>
            <Button
              label="Back to top"
              kind="primary"
              appearance="ghost"
              onClick={() => scrollToTop(true)}
            />
          </div>
        )}
        {isFetchingNextPage && (
          <div className="flex flex-col items-center">
            <Button appearance="outlined" label="loading" loading={true} />
          </div>
        )}
      </div>
    </div>
  );
}
