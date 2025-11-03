'use client';

import type { ArrayNode } from '../../types';
import { ArrayRowVariantNested } from './variants/ArrayRowVariantNested';
import { ArrayRowVariantObject } from './variants/ArrayRowVariantObject';
import { ArrayRowVariantTuple } from './variants/ArrayRowVariantTuple';
import { ArrayRowVariantValue } from './variants/ArrayRowVariantValue';

export type ArrayRowProps = { node: ArrayNode };

export function ArrayRow({ node }: ArrayRowProps): React.ReactElement | null {
  const element = node.element;

  if (element.kind === 'array') return <ArrayRowVariantNested node={node} />;
  if (element.kind === 'object') return <ArrayRowVariantObject node={node} />;
  if (element.kind === 'tuple') return <ArrayRowVariantTuple node={node} />;
  if (element.kind === 'value') return <ArrayRowVariantValue node={node} />;

  return null;
}
