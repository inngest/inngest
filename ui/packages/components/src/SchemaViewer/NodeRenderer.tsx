'use client';

import { ObjectNode } from './ObjectNode';
import { ValueNode } from './ValueNode';
import type { SchemaNode as TreeNode } from './types';

export type NodeProps = {
  depth: number;
  expanded: Set<string>;
  node: TreeNode;
  onToggle: (path: string) => void;
};

// TODO: Support array rendering.
export function NodeRenderer({ depth, expanded, node, onToggle }: NodeProps) {
  switch (node.kind) {
    case 'object':
      return <ObjectNode depth={depth} expanded={expanded} node={node} onToggle={onToggle} />;
    case 'value':
      return <ValueNode depth={depth} node={node} />;
    default: {
      console.warn('SchemaViewer: skipping array rendering.', { path: node.path, name: node.name });
      return null;
    }
  }
}
