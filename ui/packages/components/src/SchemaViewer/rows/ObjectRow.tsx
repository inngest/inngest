'use client';

import { useExpansion } from '../ExpansionContext';
import type { ObjectNode, SchemaNode, ValueNode } from '../types';
import { CollapsibleRowWidget } from './CollapsibleRowWidget';
import { Row } from './Row';
import { ValueRow } from './ValueRow';

export type ObjectRowProps = { node: ObjectNode; typeLabelOverride?: string };

export function ObjectRow({ node, typeLabelOverride }: ObjectRowProps): React.ReactElement {
  const { isExpanded, toggle } = useExpansion();

  const open = isExpanded(node.path);
  const hasChildren = node.children.length > 0;

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
          />
        </div>
      </div>
      {open && (
        <div className="border-subtle ml-2 border-l pl-3">
          <div className="flex flex-col gap-1">
            {node.children.map((child) => (
              <Row key={child.path} node={child} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function makeFauxValueNode(node: SchemaNode): ValueNode {
  return { kind: 'value', name: node.name, path: node.path, type: 'object' };
}
