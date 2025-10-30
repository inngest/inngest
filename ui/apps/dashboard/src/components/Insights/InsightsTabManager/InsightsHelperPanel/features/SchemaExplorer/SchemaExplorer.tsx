'use client';

import { SchemaViewer } from '@inngest/components/SchemaViewer/SchemaViewer';

import { useSchemas } from './useSchemas';

export function SchemaExplorer() {
  const { schemas } = useSchemas();

  return schemas.map((schema) => <SchemaViewer key={schema.name} node={schema} />);
}
