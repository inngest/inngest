import { isFunctionRunStatus } from '@inngest/components/types/functionRun';
import type { Run } from '@inngest/components/RunsPage/types';

export const RUNS_CEL_MAX_BYTES = 2048;

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
  hasAI?: boolean;
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
  ) {
    super(message);
  }
}

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
    id: run.id,
    app: { externalID: run.app.id, name: run.app.id },
    function: {
      name: run.function.name,
      slug: run.function.slug || run.function.id,
    },
    status: run.status,
    queuedAt: run.queuedAt,
    startedAt: run.startedAt ?? null,
    endedAt: run.endedAt ?? null,
    durationMS: run.durationMs === undefined ? null : Number(run.durationMs),
    eventName: run.trigger?.eventName ?? null,
    isBatch: run.trigger?.isBatch ?? false,
    cronSchedule: run.trigger?.cronSchedule ?? null,
    isDeferred: run.isDeferred ?? false,
    hasAI: run.hasAI,
  };
}

export async function fetchRunsPage(
  pathname: string,
  params: URLSearchParams,
  environmentSlug: string,
  signal?: AbortSignal,
): Promise<RestRunsPage> {
  const url = new URL(pathname, import.meta.env.VITE_API_URL);
  url.search = params.toString();
  const response = await fetch(url, {
    credentials: 'include',
    headers: { 'X-Inngest-Env': environmentSlug },
    signal,
  });
  const body = await response.json().catch(() => null);
  if (!response.ok) {
    const item = body?.errors?.[0];
    throw new RunsAPIError(
      item?.message || response.statusText || 'Unable to fetch runs',
      item?.code,
      response.status,
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
