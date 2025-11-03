'use client';

import type { ArrayNode } from '../../types';
import { ArrayRowVariantNested } from './variants/ArrayRowVariantNested';
import { ArrayRowVariantObject } from './variants/ArrayRowVariantObject';
import { ArrayRowVariantTuple } from './variants/ArrayRowVariantTuple';
import { ArrayRowVariantValue } from './variants/ArrayRowVariantValue';

export type ArrayRowProps = { node: ArrayNode };

export function ArrayRow({ node }: ArrayRowProps): React.ReactElement | null {
  const element = node.element;

  switch (element.kind) {
    case 'array':
      return <ArrayRowVariantNested node={node} />;
    case 'object':
      return <ArrayRowVariantObject node={node} />;
    case 'tuple':
      return <ArrayRowVariantTuple node={node} />;
    case 'value':
      return <ArrayRowVariantValue node={node} />;
    default:
      return null;
  }
}
