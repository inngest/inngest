import type { PathCreator } from '@inngest/components/SharedContext/usePathCreator';

import { InsightsColumnHint, InsightsColumnType } from '@/store/generated';
import { badgeHintHref, idHintHref, isSessionHintValue } from './hintLinks';

// Structural mirror of pkg/duckdb/insights.PathSegment/PathHint (surfaced
// via GQL as InsightsPathSegment/InsightsPathHint) -- generated.ts inlines
// each query's own result shape rather than exporting standalone named
// types for these, so callers pass whatever shape their query produced.
export type PathSegmentLike = { key: string | null; wildcard: boolean };
export type PathHintLike = {
  path: readonly PathSegmentLike[];
  hint: InsightsColumnHint;
};

// rootHint returns a column's own whole-value hint -- the pathHints entry
// with an empty path, if any -- mirroring pkg/duckdb/insights' own
// Column.Hint()/knownColumn.hint(). There's no separate plain `hint`
// field on InsightsQueryColumn any more; pathHints is the only source.
export function rootHint(
  pathHints: readonly PathHintLike[] | null | undefined,
): InsightsColumnHint | null {
  return pathHints?.find((ph) => ph.path.length === 0)?.hint ?? null;
}

// collectAtPath walks value along path (root to leaf, exactly like
// pkg/duckdb/insights' own resolveSubPathHint/resolveArrayElementHint
// read a PathHint's Path), returning every value found there -- a
// wildcard step fans out over an array (one result per element, or none
// for a non-array), a key step reads that literal object field (one
// result, or none for a non-object/missing field). An empty path (the
// column's own whole value) returns [value] unconditionally, matching
// PathHint's own "empty Path is the whole-value hint" convention.
function collectAtPath(
  value: unknown,
  path: readonly PathSegmentLike[],
): unknown[] {
  if (path.length === 0) return [value];
  const [step, ...rest] = path as [PathSegmentLike, ...PathSegmentLike[]];

  if (step.wildcard) {
    if (!Array.isArray(value)) return [];
    return value.flatMap((item) => collectAtPath(item, rest));
  }
  if (step.key == null || value == null || typeof value !== 'object') {
    return [];
  }
  return collectAtPath((value as Record<string, unknown>)[step.key], rest);
}

// parsedJSONValue returns value ready for collectAtPath to walk -- the
// driver can report a JSON column as either an already-decoded object/
// array or a raw JSON string (CellValueDisplay's own content computation
// handles the same ambiguity), so a string is parsed first. Falls back to
// the original string on a parse failure rather than throwing.
function parsedJSONValue(value: unknown): unknown {
  if (typeof value !== 'string') return value;
  try {
    return JSON.parse(value);
  } catch {
    return value;
  }
}

// linksForHintedValue builds every monacoLinks entry for one hinted value
// found at some pathHints entry's path -- e.g. one array_ids[i], one
// attributes["_inngest.app.name"], or (isJson false) the whole scalar
// cell value itself. text is JSON.stringify'd when isJson (matching how
// the value actually appears in CellValueDisplay's pretty-printed JSON
// content) or the raw string otherwise (a plaintext-rendered scalar
// column, e.g. a bare run_id column).
function linksForHintedValue(
  hint: InsightsColumnHint,
  value: unknown,
  isJson: boolean,
  pathCreator: PathCreator,
): { text: string; href: string }[] {
  if (value == null) return [];
  const text = (v: unknown) => (isJson ? JSON.stringify(v) : String(v));

  if (hint === InsightsColumnHint.Session) {
    // A session-hinted value is always a {key, id} object (runs.sessions'
    // element shape / event meta.sessions' per-key entry), never a bare
    // scalar -- two link entries, one for each field, since either
    // pathCreator method may be unavailable on its own.
    if (!isSessionHintValue(value)) return [];
    const links: { text: string; href: string }[] = [];
    const keysHref = pathCreator.sessions?.({ sessionKey: value.key });
    if (keysHref) links.push({ text: text(value.key), href: keysHref });
    const sessionHref = pathCreator.session?.({
      sessionKey: value.key,
      sessionId: value.id,
    });
    if (sessionHref) links.push({ text: text(value.id), href: sessionHref });
    return links;
  }

  if (
    hint === InsightsColumnHint.AppId ||
    hint === InsightsColumnHint.FunctionId
  ) {
    return [
      {
        text: text(value),
        href: badgeHintHref(hint, pathCreator, String(value)),
      },
    ];
  }
  if (
    hint === InsightsColumnHint.RunId ||
    hint === InsightsColumnHint.EventId
  ) {
    return [
      { text: text(value), href: idHintHref(hint, pathCreator, String(value)) },
    ];
  }
  return [];
}

// buildPathHintMonacoLinks resolves every pathHints entry (root and
// sub-path alike) against value, producing NewCodeBlock's monacoLinks for
// every hinted value actually found -- a bare hinted scalar column (one
// root entry, one match), a hinted array column (one wildcard entry, one
// match per element), or a hinted value nested anywhere inside a JSON
// column's own data (e.g. attributes' OTel keys, inputs' per-element
// fields), all resolved the same way this package's own
// resolveSubPathHint/resolveArrayElementHint do on the backend.
export function buildPathHintMonacoLinks(
  pathHints: readonly PathHintLike[] | null | undefined,
  value: unknown,
  columnType: InsightsColumnType,
  pathCreator: PathCreator,
): { text: string; href: string }[] | undefined {
  if (!pathHints || pathHints.length === 0 || value == null) return undefined;

  const isJson = columnType === InsightsColumnType.Json;
  const root = isJson ? parsedJSONValue(value) : value;

  const links = pathHints.flatMap((ph) =>
    collectAtPath(root, ph.path).flatMap((match) =>
      linksForHintedValue(ph.hint, match, isJson, pathCreator),
    ),
  );
  return links.length > 0 ? links : undefined;
}
