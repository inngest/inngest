'use client';

import { useState } from 'react';

import type { ObjectNode, SchemaNode, ValueNode } from '../types';
import { CollapsibleRowWidget } from './CollapsibleRowWidget';
import { Row } from './Row';
import { ValueRow } from './ValueRow';

export type ObjectRowProps = { node: ObjectNode };

export function ObjectRow({ node }: ObjectRowProps): React.ReactElement {
  const [open, setOpen] = useState(false);

  return (
    <div className="flex flex-col gap-1">
      <div className="flex cursor-pointer items-center" onClick={() => setOpen((v) => !v)}>
        <CollapsibleRowWidget open={open} />
        <div className="-ml-0.5">
          <ValueRow boldName={open} node={makeFauxValueNode(node)} typeLabelOverride={'{}'} />
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
