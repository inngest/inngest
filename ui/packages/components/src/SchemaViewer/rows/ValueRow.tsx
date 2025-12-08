import { Pill } from '@inngest/components/Pill/Pill';
import { RiFileCopyLine } from '@remixicon/react';
import { toast } from 'sonner';

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

  const handleCopyValue = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(node.name);
      toast.success('Value copied to clipboard');
    } catch (err) {
      console.error('Failed to copy:', err);
      toast.error('Failed to copy to clipboard');
    }
  };

  return (
    <div className="group flex items-baseline justify-between gap-2 rounded px-1 py-0.5">
      <div className="flex items-center gap-1">
        <span
          className={cn(
            'text-sm',
            boldName ? 'text-basis font-semibold' : 'text-subtle',
            'whitespace-nowrap'
          )}
        >
          {node.name}
        </span>
        <button
          className="hover:bg-canvasBase flex items-center rounded p-0.5 opacity-0 transition-opacity group-hover:opacity-100"
          onClick={handleCopyValue}
          type="button"
          aria-label="Copy value"
        >
          <RiFileCopyLine className="text-subtle h-3 w-3" />
        </button>
      </div>
      <div className="flex items-baseline gap-1.5">
        {Boolean(typeText) && (
          <span className="text-muted whitespace-nowrap font-mono text-xs capitalize">
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

function getTypeLabel(node: ValueNode, override?: string): string {
  return override ?? (Array.isArray(node.type) ? node.type.join(' | ') : node.type);
}
