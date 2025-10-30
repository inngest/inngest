'use client';

import type { ValueNode } from '../types';

export type ValueRowProps = { node: ValueNode };

export function ValueRow({ node }: ValueRowProps): React.ReactElement {
  const typeLabel = Array.isArray(node.type) ? node.type.join(' | ') : node.type;

  return (
    <div>
      {node.name} {typeLabel}
    </div>
  );
}
