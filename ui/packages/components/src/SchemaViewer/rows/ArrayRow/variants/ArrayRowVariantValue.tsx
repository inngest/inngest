'use client';

import { valueTypeLabel } from '../../../typeUtil';
import type { ArrayNode, ValueNode } from '../../../types';
import { ValueRow } from '../../ValueRow';

export function ArrayRowVariantValue({ node }: { node: ArrayNode }): React.ReactElement | null {
  if (node.element.kind !== 'value') return null;

  return (
    <ValueRow
      node={{ kind: 'value', name: node.name, path: node.path, type: 'array' }}
      typeLabelOverride={buildArrayValueLabel(node.element)}
    />
  );
}

function buildArrayValueLabel(element: ValueNode): string {
  if (Array.isArray(element.type)) {
    const base = valueTypeLabel(element);
    return element.type.length > 1 ? `[](${base})` : `[]${base}`;
  }
  return `[]${valueTypeLabel(element)}`;
}
