import { TimeElement } from '../DetailsCard/Element';
import { KindInngestScore, KindPrefixInngestScore } from '../generated';

type ScoreMetadata = {
  kind: string;
  updatedAt: string;
  // Loose structural supertype of every SpanMetadata arm so both the V3 and V4
  // Trace types can be passed in. `object` (not Record<string, unknown>) is
  // required because generated interface value types (e.g. AIMetadata) lack an
  // implicit index signature; this type only filters by kind and reads via
  // Object.entries, so the looser bound is sufficient.
  values: object;
};

type ScoreRow = {
  name: string;
  updatedAt: string;
  value: number | boolean;
};

type ScoreTrace = {
  metadata?: ScoreMetadata[];
  childrenSpans?: ScoreTrace[];
};

// Scores come in two kind shapes:
// - `inngest.score.<name>` w/ `{value}` (pre #4482, and per name kinds going
//   forward)
// - `inngest.score` w/ `{"<name>": {value}}` (the legacy shared kind)
export function isScoreKind(kind: string): boolean {
  return (
    kind === KindInngestScore ||
    (kind.startsWith(KindPrefixInngestScore) && kind.length > KindPrefixInngestScore.length)
  );
}

// Normalizes either score kind shape to (name, raw value) pairs so readers
// don't care which one was written.
function scoreEntries(md: ScoreMetadata): [string, unknown][] {
  if (md.kind === KindInngestScore) {
    return Object.entries(md.values);
  }
  if (isScoreKind(md.kind)) {
    return [[md.kind.slice(KindPrefixInngestScore.length), md.values]];
  }
  return [];
}

export function collectScoreMetadata(trace?: ScoreTrace): ScoreMetadata[] {
  // Run views need child spans because scores attach where they are emitted.
  const metadata = trace?.metadata?.filter((md) => isScoreKind(md.kind)) ?? [];
  const childMetadata = trace?.childrenSpans?.flatMap((child) => collectScoreMetadata(child)) ?? [];

  return [...metadata, ...childMetadata];
}

// Trim floating-point noise and excess precision from non-integer scores;
// integers and booleans render as-is.
export function formatScoreValue(value: number | boolean): string {
  if (typeof value === 'number' && !Number.isInteger(value)) {
    return String(Number(value.toPrecision(4)));
  }
  return String(value);
}

function isScoreValue(raw: unknown): raw is { value: number | boolean } {
  if (typeof raw !== 'object' || raw === null || !('value' in raw)) {
    return false;
  }
  const { value } = raw;
  return typeof value === 'boolean' || (typeof value === 'number' && Number.isFinite(value));
}

export function scoreRows(metadata: ScoreMetadata[]): ScoreRow[] {
  const latest = new Map<string, ScoreRow>();

  for (const md of metadata) {
    for (const [name, raw] of scoreEntries(md)) {
      if (!isScoreValue(raw)) {
        continue;
      }
      const prev = latest.get(name);
      if (!prev || md.updatedAt >= prev.updatedAt) {
        latest.set(name, { name, value: raw.value, updatedAt: md.updatedAt });
      }
    }
  }

  return [...latest.values()].sort((a, b) => a.name.localeCompare(b.name));
}

export const ScoresAttrs = ({ metadata }: { metadata: ScoreMetadata[] }) => {
  const rows = scoreRows(metadata);

  if (rows.length === 0) {
    return (
      <div className="flex h-full items-center justify-center px-4 py-8">
        <p className="text-muted text-center text-sm">No scores recorded</p>
      </div>
    );
  }

  return (
    <div className="relative h-full overflow-y-auto overflow-x-hidden">
      <div className="text-muted bg-canvasSubtle sticky top-0 grid grid-cols-[minmax(10rem,1fr)_8rem_12rem] gap-4 px-4 py-2 text-xs font-medium leading-tight">
        <div>Score</div>
        <div>Value</div>
        <div>Updated at</div>
      </div>
      {rows.map((row) => (
        <div
          key={row.name}
          className="border-muted grid grid-cols-[minmax(10rem,1fr)_8rem_12rem] gap-4 border-b px-4 py-3 text-sm last:border-b-0"
        >
          <div className="text-basis min-w-0 font-medium [overflow-wrap:anywhere]">{row.name}</div>
          <div className="text-basis font-mono">{formatScoreValue(row.value)}</div>
          <TimeElement date={new Date(row.updatedAt)} />
        </div>
      ))}
    </div>
  );
};
