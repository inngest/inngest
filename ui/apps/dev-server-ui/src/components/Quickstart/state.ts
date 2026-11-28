export type Completion = {
  version: 1;
  server: string;
  completedAt: string;
  appID: string;
  functionID: string;
  runID: string;
};

export type Attempt = {
  mode?: 'agent';
  functionID: string;
  startedAt: string;
  runID?: string;
};

export type QuickstartRun = {
  id: string;
  appID: string;
  status: string;
  queuedAt: string;
  endedAt: string | null;
  function: { id: string; name: string; slug: string };
};

export const storageKey = (server: string) => `inngest:quickstart:v1:${server}`;

export function readCompletion(
  raw: string | null,
  server: string,
): Completion | null {
  try {
    const value = JSON.parse(raw || 'null');
    return value?.version === 1 &&
      value.server === server &&
      Number.isFinite(Date.parse(value.completedAt)) &&
      ['appID', 'functionID', 'runID'].every(
        (key) => typeof value[key] === 'string' && value[key],
      )
      ? value
      : null;
  } catch {
    return null;
  }
}

export function readAttempt(raw: string | null): Attempt | null {
  try {
    const value = JSON.parse(raw || 'null');
    if (
      typeof value?.functionID !== 'string' ||
      (!value.functionID && value.mode !== 'agent') ||
      !Number.isFinite(Date.parse(value.startedAt)) ||
      (value.runID !== undefined && typeof value.runID !== 'string')
    )
      return null;
    return {
      ...(value.mode === 'agent' ? { mode: 'agent' as const } : {}),
      functionID: value.functionID,
      startedAt: value.startedAt,
      runID: value.runID,
    };
  } catch {
    return null;
  }
}

// Never guess which invocation produced a run when more than one matches.
export function matchingRun(
  runs: QuickstartRun[],
  attempt: Attempt | null,
  hasMore: boolean,
) {
  if (!attempt) return runs.find((run) => run.status === 'COMPLETED');
  const candidates = runs.filter(
    (run) =>
      run.function.id === attempt.functionID &&
      Date.parse(run.queuedAt) >= Date.parse(attempt.startedAt),
  );
  if (attempt.runID) return candidates.find((run) => run.id === attempt.runID);
  return !hasMore && candidates.length === 1 ? candidates[0] : undefined;
}
