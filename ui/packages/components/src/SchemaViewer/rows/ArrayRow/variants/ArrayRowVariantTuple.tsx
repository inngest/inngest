import type { ArrayNode } from '../../../types';
import { TupleRow } from '../../TupleRow';

// Renders an array of tuples
export function ArrayRowVariantTuple({ node }: { node: ArrayNode }): React.ReactElement | null {
  const { element } = node;
  if (element.kind !== 'tuple') return null;

  return (
    <TupleRow
      node={{ kind: 'tuple', name: node.name, path: node.path, elements: element.elements }}
      typeLabelOverride="[]"
    />
  );
}
