import { createContext, useCallback, useContext, useMemo, useState } from 'react';

type ExpansionProviderProps = {
  children: React.ReactNode;
  defaultExpandedPaths?: string[];
};

type ExpansionContextValue = {
  isExpanded: (path: string) => boolean;
  toggle: (path: string) => void;
  toggleRecursive: (path: string, allPaths: string[]) => void;
};

const ExpansionContext = createContext<ExpansionContextValue | undefined>(undefined);

export function ExpansionProvider({ children, defaultExpandedPaths }: ExpansionProviderProps) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set(defaultExpandedPaths ?? []));

  const isExpanded = useCallback((path: string) => expanded.has(path), [expanded]);
  const toggle = useCallback((path: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  }, []);

  const toggleRecursive = useCallback((path: string, allPaths: string[]) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        // If collapsing, only remove the clicked path
        next.delete(path);
      } else {
        // If expanding, add the path and all descendant paths
        next.add(path);
        allPaths.forEach((p) => next.add(p));
      }
      return next;
    });
  }, []);

  const value = useMemo(
    () => ({ isExpanded, toggle, toggleRecursive }),
    [isExpanded, toggle, toggleRecursive]
  );

  return <ExpansionContext.Provider value={value}>{children}</ExpansionContext.Provider>;
}

export function useExpansion(): ExpansionContextValue {
  const ctx = useContext(ExpansionContext);
  if (!ctx) throw new Error('useExpansion must be used within an ExpansionProvider');

  return ctx;
}
