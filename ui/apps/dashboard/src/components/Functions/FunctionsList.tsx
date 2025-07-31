'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Search } from '@inngest/components/Forms/Search';
import useDebounce from '@inngest/components/hooks/useDebounce';

import { StatusMenu } from '@/components/Functions/StatusMenu';
import { useBooleanFlag } from '../FeatureFlags/hooks';
import { FunctionTable } from './FunctionTable';
import { useRows } from './useRows';

type FunctionListProps = {
  envSlug: string;
  archived?: boolean;
};

export const FunctionList = ({ envSlug, archived }: FunctionListProps) => {
  const { value: isSearchEnabled } = useBooleanFlag('function-list-search');
  const [searchInput, setSearchInput] = useState<string>('');
  const [searchParam, setSearchParam] = useState<string>('');
  const debouncedSearch = useDebounce(() => {
    setSearchParam(searchInput);
  }, 400);
  const { error, isLoading, hasMore, loadMore, rows, isFirstLoad } = useRows({
    archived: !!archived,
    search: searchParam,
  });
  if (error) {
    throw error;
  }

  return (
    <div className="bg-canvasBase flex min-h-0 flex-1 flex-col">
      <div className="mx-3 flex h-11 flex-row items-center justify-start gap-1.5">
        {isSearchEnabled && (
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
        )}
        <StatusMenu archived={!!archived} envSlug={envSlug} />
      </div>
      <FunctionTable rows={rows} isLoading={isFirstLoad} />

      {hasMore !== false && !isFirstLoad && rows.length > 0 && (
        <div className="flex w-full justify-center py-2.5">
          <Button
            loading={isLoading}
            appearance="outlined"
            kind="secondary"
            onClick={loadMore}
            label={isLoading ? 'Loading' : 'Load More'}
          />
        </div>
      )}
    </div>
  );
};
