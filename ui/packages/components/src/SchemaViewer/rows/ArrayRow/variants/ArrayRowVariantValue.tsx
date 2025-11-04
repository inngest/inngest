'use client';

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
    const parts = element.type.map(capitalize).sort((a, b) => a.localeCompare(b));
    return parts.length > 1 ? `[](${parts.join(' | ')})` : `[]${parts[0]}`;
  }
  return `[]${capitalize(element.type)}`;
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}
