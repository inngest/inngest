'use client';

import { ExpansionProvider } from './ExpansionContext';
import { TypeProvider } from './TypeContext';
import { Row } from './rows/Row';
import type { SchemaNode } from './types';

export type SchemaViewerProps = {
  computeType?: (node: any, baseLabel: string) => string;
  defaultExpandedPaths?: string[];
  hide?: boolean;
  node: SchemaNode;
};

export function SchemaViewer({
  computeType,
  defaultExpandedPaths,
  hide,
  node,
}: SchemaViewerProps): React.ReactElement {
  return (
    <ExpansionProvider defaultExpandedPaths={defaultExpandedPaths}>
      <TypeProvider computeType={computeType}>{!hide && <Row node={node} />}</TypeProvider>
    </ExpansionProvider>
  );
}
