'use client';

import { makeValueTypeLabel, repeatArrayBrackets } from '../../../typeUtil';
import type { ArrayNode, ObjectNode, SchemaNode, TupleNode, ValueNode } from '../../../types';
import { ObjectRow } from '../../ObjectRow';
import { Row } from '../../Row';
import { TupleRow } from '../../TupleRow';
import { ValueRow } from '../../ValueRow';

export function ArrayRowVariantNested({ node }: { node: ArrayNode }): React.ReactElement {
  const info = computeNestedTerminal(node.element);

  switch (info.terminal) {
    case 'object': {
      const object = info.object;
      if (!object) return <Row node={node.element} />;

      return (
        <ObjectRow
          node={mkObjectNode(node, object)}
          typeLabelOverride={repeatArrayBrackets(info.bracketLayers)}
        />
      );
    }
    case 'tuple': {
      const tuple = info.tuple;
      if (!tuple) return <Row node={node.element} />;

      return (
        <TupleRow
          node={mkTupleNode(node, tuple)}
          typeLabelOverride={repeatArrayBrackets(info.bracketLayers)}
        />
      );
    }
    case 'value': {
      const value = info.value;
      if (!value) return <Row node={node.element} />;

      return (
        <ValueRow
          node={mkValueNode(node)}
          typeLabelOverride={labelForNestedValue(info.bracketLayers, value)}
        />
      );
    }
    default:
      return <Row node={node.element} />;
  }
}

function computeNestedTerminal(element: SchemaNode): {
  bracketLayers: number;
  terminal: 'value' | 'object' | 'tuple' | null;
  value?: ValueNode;
  object?: ObjectNode;
  tuple?: TupleNode;
} {
  let bracketLayers = 1;
  let cursor: SchemaNode | undefined = element;

  while (cursor && cursor.kind === 'array') {
    cursor = cursor.element;
    bracketLayers += 1;
    if (cursor.kind === 'value') return { bracketLayers, terminal: 'value', value: cursor };
    if (cursor.kind === 'object') return { bracketLayers, terminal: 'object', object: cursor };
    if (cursor.kind === 'tuple') return { bracketLayers, terminal: 'tuple', tuple: cursor };
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
