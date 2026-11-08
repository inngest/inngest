import type { ReactNode } from 'react';
import { useBooleanFlag } from '@inngest/components/SharedContext/useBooleanFlag';
import { Navigate } from '@tanstack/react-router';

// DuckDBGuard renders its children only when the dev server runs w/ --duckdb
// (the "duckdb-insights" flag). The Insights and Sessions resolvers reject
// every request w/o it, so a direct link to those pages redirects to the runs
// page instead of mounting a page that can only error.
export function DuckDBGuard({ children }: { children: ReactNode }) {
  const { booleanFlag } = useBooleanFlag();
  const { value: enabled, isReady } = booleanFlag('duckdb-insights', false);

  if (!isReady) {
    return null;
  }
  if (!enabled) {
    return <Navigate to="/runs" replace />;
  }
  return <>{children}</>;
}
