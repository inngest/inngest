import { Pill } from '@inngest/components/Pill/Pill';
import { STANDARD_EVENT_FIELDS } from '@inngest/components/constants';
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
      // The path is formatted as "eventName.field.subfield..."
      // We need to remove the event name prefix (which can contain dots)
      // We do this by finding the first occurrence of a standard field after the event name
      // Strategy: The path after the event name will always start with a dot followed by a field
      let relativePath = node.path;

      for (const field of STANDARD_EVENT_FIELDS) {
        const pattern = `.${field}`;
        const index = node.path.indexOf(pattern);
        if (index !== -1) {
          // Found a standard field, extract from this point (excluding the leading dot)
          relativePath = node.path.substring(index + 1);
          break;
        }
      }

      await navigator.clipboard.writeText(relativePath);
      toast.success('Path copied to clipboard');
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
