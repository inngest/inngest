import type { ReactElement } from 'react';
import { Pill } from '@inngest/components/Pill/Pill';
import { RiFileCopyLine } from '@remixicon/react';

import { cn } from '../../utils/classNames';
import { useRenderAdornment } from '../AdornmentContext';
import { useComputeType } from '../TypeContext';
import type { ValueNode } from '../types';
import { getTypeLabel, handleCopyValue } from './utils';

export type ValueRowProps = {
  boldName?: boolean;
  node: ValueNode;
  typeLabelOverride?: string;
  typePillOverride?: string;
  btn?: ReactElement;
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
    <div className="group flex items-center justify-between gap-2 rounded px-1 py-0.5">
      <span
        className={cn(
          'overflow-hidden text-ellipsis',
          'text-sm',
          boldName ? 'text-basis font-semibold' : 'text-subtle',
          'whitespace-nowrap'
        )}
      >
        {node.name}
      </span>
      <button
        className={cn(
          'hover:bg-canvasBase min-w-sm rounded p-0.5 opacity-0 transition-opacity group-hover:opacity-100'
        )}
        onClick={handleCopyValue(node)}
        type="button"
        aria-label="Copy value"
      >
        <RiFileCopyLine className="text-subtle h-3 w-3" />
      </button>
      <div className="ml-auto flex items-baseline gap-1.5">
        {Boolean(typeText) && (
          <span className="text-muted min-w-0 whitespace-nowrap font-mono text-xs capitalize">
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
    </div>
  );
}
