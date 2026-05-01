import { useCallback, useRef, useState } from 'react';
import { Search } from '@inngest/components/Forms/Search';
import { InfiniteScrollTrigger } from '@inngest/components/InfiniteScrollTrigger/InfiniteScrollTrigger';
import { SchemaViewer } from '@inngest/components/SchemaViewer/SchemaViewer';

import { SchemaExplorerSwitcher } from './SchemaExplorerSwitcher';
import { useSchemas } from './SchemasContext/SchemasContext';
import { useSchemasInUse } from './useSchemasInUse';
import { tableEntries } from './tableSchemas';

export function SchemaExplorer() {
  const [search, setSearch] = useState('');
  const containerRef = useRef<HTMLDivElement>(null);
  const {
    entries,
    error,
    hasFetchedMax,
    hasNextPage,
    fetchNextPage,
    isLoading,
    isFetchingNextPage,
  } = useSchemas({
    search,
  });

  const { schemasInUse } = useSchemasInUse();

  const renderEntry = useCallback((entry: (typeof entries)[number]) => {
    return <SchemaViewer key={entry.key} node={entry.node} />;
  }, []);

  return (
    <div
      className="flex h-full w-full flex-col gap-3 overflow-auto p-4"
      ref={containerRef}
    >
      <>
        <div className="text-light text-xs font-medium uppercase">
          Available Tables
        </div>
        <div className="flex flex-col gap-1">
          {tableEntries.map((tableEntry) => (
            <SchemaViewer key={tableEntry.key} node={tableEntry.node} />
          ))}
        </div>
        {schemasInUse.length > 0 && (
          <div className="mb-3 flex flex-col gap-2">
            <div className="text-light text-xs font-medium uppercase">
              Schemas in Use
            </div>
            <div className="flex flex-col gap-1">
              {schemasInUse.map((schema) => renderEntry(schema))}
            </div>
          </div>
        )}
        <div className="text-light text-xs font-medium uppercase">
          Event Schemas
        </div>
        <Search
          inngestSize="base"
          onUpdate={setSearch}
          placeholder="Search event type"
          value={search}
        />
      </>
      <div className="flex flex-col gap-1">
        <SchemaExplorerSwitcher
          entries={entries}
          error={error}
          isLoading={isLoading}
          isFetchingNextPage={isFetchingNextPage}
          hasFetchedMax={hasFetchedMax}
          hasNextPage={hasNextPage}
          fetchNextPage={fetchNextPage}
          renderEntry={renderEntry}
        />
        <InfiniteScrollTrigger
          onIntersect={fetchNextPage}
          hasMore={hasNextPage && !error && !hasFetchedMax}
          isLoading={isLoading || isFetchingNextPage}
          root={containerRef.current}
          rootMargin="50px"
        />
      </div>
    </div>
  );
}
