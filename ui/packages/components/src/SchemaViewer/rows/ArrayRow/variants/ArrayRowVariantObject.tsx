import type { ArrayNode } from '../../../types';
import { ObjectRow } from '../../ObjectRow';

// Renders an array of objects
export function ArrayRowVariantObject({ node }: { node: ArrayNode }) {
  const { element } = node;
  if (element.kind !== 'object') return null;

  return (
    <ObjectRow
      node={{ kind: 'object', children: element.children, name: node.name, path: node.path }}
      typeLabelOverride="[]"
    />
  );
}
