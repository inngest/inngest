'use client';

import { useExpansion } from '../../../ExpansionContext';
import type { ArrayNode } from '../../../types';
import { CollapsibleRowWidget } from '../../CollapsibleRowWidget';
import { Row } from '../../Row';
import { ValueRow } from '../../ValueRow';

export function ArrayRowVariantTuple({ node }: { node: ArrayNode }): React.ReactElement {
  const { isExpanded, toggle } = useExpansion();
  const open = isExpanded(node.path);
  const { element } = node;

  return (
    <div className="flex flex-col gap-1">
      <div className="flex cursor-pointer items-center" onClick={() => toggle(node.path)}>
        <CollapsibleRowWidget open={open} />
        <div className="-ml-0.5">
          <ValueRow
            boldName={open}
            node={{ kind: 'value', name: node.name, path: node.path, type: 'array' }}
            typeLabelOverride={'[]'}
          />
        </div>
      </div>
      {open && element.kind === 'tuple' && (
        <div className="border-subtle ml-2 border-l pl-3">
          <div className="flex flex-col gap-1">
            {element.elements.map((child) => (
              <Row key={child.path} node={child} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
