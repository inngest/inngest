'use client';

import { makeValueTypeLabel, repeatArrayBrackets } from '../../../typeUtil';
import type { ArrayNode, ObjectNode, SchemaNode, TupleNode, ValueNode } from '../../../types';
import { ObjectRow } from '../../ObjectRow';
import { TupleRow } from '../../TupleRow';
import { ValueRow } from '../../ValueRow';

// Renders an array of arrays (nested arrays), delegating to the terminal row type
export function ArrayRowVariantNested({ node }: { node: ArrayNode }): React.ReactElement | null {
  const info = computeNestedTerminal(node.element);

  switch (info.terminal?.kind) {
    case 'object': {
      const object = info.terminal;

      return (
        <ObjectRow
          node={mkObjectNode(node, object)}
          typeLabelOverride={repeatArrayBrackets(info.bracketLayers)}
        />
      );
    }
    case 'tuple': {
      const tuple = info.terminal;

      return (
        <TupleRow
          node={mkTupleNode(node, tuple)}
          typeLabelOverride={repeatArrayBrackets(info.bracketLayers)}
        />
      );
    }
    case 'value': {
      const value = info.terminal;

      return (
        <ValueRow
          node={mkValueNode(node)}
          typeLabelOverride={labelForNestedValue(info.bracketLayers, value)}
        />
      );
    }
    default:
      return null;
  }
}

// Identifies the first non-array schema node.
function computeNestedTerminal(element: SchemaNode): {
  bracketLayers: number;
  terminal: ObjectNode | TupleNode | ValueNode | null;
} {
  let bracketLayers = 1;
  let cursor: SchemaNode | undefined = element;

  while (cursor && cursor.kind === 'array') {
    cursor = cursor.element;
    bracketLayers += 1;
    if (cursor.kind !== 'array') {
      return { bracketLayers, terminal: cursor };
    }
  }

  return { bracketLayers, terminal: null };
}

function labelForNestedValue(bracketLayers: number, value: ValueNode): string {
  const base = makeValueTypeLabel(value);
  return `${repeatArrayBrackets(bracketLayers)}${base}`;
}

function mkObjectNode(node: ArrayNode, terminal: ObjectNode): ObjectNode {
  return { kind: 'object', name: node.name, path: node.path, children: terminal.children };
}

function mkTupleNode(node: ArrayNode, terminal: TupleNode): TupleNode {
  return { kind: 'tuple', name: node.name, path: node.path, elements: terminal.elements };
}

function mkValueNode(node: ArrayNode): ValueNode {
  return { kind: 'value', name: node.name, path: node.path, type: 'array' };
}
