'use client';

import { SchemaViewer } from '@inngest/components/SchemaViewer/SchemaViewer';

import { useSchemas } from './useSchemas';

export function SchemaExplorer() {
  const { schemas } = useSchemas();

  return (
    <div className="h-full w-full overflow-auto p-4">
      {schemas.map((schema) => (
        <SchemaViewer key={schema.name} node={schema} />
      ))}
    </div>
  );
}
