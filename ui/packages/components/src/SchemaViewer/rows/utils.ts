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

export const handleCopyValue = (node: SchemaNode) => {
  return async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      // The path is formatted as "eventName.field.subfield..."
      // We need to remove the event name prefix (which can contain dots)
      // We do this by finding the first occurrence of a standard field after the event name
      // Strategy: The path after the event name will always start with a dot followed by a field
      let relativePath = node.path;

      for (const field of STANDARD_EVENT_FIELDS) {
        const pattern = `.${field}`;
        const index = node.path.indexOf(pattern);
        if (index !== -1) {
          // Found a standard field, extract from this point (excluding the leading dot)
          relativePath = node.path.substring(index + 1);
          break;
        }
      }

      await navigator.clipboard.writeText(relativePath);
      toast.success('Path copied to clipboard');
    } catch (err) {
      console.error('Failed to copy:', err);
      toast.error('Failed to copy to clipboard');
    }
  };
};
