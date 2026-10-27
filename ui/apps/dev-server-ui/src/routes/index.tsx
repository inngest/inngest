import { useEffect, useState } from 'react';
import { createFileRoute, Navigate } from '@tanstack/react-router';
import { TooltipProvider } from '@inngest/components/Tooltip';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import Layout from '@/components/Layout/Layout';
import { Quickstart } from '@/components/Quickstart/Quickstart';
import { useQuickstart } from '@/components/Quickstart/QuickstartProvider';

export const Route = createFileRoute('/')({
  validateSearch: (
    search: Record<string, unknown>,
  ): { quickstart?: boolean } => ({
    quickstart:
      search.quickstart === true || search.quickstart === 'true'
        ? true
        : undefined,
  }),
  component: Home,
});

export function Home() {
  const { quickstart } = Route.useSearch();
  const { ready, completion, info, run } = useQuickstart();
  // Decide once on entry: explicitly reopening a completed guide must not redirect.
  const [destination, setDestination] = useState<
    'quickstart' | 'guide' | 'runs'
  >();
  useEffect(() => {
    if (ready && !destination)
      setDestination(
        info.data?.isSingleNodeService ||
          ((completion || run?.status === 'COMPLETED') && !quickstart)
          ? 'runs'
          : quickstart && (completion || run?.status === 'COMPLETED')
          ? 'guide'
          : 'quickstart',
      );
  }, [ready, destination, completion, quickstart, info.data, run?.status]);
  if (info.data?.isSingleNodeService || destination === 'runs')
    return <Navigate to="/runs" replace />;
  return (
    <TooltipProvider delayDuration={0}>
      <Layout>
        {info.isError ? (
          <div className="p-6">
            <Alert severity="error">
              Could not load Dev Server status.{' '}
              <Button label="Retry" onClick={() => info.refetch()} />
            </Alert>
          </div>
        ) : destination ? (
          <Quickstart showSetupGuide={destination === 'guide'} />
        ) : (
          <div role="status" className="text-muted p-6">
            Loading quickstart…
          </div>
        )}
      </Layout>
    </TooltipProvider>
  );
}
