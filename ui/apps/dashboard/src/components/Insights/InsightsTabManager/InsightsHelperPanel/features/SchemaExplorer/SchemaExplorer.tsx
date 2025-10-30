'use client';

import * as React from 'react';

import { useSchemas } from './useSchemas';

export function SchemaExplorer() {
  const { schemas } = useSchemas();

  return (
    <div className="h-full w-full overflow-auto p-4">
      <pre>{JSON.stringify(schemas, null, 2)}</pre>
    </div>
  );
}
