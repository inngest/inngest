import { useLayoutEffect, useMemo, useRef, type ReactNode } from 'react';
import { RiInformationLine } from '@remixicon/react';

import { InlineCode } from '../Code/InlineCode';
import { CodeBlock } from '../CodeBlock';
import { ElementWrapper, TextElement, TimeElement } from '../DetailsCard/Element';
import { Link } from '../Link';
import { OptionalTooltip } from '../Tooltip/OptionalTooltip';
import type { SpanMetadata, SpanMetadataKind } from './types';

const inngestKindLabels: Record<string, string> = {
  ai: 'AI Metadata',
  experiment: 'Experiment',
  http: 'HTTP Metadata',
  response_headers: 'Response Headers',
  score: 'Scores',
  usage: 'Run Usage',
  warning: 'Warnings',
  warnings: 'Warnings',
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
    return inngestKindLabels[kindName] || `Metadata (${kindName})`;
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
// A `userland.display` entry has one flat `display.<kind>.<key>` key per displayed value. Its
// value is a format ('text' | 'mono' | 'link' | 'code') or `{ format, description }`.
const displayKind = 'userland.display';

type DisplayHint = { format?: string; description?: string };

const getHint = (hints: Record<string, unknown>, kind: string, key: string): DisplayHint => {
  const hint = hints[`display.${kind}.${key}`];
  return typeof hint === 'string' ? { format: hint } : ((hint as DisplayHint | undefined) ?? {});
};

const MetadataValue = ({ text, format }: { text: string; format?: string }) => {
  if (format === 'mono') return <InlineCode>{text}</InlineCode>;
  if (format === 'link' && /^https?:\/\//.test(text)) return <Link href={text}>{text}</Link>;
  if (format === 'code') {
    return (
      <CodeBlock.Wrapper>
        <CodeBlock tab={{ content: text, language: 'shell' }} />
      </CodeBlock.Wrapper>
    );
  }
  return <>{text}</>;
};

type MetadataAttrRowProps = {
  hints: Record<string, unknown>;
  kind: SpanMetadataKind;
  scope: SpanMetadata['scope'];
  values: SpanMetadata['values'];
  updatedAt: string;
  isLast: boolean;
};

const MetadataAttrRow = ({
  hints,
  kind,
  scope,
  values,
  updatedAt,
  isLast,
}: MetadataAttrRowProps) => {
  const sortedEntries = useMemo(
    () =>
      Object.entries(values ?? {}).sort(([a], [b]) => {
        if (a === 'Status Code') return -1;
        if (b === 'Status Code') return 1;
        return a.localeCompare(b);
      }),
    [values],
  );

  return (
    <div className="flex flex-col justify-start gap-2">
      <div className="flex h-11 w-full flex-row items-center justify-between border-none px-4 pt-2">
        <span className="text-basis text-sm font-medium">{getKindLabel(kind)}</span>
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
          const { format, description } = getHint(hints, kind, key);
          return (
            <div
              key={`metadata-attr-${key}`}
              className="flex flex-row items-start overflow-hidden px-4 pb-2"
            >
              <div className="text-muted w-48 shrink-0 text-sm font-normal leading-tight">
                {key}
                {description && (
                  <OptionalTooltip tooltip={description} side="top">
                    <RiInformationLine className="ml-1 inline h-3.5 w-3.5" />
                  </OptionalTooltip>
                )}
              </div>
              <div className="text-basis min-w-0 flex-1 whitespace-pre-wrap text-sm font-normal leading-tight [overflow-wrap:anywhere]">
                <MetadataValue
                  format={format}
                  text={
                    byteValueKeys.has(key) && typeof value === 'number'
                      ? formatBytes(value)
                      : typeof value === 'object' && value !== null
                        ? JSON.stringify(value, null, 2)
                        : String(value) || '--'
                  }
                />
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
  const hints = Object.assign(
    {},
    ...safeMetadata.filter((md) => md.kind === displayKind).map((md) => md.values),
  );
  const shown = safeMetadata.filter((md) => md.kind !== displayKind);

  return (
    <div className="relative h-full overflow-y-auto overflow-x-hidden" ref={ref}>
      {shown.map((md, idx) => {
        const isLast = idx === shown.length - 1;

        return (
          <MetadataAttrRow
            hints={hints}
            key={`metadata-attr-${md.scope}-${md.kind}`}
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
