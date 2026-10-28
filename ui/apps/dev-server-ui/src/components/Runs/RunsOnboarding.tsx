import { useEffect, type ReactNode } from 'react';
import { Link } from '@inngest/components/Link';
import { FirstRun } from '@/components/Quickstart/FirstRun';
import { useQuickstart } from '@/components/Quickstart/QuickstartProvider';
import { useQuickstartTracking } from '@/components/Quickstart/useQuickstartTracking';

export function RunsOnboarding({
  runID,
  children,
}: {
  runID?: string;
  children: ReactNode;
}) {
  const { ready, isDevServer, completion, functions, server, info } =
    useQuickstart();
  const track = useQuickstartTracking(server);
  const showFirstRun =
    ready && isDevServer && !completion && !runID && functions.length > 0;
  useEffect(() => {
    if (!showFirstRun) return;
    track('viewed', { entry_stage: 'run' }, true);
    track('app_ready', {}, true);
  }, [showFirstRun, track]);

  if (!runID && !ready && !info.isError)
    return (
      <div role="status" className="text-muted p-6">
        Loading runs…
      </div>
    );
  if (!showFirstRun) return children;
  return (
    <main className="mx-auto w-full max-w-[648px] px-6 py-12 space-y-6">
      <FirstRun />
      <p className="text-muted text-sm">
        Need the setup guide or agent prompt?{' '}
        <Link href="/?quickstart=true" target="_self">
          Open Quickstart
        </Link>
      </p>
    </main>
  );
}
