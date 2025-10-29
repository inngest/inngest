'use client';

import { useCallback, useState } from 'react';

import { NodeRenderer } from './NodeRenderer';
import type { SchemaNode as TreeNode } from './types';

export type SchemaViewerProps = {
  className?: string;
  root: TreeNode;
};

export function SchemaViewer({ className, root }: SchemaViewerProps) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const toggle = useCallback((path: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  }, []);

  return (
    <div className={className}>
      <NodeRenderer depth={0} expanded={expanded} node={root} onToggle={toggle} />
    </div>
  );
}
