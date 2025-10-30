'use client';

import { ExpansionProvider } from './ExpansionContext';
import { Row } from './rows/Row';
import type { SchemaNode } from './types';

export type SchemaViewerProps = { node: SchemaNode };

export function SchemaViewer({ node }: SchemaViewerProps): React.ReactElement {
  return (
    <ExpansionProvider>
      <div>
        <Row node={node} />
      </div>
    </ExpansionProvider>
  );
}
