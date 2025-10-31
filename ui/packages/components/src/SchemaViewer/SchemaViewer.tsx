'use client';

import { AdornmentProvider, type RenderAdornmentFn } from './AdornmentContext';
import { ExpansionProvider } from './ExpansionContext';
import { TypeProvider } from './TypeContext';
import { Row } from './rows/Row';
import type { SchemaNode } from './types';

export type SchemaViewerProps = {
  computeType?: (node: any, baseLabel: string) => string;
  defaultExpandedPaths?: string[];
  hide?: boolean;
  node: SchemaNode;
  renderAdornment?: RenderAdornmentFn;
};

export function SchemaViewer({
  computeType,
  defaultExpandedPaths,
  hide,
  node,
  renderAdornment,
}: SchemaViewerProps): React.ReactElement {
  return (
    <ExpansionProvider defaultExpandedPaths={defaultExpandedPaths}>
      <AdornmentProvider renderAdornment={renderAdornment}>
        <TypeProvider computeType={computeType}>{!hide && <Row node={node} />}</TypeProvider>
      </AdornmentProvider>
    </ExpansionProvider>
  );
}
