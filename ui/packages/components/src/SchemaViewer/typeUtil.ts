'use client';

import type { ValueNode } from './types';

export function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

export function repeatArrayBrackets(layers: number): string {
  return layers > 0 ? '[]'.repeat(layers) : '';
}

export function makeValueTypeLabel(value: ValueNode): string {
  if (Array.isArray(value.type)) {
    const parts = value.type.map(capitalize).sort((a, b) => a.localeCompare(b));
    return parts.join(' | ');
  }

  return capitalize(value.type);
}
