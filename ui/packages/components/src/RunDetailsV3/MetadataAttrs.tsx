import { useRef, useState } from 'react';
import { RiArrowRightSLine } from '@remixicon/react';

import { ElementWrapper, TextElement } from '../DetailsCard/NewElement';
import type { SpanMetadata } from './types';

const MetadataAttrRow = ({ kind, scope, values }: SpanMetadata) => {
  const [expanded, setExpanded] = useState(true);
  return (
    <div className="flex flex-col justify-start gap-2 overflow-hidden">
      <div className="flex h-11 w-full flex-row items-center justify-between border-none px-4 pt-2">
        <div
          className="text-basis flex cursor-pointer items-center justify-start gap-2"
          onClick={() => setExpanded(!expanded)}
        >
          <RiArrowRightSLine
            className={`shrink-0 transition-transform duration-[250ms] ${
              expanded ? 'rotate-90' : ''
            }`}
          />
          <span className="text-basis text-sm font-normal">{`${scope}/${kind}`}</span>
        </div>
      </div>
      {expanded ? (
        <>
          <div className="flex flex-row flex-wrap items-center justify-start gap-x-10 gap-y-4 px-4">
            <ElementWrapper label="Scope">
              <TextElement>{scope}</TextElement>
            </ElementWrapper>
            <ElementWrapper label="Kind">
              <TextElement>{kind}</TextElement>
            </ElementWrapper>
            {/* TODO: updated timestamp */}
          </div>
          <div className="mb-4 mt-2 flex max-h-full flex-col gap-2">
            <div className="text-muted bg-canvasSubtle sticky top-0 flex flex-row px-4 py-2 text-sm font-medium leading-tight">
              <div className="w-72">Key</div>
              <div className="">Value</div>
            </div>
            {Object.entries(values).map(([key, value]) => {
              return (
                <div
                  key={`metadata-attr-${key}`}
                  className="border-canvasSubtle flex flex-row items-center border-b px-4 pb-2"
                >
                  <div className="text-muted w-72 text-sm font-normal leading-tight">{key}</div>
                  <div className="text-basis truncate text-sm font-normal leading-tight">
                    {String(value) || '--'}
                  </div>
                </div>
              );
            })}
          </div>
        </>
      ) : null}
    </div>
  );
};

export const MetadataAttrs = ({ metadata }: { metadata: SpanMetadata[] }) => {
  return (
    <div className="h-full overflow-y-auto">
      {metadata.map((md) => (
        <MetadataAttrRow
          key={`metadata-attr-${md.scope}-${md.kind}`}
          kind={md.kind}
          scope={md.scope}
          values={md.values}
        />
      ))}
    </div>
  );
};
