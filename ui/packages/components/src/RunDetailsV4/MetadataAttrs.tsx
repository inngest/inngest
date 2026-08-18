import { useLayoutEffect, useMemo, useRef, type ReactNode } from 'react';

import { ElementWrapper, TextElement, TimeElement } from '../DetailsCard/Element';
import type { SpanMetadata, SpanMetadataKind } from './types';

// Keyed by full kind, not by the first suffix segment: `inngest.ai` and
// `inngest.ai.summary` (and `inngest.http` / `inngest.http.timing`) are
// distinct kinds that would otherwise collide onto one label.
const inngestKindLabels: Record<string, string> = {
  'inngest.ai': 'AI Metadata',
  'inngest.ai.summary': 'AI Summary',
  'inngest.experiment': 'Experiment',
  'inngest.http': 'HTTP Metadata',
  'inngest.http.timing': 'HTTP Timing',
  'inngest.response_headers': 'Response Headers',
  'inngest.score': 'Scores',
  'inngest.usage': 'Run Usage',
  'inngest.warnings': 'Warnings',
};

const formatBytes = (bytes: number): string => {
  if (bytes < 1024) return `${bytes} B`;
  return `${(bytes / 1024).toFixed(1)} KB`;
};

const byteValueKeys = new Set(['metadata_bytes']);

/** Returns a human-readable label for a metadata kind. Handles both inngest.* and userland.* kinds. */
const getKindLabel = (kind: SpanMetadataKind): string => {
  if (!kind) return 'Unknown Metadata';
  const [namespace, kindName] = kind.split('.');
  if (!kindName) {
    return `Unknown Metadata (kind: ${kind})`;
  }

  if (namespace === 'inngest') {
    return inngestKindLabels[kind] || `Metadata (${kindName})`;
  }

  if (kindName === 'default') {
    return `User Metadata`;
  }

  return `User Metadata (${kindName})`;
};

// Row props are intentionally decoupled from the SpanMetadata discriminated
// union: the row renders values generically via Object.entries and never relies
// on the kind/values correlation. Indexing the union keeps it permissive enough
// to accept any arm's values (including generated interface types like
// AIMetadata, which lack an implicit index signature).
type MetadataAttrRowProps = {
  kind: SpanMetadataKind;
  scope: SpanMetadata['scope'];
  values: SpanMetadata['values'];
  updatedAt: string;
  isLast: boolean;
};

const MetadataAttrRow = ({ kind, scope, values, updatedAt, isLast }: MetadataAttrRowProps) => {
  // A partial AI summary is known-incomplete (e.g. usage from invoked child
  // runs is not folded in), so the totals must not read as authoritative.
  const isPartialSummary =
    kind === 'inngest.ai.summary' && Boolean((values as { partial?: boolean } | null)?.partial);

  const sortedEntries = useMemo(
    () =>
      Object.entries(values ?? {}).sort(([a], [b]) => {
        if (a === 'Status Code') return -1;
        if (b === 'Status Code') return 1;
        return a.localeCompare(b);
      }),
    [values]
  );

  return (
    <div className="flex flex-col justify-start gap-2">
      <div className="flex h-11 w-full flex-row items-center justify-between border-none px-4 pt-2">
        <span className="text-basis text-sm font-medium">
          {getKindLabel(kind)}
          {isPartialSummary && (
            <span className="text-muted ml-2 text-xs font-normal">
              partial — may exclude invoked child run usage
            </span>
          )}
        </span>
      </div>
      <div className="flex flex-row flex-wrap items-center justify-start gap-x-10 gap-y-4 px-4">
        <ElementWrapper label="Metadata Kind">
          <TextElement>{kind}</TextElement>
        </ElementWrapper>
        <ElementWrapper label="Metadata Scope">
          <TextElement>{scope}</TextElement>
        </ElementWrapper>
        <ElementWrapper label="Updated at">
          <TimeElement date={new Date(updatedAt)} />
        </ElementWrapper>
      </div>
      <div
        className={`${
          isLast ? '' : 'border-muted border-b pb-4'
        } mt-2 flex max-h-full flex-col gap-2 overflow-hidden`}
      >
        <div className="text-muted bg-canvasSubtle sticky top-0 flex flex-row px-4 py-2 text-xs font-medium leading-tight">
          <div className="w-48">Key</div>
          <div className="">Value</div>
        </div>
        {sortedEntries.map(([key, value]) => {
          return (
            <div
              key={`metadata-attr-${key}`}
              className="flex flex-row items-start overflow-hidden px-4 pb-2"
            >
              <div className="text-muted w-48 shrink-0 text-sm font-normal leading-tight">
                {key}
              </div>
              <div className="text-basis min-w-0 whitespace-pre-wrap text-sm font-normal leading-tight [overflow-wrap:anywhere]">
                {byteValueKeys.has(key) && typeof value === 'number'
                  ? formatBytes(value)
                  : typeof value === 'object' && value !== null
                  ? JSON.stringify(value, null, 2)
                  : String(value) || '--'}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

/** Displays metadata attributes as key-value pairs. Status Code is pinned to the top, remaining entries sorted alphabetically. */
export const MetadataAttrs = ({
  metadata,
  footer,
}: {
  metadata: SpanMetadata[];
  footer?: ReactNode;
}) => {
  const ref = useRef<HTMLDivElement>(null);
  useLayoutEffect(() => {
    if (ref.current && ref.current.clientHeight > 0) {
      ref.current.style.height = `${ref.current.clientHeight}px`;
    }
  }, [metadata]);

  const safeMetadata = metadata.filter((md) => md && typeof md === 'object' && md.kind);

  return (
    <div className="relative h-full overflow-y-auto overflow-x-hidden" ref={ref}>
      {safeMetadata.map((md, idx) => {
        const isLast = idx === safeMetadata.length - 1;

        return (
          <MetadataAttrRow
            key={`metadata-attr-${idx}-${md.scope}-${md.kind}`}
            kind={md.kind}
            scope={md.scope}
            values={md.values}
            updatedAt={md.updatedAt}
            isLast={isLast}
          />
        );
      })}
      {footer}
    </div>
  );
};
