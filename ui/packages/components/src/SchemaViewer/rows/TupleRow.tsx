'use client';

import { useExpansion } from '../ExpansionContext';
import type { TupleNode, ValueNode } from '../types';
import { CollapsibleRowWidget } from './CollapsibleRowWidget';
import { Row } from './Row';
import { ValueRow } from './ValueRow';

export type TupleRowProps = { node: TupleNode };

export function TupleRow({ node }: TupleRowProps): React.ReactElement {
  const { isExpanded, toggle } = useExpansion();
  const open = isExpanded(node.path);

  return (
    <div className="flex flex-col gap-1">
      <div className="flex cursor-pointer items-center" onClick={() => toggle(node.path)}>
        <CollapsibleRowWidget open={open} />
        <div className="-ml-0.5">
          <ValueRow boldName={open} node={makeFauxValueNode(node)} typeLabelOverride={'[]'} />
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
