'use client';

import { SchemaViewer } from '@inngest/components/SchemaViewer/SchemaViewer';
import type { ValueNode } from '@inngest/components/SchemaViewer/types';

import { useSchemas } from './useSchemas';

export function SchemaExplorer() {
  const { schemas } = useSchemas();

  return (
    <div className="h-full w-full overflow-auto p-4">
      {schemas.map((schema) => (
        <SchemaViewer
          key={schema.name}
          computeType={computeType}
          defaultExpandedPaths={['event']}
          node={schema}
        />
      ))}
    </div>
  );
}

const computeType = (node: ValueNode, baseLabel: string): string => {
  if (node.path === 'event.data' && baseLabel === 'string') return 'JSON';
  return baseLabel;
};
