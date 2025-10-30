'use client';

import { createContext, useCallback, useContext, useMemo, useState } from 'react';

type ExpansionContextValue = {
  isExpanded: (path: string) => boolean;
  toggle: (path: string) => void;
};

const ExpansionContext = createContext<ExpansionContextValue | undefined>(undefined);

export function ExpansionProvider({ children }: { children: React.ReactNode }) {
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set());

  const isExpanded = useCallback((path: string) => expanded.has(path), [expanded]);
  const toggle = useCallback((path: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  }, []);

  const value = useMemo(() => ({ isExpanded, toggle }), [isExpanded, toggle]);

  return <ExpansionContext.Provider value={value}>{children}</ExpansionContext.Provider>;
}

export function useExpansion(): ExpansionContextValue {
  const ctx = useContext(ExpansionContext);
  if (!ctx) throw new Error('useExpansion must be used within an ExpansionProvider');

  return ctx;
}
