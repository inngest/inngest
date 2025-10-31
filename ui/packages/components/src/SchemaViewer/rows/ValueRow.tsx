'use client';

import { cn } from '../../utils/classNames';
import { useRenderAdornment } from '../AdornmentContext';
import { useComputeType } from '../TypeContext';
import type { ValueNode } from '../types';

export type ValueRowProps = { boldName?: boolean; node: ValueNode; typeLabelOverride?: string };

export function ValueRow({ node, typeLabelOverride, boldName }: ValueRowProps): React.ReactElement {
  const computeType = useComputeType();
  const renderAdornment = useRenderAdornment();

  const baseLabel = getTypeLabel(node, typeLabelOverride);
  const typeLabel = computeType(node, baseLabel);

  return (
    <div className="flex select-none items-baseline gap-1.5 px-1 py-0.5">
      <span
        className={cn(
          'text-sm',
          boldName ? 'text-basis font-semibold' : 'text-subtle',
          'whitespace-nowrap'
        )}
      >
        {node.name}
      </span>
      <span className="text-quaternary-warmerxIntense font-mono text-xs capitalize">
        {computeType(node, baseLabel)}
      </span>
      <span className="self-baseline align-baseline text-xs leading-none">
        {renderAdornment(node, typeLabel)}
      </span>
    </div>
  );
}

function getTypeLabel(node: ValueNode, override?: string): string {
  return override ?? (Array.isArray(node.type) ? node.type.join(' | ') : node.type);
}
