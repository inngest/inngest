'use client';

import { useExpansion } from '../../../ExpansionContext';
import type { ArrayNode } from '../../../types';
import { CollapsibleRowWidget } from '../../CollapsibleRowWidget';
import { Row } from '../../Row';
import { ValueRow } from '../../ValueRow';

export function ArrayRowVariantObject({ node }: { node: ArrayNode }): React.ReactElement | null {
  const { isExpanded, toggle } = useExpansion();

  const { element } = node;
  if (element.kind !== 'object') return null;

  const open = isExpanded(node.path);

  const hasChildren = element.children.length > 0;
  const keysCount = element.children.length;
  const keysLabel = keysCount === 1 ? '1 key' : `${keysCount} keys`;

  return (
    <div className="flex flex-col gap-1">
      <div
        className={`flex items-center ${hasChildren ? 'cursor-pointer' : ''}`}
        onClick={hasChildren ? () => toggle(node.path) : undefined}
      >
        {hasChildren ? <CollapsibleRowWidget open={open} /> : <CollapsibleIconPlaceholder />}
        <div className="-ml-0.5">
          <ValueRow
            boldName={open}
            node={{ kind: 'value', name: node.name, path: node.path, type: 'array' }}
            typeLabelOverride={'[]'}
            typePillOverride={hasChildren ? keysLabel : 'object'}
          />
        </div>
      </div>
      {hasChildren && open && (
        <div className="border-subtle ml-2 border-l pl-3">
          <div className="flex flex-col gap-1">
            {element.children.map((child) => (
              <Row key={child.path} node={child} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function CollapsibleIconPlaceholder() {
  return <span className="text-muted -mb-0.5 inline-flex h-4 w-4 items-center justify-center" />;
}
