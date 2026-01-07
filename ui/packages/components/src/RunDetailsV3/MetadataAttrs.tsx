import { useLayoutEffect, useRef, useState } from 'react';

import { ElementWrapper, TextElement, TimeElement } from '../DetailsCard/Element';
import type { SpanMetadata, SpanMetadataKind } from './types';

const inngestKindLabels: Record<string, string> = {
  ai: 'AI Metadata',
  http: 'HTTP Metadata',
  warnings: 'Warnings',
};

const getKindLabel = (kind: SpanMetadataKind): string => {
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

const MetadataAttrRow = ({
  kind,
  scope,
  values,
  updatedAt,
  isLast,
}: SpanMetadata & { isLast: boolean }) => {
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
        } mt-2 flex max-h-full flex-col gap-2`}
      >
        <div className="text-muted bg-canvasSubtle sticky top-0 flex flex-row px-4 py-2 text-sm font-medium leading-tight">
          <div className="w-48">Key</div>
          <div className="">Value</div>
        </div>
        {Object.entries(values).map(([key, value]) => {
          return (
            <div key={`metadata-attr-${key}`} className="flex flex-row items-center px-4 pb-2">
              <div className="text-muted w-48 text-sm font-normal leading-tight">{key}</div>
              <div className="text-basis truncate text-sm font-normal leading-tight">
                {String(value) || '--'}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export const MetadataAttrs = ({ metadata }: { metadata: SpanMetadata[] }) => {
  const ref = useRef<HTMLDivElement>(null);
  const [height, setHeight] = useState<number | null>(null);
  useLayoutEffect(() => {
    if (ref.current) {
      ref.current.style.height = `${ref.current.clientHeight}px`;
      setHeight(ref.current.offsetHeight);
    }
  }, [metadata]);

  return (
    <div className="relative h-full overflow-y-auto" ref={ref}>
      {height
        ? metadata.map((md, idx) => {
            const isLast = idx === metadata.length - 1;

            return (
              <MetadataAttrRow
                key={`metadata-attr-${md.scope}-${md.kind}`}
                kind={md.kind}
                scope={md.scope}
                values={md.values}
                updatedAt={md.updatedAt}
                isLast={isLast}
              />
            );
          })
        : null}
    </div>
  );
};
