'use client';

import { Row } from './rows/Row';
import type { SchemaNode } from './types';

export type SchemaViewerProps = { node: SchemaNode };

export function SchemaViewer({ node }: SchemaViewerProps): React.ReactElement {
  return (
    <div>
      <Row node={node} />
    </div>
  );
}
