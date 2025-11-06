'use client';

import { Pill } from '@inngest/components/Pill/Pill';

import { cn } from '../../utils/classNames';
import { useRenderAdornment } from '../AdornmentContext';
import { useComputeType } from '../TypeContext';
import type { ValueNode } from '../types';

export type ValueRowProps = {
  boldName?: boolean;
  node: ValueNode;
  typeLabelOverride?: string;
  typePillOverride?: string;
};

export function ValueRow({
  boldName,
  node,
  typeLabelOverride,
  typePillOverride,
}: ValueRowProps): React.ReactElement {
  const computeType = useComputeType();
  const renderAdornment = useRenderAdornment();

  const baseLabel = getTypeLabel(node, typeLabelOverride);
  const computed = computeType(node, baseLabel);
  const typeText = typeLabelOverride !== undefined ? typeLabelOverride : computed;

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
      {Boolean(typeText) && (
        <span className="text-quaternary-warmerxIntense whitespace-nowrap font-mono text-xs capitalize">
          {typeText}
        </span>
      )}
      {Boolean(typePillOverride) && (
        <Pill
          appearance="outlined"
          className="border-subtle text-subtle whitespace-nowrap"
          kind="secondary"
        >
          {typePillOverride}
        </Pill>
      )}
      <span className="self-baseline align-baseline text-xs leading-none">
        {renderAdornment(node, computed)}
      </span>
    </div>
  );
}

function getTypeLabel(node: ValueNode, override?: string): string {
  return override ?? (Array.isArray(node.type) ? node.type.join(' | ') : node.type);
}
