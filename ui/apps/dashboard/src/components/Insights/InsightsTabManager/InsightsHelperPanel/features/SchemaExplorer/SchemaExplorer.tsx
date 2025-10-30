'use client';

import * as React from 'react';

import { useSchemas } from './useSchemas';

export function SchemaExplorer() {
  const { schemas } = useSchemas();

  return (
    <div>
      <h1>Schema Explorer</h1>
      <pre>{JSON.stringify(schemas, null, 2)}</pre>
    </div>
  );
}
