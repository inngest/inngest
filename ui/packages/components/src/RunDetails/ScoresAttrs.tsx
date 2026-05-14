import { TimeElement } from '../DetailsCard/Element';

type ScoreMetadata = {
  updatedAt: string;
  values: Record<string, unknown>;
};

type ScoreRow = {
  name: string;
  updatedAt: string;
  value: number;
};

type ScoreTrace = {
  metadata?: Array<ScoreMetadata & { kind: string }>;
  childrenSpans?: ScoreTrace[];
};

export function collectScoreMetadata(trace?: ScoreTrace): ScoreMetadata[] {
  // Run views need child spans because scores attach where they are emitted.
  const metadata = trace?.metadata?.filter((md) => md.kind === 'inngest.score') ?? [];
  const childMetadata = trace?.childrenSpans?.flatMap((child) => collectScoreMetadata(child)) ?? [];

  return [...metadata, ...childMetadata];
}

function scoreRows(metadata: ScoreMetadata[]): ScoreRow[] {
  return metadata
    .flatMap((md) =>
      Object.entries(md.values)
        .filter((entry): entry is [string, number] => {
          const [, value] = entry;
          return typeof value === 'number' && Number.isFinite(value);
        })
        .map(([name, value]) => ({
          name,
          value,
          updatedAt: md.updatedAt,
        }))
    )
    .sort((a, b) => a.name.localeCompare(b.name) || a.updatedAt.localeCompare(b.updatedAt));
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
      <div className="text-muted bg-canvasSubtle sticky top-0 grid grid-cols-[minmax(10rem,1fr)_8rem_12rem] gap-4 px-4 py-2 text-sm font-medium leading-tight">
        <div>Score</div>
        <div>Value</div>
        <div>Updated at</div>
      </div>
      {rows.map((row, index) => (
        <div
          key={`score-${row.name}-${row.updatedAt}-${index}`}
          className="border-muted grid grid-cols-[minmax(10rem,1fr)_8rem_12rem] gap-4 border-b px-4 py-3 text-sm last:border-b-0"
        >
          <div className="text-basis min-w-0 font-medium [overflow-wrap:anywhere]">{row.name}</div>
          <div className="text-basis font-mono">{row.value}</div>
          <TimeElement date={new Date(row.updatedAt)} />
        </div>
      ))}
    </div>
  );
};
