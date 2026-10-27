import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react';
import { createDevServerURL } from '@/hooks/useFeatureFlags';
import { useInfoQuery } from '@/store/devApi';
import { useQuickstartTracking } from './useQuickstartTracking';
import {
  AppMethod,
  FunctionRunStatus,
  useGetAppsQuery,
  useQuickstartRunsQuery,
} from '@/store/generated';
import {
  matchingRun,
  readAttempt,
  readCompletion,
  storageKey,
  type Attempt,
  type Completion,
} from './state';

function useQuickstartState() {
  const info = useInfoQuery();
  const [server, setServer] = useState('');
  const track = useQuickstartTracking(server);
  const [completion, setCompletion] = useState<Completion | null>(null);
  const [attempt, setAttemptState] = useState<Attempt | null>(null);
  const isDevServer = !!info.data && !info.data.isSingleNodeService;
  const apps = useGetAppsQuery(undefined, {
    skip: !isDevServer,
    pollingInterval: completion ? 0 : 1500,
  });
  const functions = (apps.data?.apps || [])
    .filter((app) => app.connected && app.method !== AppMethod.Api)
    .flatMap((app) =>
      app.functions.map((fn) => ({ ...fn, appID: app.id, appName: app.name })),
    );

  useEffect(() => {
    const origin = new URL(createDevServerURL('/'), window.location.origin)
      .origin;
    setServer(origin);
    try {
      setCompletion(
        readCompletion(localStorage.getItem(storageKey(origin)), origin),
      );
      setAttemptState(
        readAttempt(sessionStorage.getItem(`${storageKey(origin)}:attempt`)),
      );
    } catch {
      /* Onboarding still works when browser storage is unavailable. */
    }
  }, []);

  const setAttempt = useCallback(
    (value: Attempt | null) => {
      setAttemptState(value);
      if (!server) return;
      try {
        const key = `${storageKey(server)}:attempt`;
        if (value) sessionStorage.setItem(key, JSON.stringify(value));
        else sessionStorage.removeItem(key);
      } catch {
        /* Keep the current session usable without storage. */
      }
    },
    [server],
  );
  const observedAttempt =
    attempt?.mode === 'agent' && !attempt.functionID && functions.length === 1
      ? { ...attempt, functionID: functions[0].id }
      : attempt;
  const functionIDs = observedAttempt
    ? observedAttempt.functionID
      ? [observedAttempt.functionID]
      : []
    : functions.map((fn) => fn.id).sort();
  const runs = useQuickstartRunsQuery(
    {
      functionIDs,
      from: observedAttempt?.startedAt || '1970-01-01T00:00:00.000Z',
      status: attempt ? null : [FunctionRunStatus.Completed],
    },
    {
      skip:
        !server ||
        !isDevServer ||
        (!!completion && !attempt) ||
        functionIDs.length === 0,
      pollingInterval: completion && !attempt ? 0 : 1500,
    },
  );
  const candidates = runs.currentData?.runs.edges.map(({ node }) => node) || [];
  const run = matchingRun(
    candidates,
    observedAttempt,
    runs.currentData?.runs.pageInfo.hasNextPage || false,
  );

  useEffect(() => {
    if (completion || !server || run?.status !== 'COMPLETED') return;
    const value: Completion = {
      version: 1,
      server,
      completedAt: run.endedAt || new Date().toISOString(),
      appID: run.appID,
      functionID: run.function.id,
      runID: run.id,
    };
    // Historical runs establish returning-user state, not new conversions.
    if (attempt)
      track(
        'first_run_completed',
        {
          path: attempt.mode === 'agent' ? 'agent' : 'manual',
        },
        true,
      );
    setCompletion(value);
    try {
      localStorage.setItem(storageKey(server), JSON.stringify(value));
    } catch {
      /* In-memory fallback. */
    }
    setAttempt(null);
  }, [server, run, completion, setAttempt, attempt, track]);

  return {
    server,
    completion,
    attempt: observedAttempt,
    setAttempt,
    functions,
    apps,
    info,
    runs,
    candidates,
    run,
    isDevServer,
    ready:
      !!server &&
      !!info.data &&
      (!isDevServer ||
        !!completion ||
        apps.isError ||
        (!!apps.data &&
          (functions.length === 0 ||
            (observedAttempt?.mode === 'agent' &&
              !observedAttempt.functionID) ||
            !!runs.currentData ||
            runs.isError))),
  };
}
const QuickstartContext = createContext<ReturnType<
  typeof useQuickstartState
> | null>(null);
export function QuickstartProvider({ children }: { children: ReactNode }) {
  return (
    <QuickstartContext.Provider value={useQuickstartState()}>
      {children}
    </QuickstartContext.Provider>
  );
}
export function useQuickstart() {
  const context = useContext(QuickstartContext);
  if (!context) throw new Error('QuickstartProvider is required');
  return context;
}
