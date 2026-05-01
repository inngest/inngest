import { RiFileCopyLine } from '@remixicon/react';

import { cn } from '../../utils/classNames';
import { useRenderAdornment } from '../AdornmentContext';
import { useExpansion } from '../ExpansionContext';
import { useComputeType } from '../TypeContext';
import type { ObjectNode } from '../types';
import { CollapsibleRowWidget } from './CollapsibleRowWidget';
import { Row } from './Row';
import { collectAllExpandablePaths, getTypeLabel, handleCopyValue } from './utils';

export type ObjectRowProps = { node: ObjectNode; typeLabelOverride?: string };

export function ObjectRow({ node }: ObjectRowProps): React.ReactElement {
  const computeType = useComputeType();
  const renderAdornment = useRenderAdornment();
  const { isExpanded, toggleRecursive } = useExpansion();

  const open = isExpanded(node.path);

  const handleToggle = () => {
    const allDescendantPaths = collectAllExpandablePaths(node);
    toggleRecursive(node.path, allDescendantPaths);
  };

  const baseLabel = getTypeLabel(node);
  const typeText = computeType(node, baseLabel);

  return (
    <div className="flex flex-col gap-1">
      <div
        className="hover:bg-canvasSubtle group flex cursor-pointer items-center justify-between gap-2 rounded px-1 py-0.5"
        onClick={handleToggle}
      >
        <CollapsibleRowWidget open={open} />
        <span className={'text-subtle overflow-hidden text-ellipsis whitespace-nowrap text-sm'}>
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
          <span className="text-muted min-w-0 whitespace-nowrap font-mono text-xs capitalize">
            {typeText}
          </span>
          <span className="self-baseline align-baseline text-xs leading-none">
            {renderAdornment(node, typeText)}
          </span>
        </div>
      </div>
      {open && (
        <div
          className={cn(
            'border-subtle pl-3',
            node.children.length === 0 ? 'ml-1.5' : 'ml-2 border-l'
          )}
        >
          <div className="flex flex-col gap-1">
            {node.children.map((child) => (
              <Row key={child.path} node={child} />
            ))}
            {node.children.length === 0 && (
              <div className="text-light text-sm">No data to show</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
