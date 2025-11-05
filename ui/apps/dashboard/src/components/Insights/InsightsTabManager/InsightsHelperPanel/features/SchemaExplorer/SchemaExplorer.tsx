'use client';

import { useCallback, useRef, useState } from 'react';
import { Search } from '@inngest/components/Forms/Search';
import { Pill } from '@inngest/components/Pill/Pill';
import { SchemaViewer } from '@inngest/components/SchemaViewer/SchemaViewer';
import type { ValueNode } from '@inngest/components/SchemaViewer/types';

import { SHOW_SCHEMA_SEARCH } from '@/components/Insights/temp-flags';
import { useSchemas } from './SchemasContext/SchemasContext';

export function SchemaExplorer() {
  const [search, setSearch] = useState('');
  const containerRef = useRef<HTMLDivElement>(null);
  const { entries } = useSchemas({ search });

  const renderSharedAdornment = useCallback((node: ValueNode) => {
    if (node.path !== 'event') return null;
    return (
      <Pill appearance="outlined" className="border-subtle text-subtle" kind="secondary">
        Shared schema
      </Pill>
    );
  }, []);

  const renderEntry = useCallback(
    (entry: (typeof entries)[number], idx: number) => (
      <SchemaViewer
        key={entry.key || `${entry.displayName}:${idx}`}
        computeType={entry.key === 'common:event' ? computeSharedEventSchemaType : undefined}
        defaultExpandedPaths={entry.key === 'common:event' ? ['event'] : undefined}
        node={entry.node}
        renderAdornment={entry.key === 'common:event' ? renderSharedAdornment : undefined}
      />
    ),
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
      <>
        {entries.map(renderEntry)}
        {/* TODO: Handle infinite scroll and loading, error states */}
        {/* TODO: Add infinite scroll trigger */}
      </>
    </div>
  );
}

function computeSharedEventSchemaType(node: ValueNode, baseLabel: string): string {
  if (node.path === 'event.data' && baseLabel === 'string') return 'JSON';
  return baseLabel;
}
