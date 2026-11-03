import { Link } from '@inngest/components/Link';
import { useEffect } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { AgentSetup } from './AgentSetup';
import { FirstRun } from './FirstRun';
import { ManualSetup } from './ManualSetup';
import { useQuickstart } from './QuickstartProvider';
import { useQuickstartTracking } from './useQuickstartTracking';

export function Quickstart({
  showSetupGuide = false,
}: {
  showSetupGuide?: boolean;
}) {
  const { server, apps, functions, info, completion, run } = useQuickstart();
  const completedRunID =
    completion?.runID || (run?.status === 'COMPLETED' ? run.id : undefined);
  const track = useQuickstartTracking(server);
  const hasFunctions = functions.length > 0;
  useEffect(() => {
    if (!apps.data) return;
    track(
      'viewed',
      {
        entry_stage: completion ? 'complete' : hasFunctions ? 'run' : 'setup',
      },
      true,
    );
    if (hasFunctions && !completion) track('app_ready', {}, true);
  }, [apps.data, completion, hasFunctions, track]);
  const connectedAppNames = (apps.data?.apps || [])
    .filter((app) => app.connected)
    .map((app) => app.name);
  const connected = connectedAppNames.length > 0;
  return (
    <main className="mx-auto w-full max-w-[648px] px-6 py-12 space-y-8">
      <div className="space-y-2">
        <h1 className="text-xl font-medium">Run your first Inngest function</h1>
        <p className="text-muted text-sm">
          Set up an app, create a function, and test a run.{' '}
          <Link
            className="inline-flex max-w-full align-baseline"
            href="https://www.inngest.com/docs"
            onClick={() => track('docs_clicked', { guide: 'overview' })}
          >
            Read docs
          </Link>
        </p>
      </div>
      {info.isError || apps.isError ? (
        <Alert severity="error">
          Could not load Dev Server status.{' '}
          <Button
            label="Retry"
            onClick={() => {
              info.refetch();
              apps.refetch();
            }}
          />
        </Alert>
      ) : !apps.data ? (
        <p role="status">Loading apps…</p>
      ) : (
        <>
          {(showSetupGuide || !functions.length) && (
            <>
              <AgentSetup server={server} />
              {connected && !showSetupGuide ? (
                <Alert severity="warning">
                  Your app is connected but has no invokable functions. Register
                  a function and check the app’s sync status. Durable endpoints
                  are available on the Functions page.
                  <div className="mt-3 flex gap-3">
                    <Link
                      className="inline-flex max-w-full align-baseline"
                      href="/apps"
                      target="_self"
                    >
                      View apps
                    </Link>
                  </div>
                </Alert>
              ) : (
                <>
                  <div className="text-muted flex items-center gap-2 font-mono text-xs uppercase">
                    <span
                      aria-hidden="true"
                      className="border-subtle flex-1 border-t"
                    />
                    <span className="shrink-0">Or set it up manually</span>
                    <span
                      aria-hidden="true"
                      className="border-subtle flex-1 border-t"
                    />
                  </div>
                  <ManualSetup
                    discoveryEnabled={!!info.data?.isDiscoveryEnabled}
                    connectedAppNames={connectedAppNames}
                    disconnected={!connected && !!apps.data.apps.length}
                  />
                </>
              )}
            </>
          )}
          <FirstRun showSetupGuide={showSetupGuide} />
          {showSetupGuide && completedRunID && (
            <Link
              href={`/runs?runID=${encodeURIComponent(completedRunID)}`}
              target="_self"
            >
              View your run
            </Link>
          )}
        </>
      )}
    </main>
  );
}
