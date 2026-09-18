import type { PathCreator } from '@inngest/components/SharedContext/usePathCreator';

import { InsightsColumnHint } from '@/store/generated';
import {
  badgeHintHref,
  idHintHref,
  isSessionHintValue,
  sessionPairs,
} from './hintLinks';
import {
  resolvedPathKey,
  stringifyWithSpans,
  type JsonSpans,
  type ResolvedPathSegment,
} from './jsonSpans';

// Structural mirror of pkg/duckdb/insights.PathSegment/PathHint (surfaced
// via GQL as InsightsPathSegment/InsightsPathHint) -- generated.ts inlines
// each query's own result shape rather than exporting standalone named
// types for these, so callers pass whatever shape their query produced.
export type PathSegmentLike = { key: string | null; wildcard: boolean };
export type PathHintLike = {
  path: readonly PathSegmentLike[];
  hint: InsightsColumnHint;
};

// OffsetLink is NewCodeBlock's exact-range monacoLinks entry shape -- an
// explicit character range into its rendered content, rather than a
// literal substring to search for. Built from a structural (path-based)
// match, not text content, so two occurrences of the identical string
// value at different JSON paths (e.g. an unrelated field that happens to
// equal a nearby session ID) can never be confused with one another.
export type OffsetLink = {
  startOffset: number;
  endOffset: number;
  href: string;
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

type PathMatch = { value: unknown; path: ResolvedPathSegment[] };

// collectAtPath walks value along path (root to leaf, exactly like
// pkg/duckdb/insights' own resolveSubPathHint/resolveArrayElementHint
// read a PathHint's Path), returning every value found there together
// with the *concrete* path (real array indices/object keys, no
// wildcards) that reached it -- a wildcard step fans out over an array
// (one result per element), a key step reads that literal object field
// (one result, or none for a non-object/missing field). An empty path
// (the column's own whole value) returns [{value, path: resolvedSoFar}]
// unconditionally, matching PathHint's own "empty Path is the
// whole-value hint" convention. The resolved path is what makes a match
// locatable in a JsonSpans -- resolvedPathKey(match.path) is exactly the
// key stringifyWithSpans recorded that same node's own range under, since
// both walk the identical value using the identical path-building rule.
function collectAtPath(
  value: unknown,
  path: readonly PathSegmentLike[],
  resolvedSoFar: ResolvedPathSegment[] = [],
): PathMatch[] {
  if (path.length === 0) return [{ value, path: resolvedSoFar }];
  const [step, ...rest] = path as [PathSegmentLike, ...PathSegmentLike[]];

  if (step.wildcard) {
    if (!Array.isArray(value)) return [];
    return value.flatMap((item, i) =>
      collectAtPath(item, rest, [...resolvedSoFar, i]),
    );
  }
  if (step.key == null || value == null || typeof value !== 'object') {
    return [];
  }
  return collectAtPath((value as Record<string, unknown>)[step.key], rest, [
    ...resolvedSoFar,
    step.key,
  ]);
}

// parsedJSONValue returns value ready for collectAtPath/stringifyWithSpans
// to walk -- the driver can report a JSON column as either an
// already-decoded object/array or a raw JSON string, so a string that
// looks like one (cheap prefix/suffix check, mirroring json.ts's own
// mayBeJSONArray/mayBeJSONObject) is parsed first. Returns null --
// meaning "render as opaque text, no path hints applicable" -- for a
// string that doesn't look like JSON or fails to parse.
function parsedJSONValue(value: unknown): unknown | null {
  if (typeof value !== 'string') return value;
  const looksLikeJson =
    (value.startsWith('[') && value.endsWith(']')) ||
    (value.startsWith('{') && value.endsWith('}'));
  if (!looksLikeJson) return null;
  try {
    return JSON.parse(value);
  } catch {
    return null;
  }
}

function pushLink(
  links: OffsetLink[],
  ranges: Map<string, [number, number]>,
  atPath: readonly ResolvedPathSegment[],
  href: string | undefined,
) {
  if (!href) return;
  const range = ranges.get(resolvedPathKey(atPath));
  if (!range) return;
  links.push({ startOffset: range[0], endOffset: range[1], href });
}

// sessionLinksForMatch handles HintSession's two distinct value shapes
// (pkg/duckdb/insights/tables.go's own doc comments): one {key, id}
// STRUCT (runs.sessions' array element -- collectAtPath's wildcard step
// already isolated one element, so its own literal "key"/"id" fields are
// linked directly), or the whole per-run/per-event sessions map
// (events.meta.sessions / runs.inputs[*].meta.sessions -- a Go map has no
// array-wildcard pathHints step to isolate one entry at a time, so the
// match is the entire map and every one of its own entries -- located by
// its own dynamic object key, not a declared PathHint path -- is a
// session pair in its own right).
function sessionLinksForMatch(
  value: unknown,
  path: readonly ResolvedPathSegment[],
  spans: Pick<JsonSpans, 'valueRange' | 'keyRange'>,
  pathCreator: PathCreator,
): OffsetLink[] {
  const links: OffsetLink[] = [];

  if (isSessionHintValue(value)) {
    pushLink(
      links,
      spans.valueRange,
      [...path, 'key'],
      pathCreator.sessions?.({ sessionKey: value.key }),
    );
    pushLink(
      links,
      spans.valueRange,
      [...path, 'id'],
      pathCreator.session?.({ sessionKey: value.key, sessionId: value.id }),
    );
    return links;
  }

  for (const pair of sessionPairs(value)) {
    pushLink(
      links,
      spans.keyRange,
      [...path, pair.key],
      pathCreator.sessions?.({ sessionKey: pair.key }),
    );
    pushLink(
      links,
      spans.valueRange,
      [...path, pair.key],
      pathCreator.session?.({ sessionKey: pair.key, sessionId: pair.id }),
    );
  }
  return links;
}

function linksForMatch(
  hint: InsightsColumnHint,
  match: PathMatch,
  spans: Pick<JsonSpans, 'valueRange' | 'keyRange'>,
  pathCreator: PathCreator,
): OffsetLink[] {
  const { value, path } = match;
  if (value == null) return [];

  if (hint === InsightsColumnHint.Session) {
    return sessionLinksForMatch(value, path, spans, pathCreator);
  }

  const links: OffsetLink[] = [];
  if (
    hint === InsightsColumnHint.AppId ||
    hint === InsightsColumnHint.FunctionId
  ) {
    pushLink(
      links,
      spans.valueRange,
      path,
      badgeHintHref(hint, pathCreator, String(value)),
    );
  } else if (
    hint === InsightsColumnHint.RunId ||
    hint === InsightsColumnHint.EventId
  ) {
    pushLink(
      links,
      spans.valueRange,
      path,
      idHintHref(hint, pathCreator, String(value)),
    );
  }
  return links;
}

// buildJsonCellRender is a JSON-typed cell's single source of truth for
// both its rendered text and its monacoLinks: both come from the *same*
// stringifyWithSpans pass over the *same* parsed value, so a link's
// offsets are always exactly the range of the value that produced it --
// never a separately-computed, potentially out-of-sync search. Returns
// monacoLinks: undefined (with the raw string as content, unformatted)
// when rawValue is a string that isn't actually valid/JSON-shaped.
export function buildJsonCellRender(
  pathHints: readonly PathHintLike[] | null | undefined,
  rawValue: unknown,
  pathCreator: PathCreator,
): { content: string; monacoLinks: OffsetLink[] | undefined } {
  const parsed = parsedJSONValue(rawValue);
  if (parsed === null && typeof rawValue === 'string') {
    return { content: rawValue, monacoLinks: undefined };
  }

  const spans = stringifyWithSpans(parsed);
  if (!pathHints || pathHints.length === 0) {
    return { content: spans.text, monacoLinks: undefined };
  }

  const links = pathHints.flatMap((ph) =>
    collectAtPath(parsed, ph.path).flatMap((match) =>
      linksForMatch(ph.hint, match, spans, pathCreator),
    ),
  );
  return {
    content: spans.text,
    monacoLinks: links.length > 0 ? links : undefined,
  };
}

// buildScalarPathHintLink is a plaintext (non-JSON) cell's monacoLinks --
// e.g. a bare run_id column. Only a column's own whole-value (root) hint
// can ever apply here (a sub-path hint only means something inside real
// JSON structure), and the whole rendered text is the value, so there's
// at most one link covering it entirely. Session has no scalar shape
// (pkg/duckdb/insights/tables.go), so it never produces one here.
export function buildScalarPathHintLink(
  pathHints: readonly PathHintLike[] | null | undefined,
  text: string,
  pathCreator: PathCreator,
): OffsetLink[] | undefined {
  const hint = rootHint(pathHints);
  if (!hint) return undefined;

  let href: string | undefined;
  if (
    hint === InsightsColumnHint.AppId ||
    hint === InsightsColumnHint.FunctionId
  ) {
    href = badgeHintHref(hint, pathCreator, text);
  } else if (
    hint === InsightsColumnHint.RunId ||
    hint === InsightsColumnHint.EventId
  ) {
    href = idHintHref(hint, pathCreator, text);
  }
  return href ? [{ startOffset: 0, endOffset: text.length, href }] : undefined;
}
