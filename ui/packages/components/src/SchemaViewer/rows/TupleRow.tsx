'use client';

import { useExpansion } from '../ExpansionContext';
import type { TupleNode, ValueNode } from '../types';
import { CollapsibleRowWidget } from './CollapsibleRowWidget';
import { Row } from './Row';
import { ValueRow } from './ValueRow';

export type TupleRowProps = { node: TupleNode; typeLabelOverride?: string };

export function TupleRow({ node, typeLabelOverride }: TupleRowProps): React.ReactElement {
  const { isExpanded, toggle } = useExpansion();
  const open = isExpanded(node.path);
  const hasChildren = node.elements.length > 0;

  return (
    <div className="flex flex-col gap-1">
      <div
        className={`flex items-center ${hasChildren ? 'cursor-pointer' : ''}`}
        onClick={hasChildren ? () => toggle(node.path) : undefined}
      >
        {hasChildren ? <CollapsibleRowWidget open={open} /> : null}
        <div className="-ml-0.5">
          <ValueRow
            boldName={open}
            node={makeFauxValueNode(node)}
            typeLabelOverride={typeLabelOverride ?? ''}
            typePillOverride={hasChildren ? `${node.elements.length} item tuple` : 'Tuple'}
          />
        </div>
      </div>
      {open && (
        <div className="border-subtle ml-2 border-l pl-3">
          <div className="flex flex-col gap-1">
            {node.elements.map((child) => (
              <Row key={child.path} node={child} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function makeFauxValueNode(node: TupleNode): ValueNode {
  return { kind: 'value', name: node.name, path: node.path, type: 'array' };
}
