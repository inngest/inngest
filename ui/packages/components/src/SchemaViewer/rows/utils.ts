import { STANDARD_EVENT_FIELDS } from '@inngest/components/constants';
import { toast } from 'sonner';

import type { SchemaNode, TypedNode } from '../types';

export function collectAllExpandablePaths(node: SchemaNode): string[] {
  const paths: string[] = [];

  function traverse(n: SchemaNode) {
    if (n.kind === 'object') {
      paths.push(n.path);
      n.children.forEach(traverse);
    } else if (n.kind === 'array') {
      paths.push(n.path);
      traverse(n.element);
    } else if (n.kind === 'tuple') {
      paths.push(n.path);
      n.elements.forEach(traverse);
    }
  }

  // Start with children to avoid adding the current node's path
  if (node.kind === 'object') {
    node.children.forEach(traverse);
  }

  return paths;
}

export function getTypeLabel(node: TypedNode, override?: string): string {
  const type = node.type ?? '';
  return override ?? (Array.isArray(type) ? type.join(' | ') : type);
}

// The path is formatted as "eventName.field.subfield...", and the event name
// can contain dots itself. To drop the event name we look for the standard
// event field that starts the earliest path segment and keep everything from
// there. Taking the earliest one matters: a property under data can be named
// like a standard field ("data.name"), and that must not win over "data".
export function getRelativePath(path: string): string {
  let start = -1;

  for (const field of STANDARD_EVENT_FIELDS) {
    const pattern = `.${field}`;
    let index = path.indexOf(pattern);

    while (index !== -1) {
      // Only accept a whole segment, so that ".id" doesn't match ".ids"
      const next = path[index + pattern.length];
      if (next === undefined || next === '.' || next === '[') {
        break;
      }
      index = path.indexOf(pattern, index + 1);
    }

    if (index !== -1 && (start === -1 || index < start)) {
      start = index;
    }
  }

  // Skip the leading dot
  return start === -1 ? path : path.substring(start + 1);
}

export const handleCopyValue = (node: SchemaNode) => {
  return async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      const relativePath = getRelativePath(node.path);

      await navigator.clipboard.writeText(relativePath);
      toast.success('Path copied to clipboard');
    } catch (err) {
      console.error('Failed to copy:', err);
      toast.error('Failed to copy to clipboard');
    }
  };
};
