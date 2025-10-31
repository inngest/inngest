'use client';

import { useCallback, useState } from 'react';
import { Search } from '@inngest/components/Forms/Search';
import { Pill } from '@inngest/components/Pill/Pill';
import { SchemaViewer } from '@inngest/components/SchemaViewer/SchemaViewer';
import type { ValueNode } from '@inngest/components/SchemaViewer/types';

import { SHOW_SCHEMA_SEARCH } from '@/components/Insights/temp-flags';
import { useSchemas } from './useSchemas';

export function SchemaExplorer() {
  const { schemas } = useSchemas();
  const [search, setSearch] = useState('');

  // TODO: Make more resilient, an event type could be named "event"
  const renderAdornment = useCallback((node: ValueNode) => {
    if (node.path === 'event') {
      return (
        <Pill appearance="outlined" className="border-subtle text-subtle" kind="secondary">
          Shared schema
        </Pill>
      );
    }

    return null;
  }, []);

  return (
    <div className="flex h-full w-full flex-col gap-3 overflow-auto p-4">
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

      <div>
        {schemas.map((schema) => (
          <SchemaViewer
            key={schema.name}
            computeType={computeType}
            defaultExpandedPaths={['event']}
            hide={Boolean(search) && !schema.name.startsWith(search)}
            node={schema}
            renderAdornment={renderAdornment}
          />
        ))}
      </div>
    </div>
  );
}

// TODO: Make more resilient, an event type could be named "event"
const computeType = (node: ValueNode, baseLabel: string): string => {
  if (node.path === 'event.data' && baseLabel === 'string') return 'JSON';
  return baseLabel;
};
