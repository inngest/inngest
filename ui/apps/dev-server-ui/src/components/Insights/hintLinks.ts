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
