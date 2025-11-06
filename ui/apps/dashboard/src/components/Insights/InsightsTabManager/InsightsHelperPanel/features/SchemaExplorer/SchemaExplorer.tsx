'use client';

import { useCallback, useRef, useState } from 'react';
import { Search } from '@inngest/components/Forms/Search';
import { InfiniteScrollTrigger } from '@inngest/components/InfiniteScrollTrigger/InfiniteScrollTrigger';
import { Pill } from '@inngest/components/Pill/Pill';
import { SchemaViewer } from '@inngest/components/SchemaViewer/SchemaViewer';
import type { ValueNode } from '@inngest/components/SchemaViewer/types';

import { SHOW_SCHEMA_SEARCH } from '@/components/Insights/temp-flags';
import { SchemaExplorerSwitcher } from './SchemaExplorerSwitcher';
import { useSchemas } from './SchemasContext/SchemasContext';

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

  const renderSharedAdornment = useCallback((node: ValueNode) => {
    if (node.path !== 'event') return null;
    return (
      <Pill appearance="outlined" className="border-subtle text-subtle" kind="secondary">
        Shared schema
      </Pill>
    );
  }, []);

  const renderEntry = useCallback(
    (entry: (typeof entries)[number]) => {
      const isCommonEventSchema = entry.key === 'common:event';

      return (
        <SchemaViewer
          key={entry.key}
          computeType={isCommonEventSchema ? computeSharedEventSchemaType : undefined}
          defaultExpandedPaths={isCommonEventSchema ? ['event'] : undefined}
          node={entry.node}
          renderAdornment={isCommonEventSchema ? renderSharedAdornment : undefined}
        />
      );
    },
    [renderSharedAdornment]
  );

  return (
    <div className="flex h-full w-full flex-col gap-3 overflow-auto p-4" ref={containerRef}>
      {SHOW_SCHEMA_SEARCH && (
        <>
          <div className="text-light text-xs font-medium uppercase">All Schemas</div>
          <Search
            inngestSize="base"
            onUpdate={setSearch}
            placeholder="Search event type"
            value={search}
          />
        </>
      )}
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

function computeSharedEventSchemaType(node: ValueNode, baseLabel: string): string {
  if (node.path === 'event.data' && baseLabel === 'string') return 'JSON';
  return baseLabel;
}
