'use client';

import * as React from 'react';

import type { ObjectNode, SchemaNode, ValueNode } from '../types';

export type ObjectRowProps = { node: ObjectNode };

export function ObjectRow({ node }: ObjectRowProps): React.ReactElement {
  return (
    <div>
      {} {node.name} object
      <div style={{ paddingLeft: 12 }}>
        {node.children.map((child) => (
          <React.Fragment key={child.path}>{renderChild(child)}</React.Fragment>
        ))}
      </div>
    </div>
  );
}

function renderChild(child: SchemaNode): React.ReactElement | null {
  if (child.kind === 'value') {
    const valueNode = child as ValueNode;
    const typeLabel = Array.isArray(valueNode.type) ? valueNode.type.join(' | ') : valueNode.type;
    return (
      <div>
        {valueNode.name} {typeLabel}
      </div>
    );
  }
  if (child.kind === 'array') {
    // Ignoring arrays for now
    return null;
  }
  // object
  return (
    <div>
      {} {child.name} object
    </div>
  );
}
