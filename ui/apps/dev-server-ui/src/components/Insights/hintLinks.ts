import type { PathCreator } from '@inngest/components/SharedContext/usePathCreator';

import { InsightsColumnHint } from '@/store/generated';

// A session-hinted value is a {key, id} object (runs.sessions' element
// shape / event meta.sessions' per-key entry -- see
// pkg/duckdb/insights/tables.go's HintSession), not a bare ID string like
// every other hint -- both fields are needed to build a session link.
export type SessionHintValue = { key: string; id: string };

export function isSessionHintValue(value: unknown): value is SessionHintValue {
  return (
    typeof value === 'object' &&
    value !== null &&
    typeof (value as Record<string, unknown>).key === 'string' &&
    typeof (value as Record<string, unknown>).id === 'string'
  );
}

// sessionPairs normalizes a Session-hinted match into a flat list of
// {key, id} pairs -- a match can be either shape HintSession describes
// (pkg/duckdb/insights/tables.go): one {key, id} STRUCT (runs.sessions'
// array element -- collectAtPath's own wildcard step already isolates
// one element, so isSessionHintValue matches it directly), or the whole
// event.EventMeta.sessions map[string]string object (events.meta.sessions
// / runs.inputs[*].meta.sessions -- a Go map has no array-wildcard
// pathHints step to isolate one entry at a time, so the match is the
// entire map and every one of its own string-valued entries is a pair).
export function sessionPairs(value: unknown): SessionHintValue[] {
  if (isSessionHintValue(value)) return [value];
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return [];
  }
  const entries = Object.entries(value as Record<string, unknown>);
  if (
    entries.length === 0 ||
    !entries.every(([, v]) => typeof v === 'string')
  ) {
    return [];
  }
  return entries.map(([key, id]) => ({ key, id: id as string }));
}

export function idHintHref(
  hint: InsightsColumnHint.RunId | InsightsColumnHint.EventId,
  pathCreator: PathCreator,
  id: string,
): string {
  return hint === InsightsColumnHint.RunId
    ? pathCreator.runPopout({ runID: id })
    : pathCreator.eventPopout({ eventID: id });
}

export function badgeHintHref(
  hint: InsightsColumnHint.AppId | InsightsColumnHint.FunctionId,
  pathCreator: PathCreator,
  value: string,
): string {
  return hint === InsightsColumnHint.AppId
    ? pathCreator.app({ externalAppID: value })
    : pathCreator.function({ functionSlug: value });
}
