'use client';

import { cn } from '../../utils/classNames';
import type { ValueNode } from '../types';

export type ValueRowProps = { node: ValueNode; typeLabelOverride?: string; boldName?: boolean };

export function ValueRow({ node, typeLabelOverride, boldName }: ValueRowProps): React.ReactElement {
  const typeLabel = getTypeLabel(node, typeLabelOverride);

  return (
    <div className="flex items-center gap-1.5 px-1 py-0.5">
      <span className={cn('text-sm', boldName ? 'text-basis font-semibold' : 'text-subtle')}>
        {node.name}
      </span>
      <span className="text-muted font-mono text-xs capitalize">{typeLabel}</span>
    </div>
  );
}

function getTypeLabel(node: ValueNode, override?: string): string {
  return override ?? (Array.isArray(node.type) ? node.type.join(' | ') : node.type);
}
