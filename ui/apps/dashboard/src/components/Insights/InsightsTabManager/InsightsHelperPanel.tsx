'use client';

import { useMemo } from 'react';

export function InsightsHelperPanel({ active }: { active: null | string }) {
  const content = useMemo(() => {
    switch (active) {
      case 'AI':
        return <div className="text-sm">AI helper (placeholder)</div>;
      case 'Docs':
        return <div className="text-sm">Docs helper (placeholder)</div>;
      case 'Schemas':
        return <div className="text-sm">Schemas helper (placeholder)</div>;
      case 'Support':
        return <div className="text-sm">Support helper (placeholder)</div>;
      default:
        return null;
    }
  }, [active]);

  if (content === null) return null;

  return <div className="h-full w-full overflow-auto p-3">{content}</div>;
}
