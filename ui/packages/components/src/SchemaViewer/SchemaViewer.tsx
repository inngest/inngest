'use client';

import { ExpansionProvider } from './ExpansionContext';
import { TypeProvider } from './TypeContext';
import { Row } from './rows/Row';
import type { SchemaNode } from './types';

export type SchemaViewerProps = {
  computeType?: (node: any, baseLabel: string) => string;
  node: SchemaNode;
};

export function SchemaViewer({ computeType, node }: SchemaViewerProps): React.ReactElement {
  return (
    <ExpansionProvider>
      <TypeProvider computeType={computeType}>
        <Row node={node} />
      </TypeProvider>
    </ExpansionProvider>
  );
}
