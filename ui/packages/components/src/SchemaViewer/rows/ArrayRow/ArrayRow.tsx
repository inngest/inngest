'use client';

import type { ArrayNode } from '../../types';
import { ArrayRowVariantNested } from './variants/ArrayRowVariantNested';
import { ArrayRowVariantObject } from './variants/ArrayRowVariantObject';
import { ArrayRowVariantTuple } from './variants/ArrayRowVariantTuple';
import { ArrayRowVariantValue } from './variants/ArrayRowVariantValue';

export type ArrayRowProps = { node: ArrayNode };

export function ArrayRow({ node }: ArrayRowProps) {
  const element = node.element;

  switch (element.kind) {
    case 'array':
      return <ArrayRowVariantNested node={node} />; // Array of arrays
    case 'object':
      return <ArrayRowVariantObject node={node} />; // Array of objects
    case 'tuple':
      return <ArrayRowVariantTuple node={node} />; // Array of tuples
    case 'value':
      return <ArrayRowVariantValue node={node} />; // Array of values
    default:
      return null;
  }
}
