'use client';

import type { SchemaNode } from '../types';
import { ArrayRow } from './ArrayRow';
import { ObjectRow } from './ObjectRow';
import { TupleRow } from './TupleRow';
import { ValueRow } from './ValueRow';

export type RowProps = { node: SchemaNode };

export function Row({ node }: RowProps): React.ReactElement | null {
  switch (node.kind) {
    case 'array':
      return <ArrayRow node={node} />;
    case 'tuple':
      return <TupleRow node={node} />;
    case 'object':
      return <ObjectRow node={node} />;
    case 'value':
      return <ValueRow node={node} />;
    default:
      return null;
  }
}
