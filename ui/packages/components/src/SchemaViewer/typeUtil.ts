'use client';

import type { ValueNode } from './types';

export function capitalize(s: string): string {
  if (!s) return s;
  return s.charAt(0).toUpperCase() + s.slice(1);
}

export function repeatArrayBrackets(layers: number): string {
  if (!layers || layers <= 0) return '';
  return '[]'.repeat(layers);
}

export function valueTypeLabel(value: ValueNode): string {
  if (Array.isArray(value.type)) {
    const parts = value.type.map(capitalize).sort((a, b) => a.localeCompare(b));
    return parts.join(' | ');
  }

  return capitalize(value.type);
}
