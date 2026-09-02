import { isFunctionRunStatus } from '@inngest/components/types/functionRun';
import type { Run } from '@inngest/components/RunsPage/types';

import type { InngestAPIFetch } from '@/queries/useInngestAPIFetch';

export const RUNS_CEL_MAX_BYTES = 2048;
export const REST_RUNS_REFETCH_INTERVAL_MS = 1000;

export function restRunsRefetchInterval(hasCEL: boolean, pageCount: number) {
  return !hasCEL && pageCount === 1
    ? REST_RUNS_REFETCH_INTERVAL_MS
    : false;
}

// TODO: Replace these handwritten REST wire types with types generated from
// the protobuf HTTP contract.
export type RestFunctionRun = {
  id: string;
  function?: { id?: string; name?: string; slug?: string };
  app?: { id?: string };
  status: string;
  queuedAt: string;
  startedAt?: string;
  endedAt?: string;
  durationMs?: number | string;
  trigger?: {
    eventName?: string;
    isBatch?: boolean;
    cronSchedule?: string;
  };
  isDeferred?: boolean;
  hasAi?: boolean;
};

export type RestRunsPage = {
  data: RestFunctionRun[];
  page: { cursor?: string; hasMore: boolean; limit: number };
};

export class RunsAPIError extends Error {
  constructor(
    message: string,
    readonly code?: string,
    readonly status?: number,
    readonly retryAfter?: string | null,
  ) {
    super(message);
  }
}

const RUNS_RATE_LIMIT_MAX_RETRIES = 3;

async function waitForRetry(milliseconds: number, signal?: AbortSignal) {
  if (signal?.aborted) throw signal.reason;

  await new Promise<void>((resolve, reject) => {
    const onAbort = () => {
      clearTimeout(timeout);
      reject(signal?.reason);
    };
    const timeout = setTimeout(() => {
      signal?.removeEventListener('abort', onAbort);
      resolve();
    }, milliseconds);
    signal?.addEventListener('abort', onAbort, { once: true });
  });
}

function rateLimitDelay(error: RunsAPIError, attempt: number) {
  if (error.retryAfter) {
    const seconds = Number(error.retryAfter);
    if (Number.isFinite(seconds) && seconds >= 0) return seconds * 1000;

    const date = Date.parse(error.retryAfter);
    if (!Number.isNaN(date)) return Math.max(0, date - Date.now());
  }
  return 1000 * 2 ** attempt;
}

// Adapt protobuf JSON to the GraphQL-shaped model currently consumed by the
// runs table. Nested or renamed fields change shape; defaults normalize values
// that protobuf JSON omits.
export function restFunctionRunToTableRun(run: RestFunctionRun): Run {
  if (!run.id || !run.function?.id || !run.function.name || !run.app?.id) {
    throw new RunsAPIError(
      'Runs response is missing required IDs or references',
    );
  }
  if (!isFunctionRunStatus(run.status)) {
    throw new RunsAPIError(
      `Runs response has an unsupported status: ${run.status}`,
    );
  }

  return {
    // REST-to-table shape and type adaptations.
    id: run.id,
    app: { externalID: run.app.id, name: run.app.id },
    function: {
      name: run.function.name,
      slug: run.function.slug || run.function.id,
    },
    status: run.status,
    durationMS: run.durationMs === undefined ? null : Number(run.durationMs),
    eventName: run.trigger?.eventName ?? null,
    isBatch: run.trigger?.isBatch ?? false,
    cronSchedule: run.trigger?.cronSchedule ?? null,

    // Fields already matching the table shape; normalize omitted values where
    // the table model requires an explicit default.
    queuedAt: run.queuedAt,
    startedAt: run.startedAt ?? null,
    endedAt: run.endedAt ?? null,
    isDeferred: run.isDeferred ?? false,
    hasAI: run.hasAi,
  };
}

export async function fetchRunsPage(
  apiFetch: InngestAPIFetch,
  pathname: string,
  params: URLSearchParams,
  signal?: AbortSignal,
): Promise<RestRunsPage> {
  for (let attempt = 0; ; attempt += 1) {
    try {
      return await fetchRunsPageAttempt(apiFetch, pathname, params, signal);
    } catch (error) {
      if (
        !(error instanceof RunsAPIError) ||
        error.status !== 429 ||
        attempt >= RUNS_RATE_LIMIT_MAX_RETRIES
      ) {
        throw error;
      }
      await waitForRetry(rateLimitDelay(error, attempt), signal);
    }
  }
}

async function fetchRunsPageAttempt(
  apiFetch: InngestAPIFetch,
  pathname: string,
  params: URLSearchParams,
  signal?: AbortSignal,
): Promise<RestRunsPage> {
  const query = params.toString();
  const response = await apiFetch(query ? `${pathname}?${query}` : pathname, {
    signal,
  });
  const body = await response.json().catch(() => null);
  if (!response.ok) {
    const item = body?.errors?.[0];
    throw new RunsAPIError(
      item?.message || response.statusText || 'Unable to fetch runs',
      item?.code,
      response.status,
      response.headers.get('Retry-After'),
    );
  }
  if (!body?.page || (body.data !== undefined && !Array.isArray(body.data))) {
    throw new RunsAPIError('Runs response has an invalid shape');
  }
  return {
    ...body,
    data: body.data ?? [],
    page: {
      ...body.page,
      hasMore: body.page.hasMore ?? false,
    },
  } as RestRunsPage;
}

// TODO: Expose the scan frontier as response metadata. For implementation
// expediency, this display-only helper violates cursor opacity by decoding the
// Cloud metrics cursor. Pagination still passes the cursor through unchanged;
// decoding failures only omit the progress label.
export function decodeRunsFrontier(
  cursor: string | undefined,
  timeField: string,
): Date | undefined {
  if (!cursor) return;
  try {
    const normalized = cursor.replace(/-/g, '+').replace(/_/g, '/');
    const decoded = JSON.parse(atob(normalized));
    const field = timeField.toLowerCase();
    const value = decoded?.c?.[field]?.v;
    if (typeof value !== 'number') return;
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? undefined : date;
  } catch {
    return;
  }
}
