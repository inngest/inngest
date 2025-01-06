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
  const { error, isLoading, hasMore, loadMore, rows } = useRows({
    archived: !!archived,
    search: searchParam,
  });
  if (error) {
    throw error;
  }

  return (
    <div className="bg-canvasBase divide-subtle flex min-h-0 flex-1 flex-col divide-y">
      <div className="mx-4 my-1 flex h-10 flex-row items-center justify-start">
        <StatusMenu archived={!!archived} envSlug={envSlug} />
        {isSearchEnabled && (
          <Search
            name="search"
            placeholder="Search by name"
            value={searchInput}
            // Match the height of StatusMenu for now
            className="h-[30px] w-48 py-3"
            onUpdate={(value) => {
              setSearchInput(value);
              debouncedSearch();
            }}
          />
        )}
      </div>

      <FunctionTable rows={rows} />

      {hasMore !== false && (
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
