import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import EntityFilter from '@inngest/components/Filter/EntityFilter';
import { Search } from '@inngest/components/Forms/Search';
import TableBlankState from '@inngest/components/Functions/TableBlankState';
import { InfiniteScrollTrigger } from '@inngest/components/InfiniteScrollTrigger/InfiniteScrollTrigger';
import { type Option } from '@inngest/components/Select/Select';
import { Table } from '@inngest/components/Table';
import useDebounce from '@inngest/components/hooks/useDebounce';
import { type Function, type PageInfo } from '@inngest/components/types/function';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useNavigate, type LinkComponentProps } from '@tanstack/react-router';

import { useSearchParam, useStringArraySearchParam } from '../hooks/useSearchParams';
import FunctionsStatusFilter from './StatusMenu';
import { useColumns } from './columns';

export function FunctionsTable({
  getFunctions,
  getFunctionVolume,
  pathCreator,
  emptyActions,
  apps,
}: {
  emptyActions: React.ReactNode;
  pathCreator: {
    function: (params: { functionSlug: string }) => LinkComponentProps['to'];
    eventType: (params: { eventName: string }) => LinkComponentProps['to'];
    app: (params: { externalAppID: string }) => LinkComponentProps['to'];
  };
  getFunctions: ({
    cursor,
    archived,
    nameSearch,
    appIDs,
  }: {
    cursor: number | null;
    nameSearch: string | null;
    appIDs: string[] | null;
    archived: boolean;
  }) => Promise<{ functions: Omit<Function, 'usage'>[]; pageInfo: PageInfo }>;
  getFunctionVolume: ({
    functionID,
  }: {
    functionID: string;
  }) => Promise<Pick<Function, 'usage' | 'failureRate'>>;
  apps?: Option[];
}) {
  const navigate = useNavigate();
  const columns = useColumns({ pathCreator, getFunctionVolume });

  const [filteredStatus, setFilteredStatus, removeFilteredStatus] = useSearchParam('archived');
  const archived = filteredStatus === 'true';
  const [isScrollable, setIsScrollable] = useState(false);
  const [nameSearch = null, setNameSearch, removeNameSearch] = useSearchParam('nameSearch');
  const [searchInput, setSearchInput] = useState<string>(nameSearch || '');
  const [filteredApp = [], setFilteredApp, removeFilteredApp] =
    useStringArraySearchParam('filterApp');
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

  const onAppFilterChange = useCallback(
    (value: string[]) => {
      if (value.length > 0) {
        setFilteredApp(value);
      } else {
        removeFilteredApp();
      }
      scrollToTop();
    },
    [setFilteredApp, removeFilteredApp]
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
    queryKey: ['functions', { archived, nameSearch, appIDs: filteredApp }],
    queryFn: ({ pageParam = 1 }: { pageParam: number }) =>
      getFunctions({
        cursor: pageParam,
        archived,
        nameSearch,
        appIDs: filteredApp.length > 0 ? filteredApp : null,
      }),
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

  if (error) {
    return <ErrorCard error={error} reset={() => refetch()} />;
  }

  return (
    <div className="bg-canvasBase text-basis no-scrollbar flex flex-1 flex-col overflow-hidden focus-visible:outline-none">
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
        {apps && (
          <EntityFilter
            type="app"
            onFilterChange={onAppFilterChange}
            selectedEntities={filteredApp}
            entities={apps}
          />
        )}
      </div>
      <div className="flex-1 overflow-y-auto" ref={containerRef}>
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
            navigate({ to: pathCreator.function({ functionSlug: row.original.slug }) })
          }
          getRowHref={(row) => pathCreator.function({ functionSlug: row.original.slug })}
        />
        <InfiniteScrollTrigger
          onIntersect={fetchNextPage}
          hasMore={hasNextPage ?? false}
          isLoading={isFetching || isFetchingNextPage}
          root={containerRef.current}
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
