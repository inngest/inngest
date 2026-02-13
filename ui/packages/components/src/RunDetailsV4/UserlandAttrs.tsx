import type { UserlandSpanType } from './types';

const internalPrefixes = ['sys', 'inngest', 'userland', 'sdk'];

function safeParseJson(jsonStr: string | null): Record<string, unknown> | null {
  if (!jsonStr) return null;
  try {
    return JSON.parse(jsonStr);
  } catch (error) {
    console.info('Error parsing JSON attributes', error);
    return null;
  }
}

function filterEntries(obj: Record<string, unknown>): [string, unknown][] {
  return Object.entries(obj).filter(
    ([key]) => !internalPrefixes.some((prefix) => key.startsWith(prefix))
  );
}

function AttrTable({
  entries,
  keyPrefix,
  testId,
}: {
  entries: [string, unknown][];
  keyPrefix: string;
  testId?: string;
}) {
  if (entries.length === 0) return null;
  return (
    <div data-testid={testId} className="mb-4 mt-2 flex max-h-full flex-col gap-2">
      <div className="text-muted bg-canvasSubtle sticky top-0 flex flex-row px-4 py-2 text-sm font-medium leading-tight">
        <div className="w-72">Key</div>
        <div className="">Value</div>
      </div>
      {entries.map(([key, value]) => (
        <div
          key={`${keyPrefix}-${key}`}
          className="border-canvasSubtle flex flex-row items-center border-b px-4 pb-2"
        >
          <div className="text-muted w-72 text-sm font-normal leading-tight">{key}</div>
          <div className="text-basis truncate text-sm font-normal leading-tight">
            {String(value) || '--'}
          </div>
        </div>
      ))}
    </div>
  );
}

export const UserlandAttrs = ({ userlandSpan }: { userlandSpan: UserlandSpanType }) => {
  const spanAttrs = safeParseJson(userlandSpan.spanAttrs);
  const resourceAttrs = safeParseJson(userlandSpan.resourceAttrs);

  const filteredSpanAttrs = spanAttrs ? filterEntries(spanAttrs) : [];
  const filteredResourceAttrs = resourceAttrs ? filterEntries(resourceAttrs) : [];

  const { spanName, spanKind, serviceName, scopeName, scopeVersion } = userlandSpan;
  const hasMetadata = !!(spanName || spanKind || serviceName || scopeName || scopeVersion);
  const hasContent =
    hasMetadata || filteredSpanAttrs.length > 0 || filteredResourceAttrs.length > 0;

  if (!hasContent) return null;

  return (
    <div className="h-full overflow-y-auto">
      {hasMetadata && (
        <div
          data-testid="userland-metadata-header"
          className="flex flex-row flex-wrap gap-x-10 gap-y-2 px-4 py-2"
        >
          {spanName && (
            <div className="text-sm">
              <dt className="text-muted text-xs">Span</dt>
              <dd className="text-basis">{spanName}</dd>
            </div>
          )}
          {spanKind && (
            <div className="text-sm">
              <dt className="text-muted text-xs">Kind</dt>
              <dd>
                <span className="bg-canvasMuted text-muted rounded-full px-2 py-0.5 text-xs font-medium">
                  {spanKind}
                </span>
              </dd>
            </div>
          )}
          {serviceName && (
            <div className="text-sm">
              <dt className="text-muted text-xs">Service</dt>
              <dd className="text-basis">{serviceName}</dd>
            </div>
          )}
          {scopeName && (
            <div className="text-sm">
              <dt className="text-muted text-xs">Scope</dt>
              <dd className="text-basis">{scopeName}</dd>
            </div>
          )}
          {scopeVersion && (
            <div className="text-sm">
              <dt className="text-muted text-xs">Version</dt>
              <dd className="text-basis">{scopeVersion}</dd>
            </div>
          )}
        </div>
      )}

      <AttrTable entries={filteredSpanAttrs} keyPrefix="userland-span-attr" />
      <AttrTable
        entries={filteredResourceAttrs}
        keyPrefix="userland-resource-attr"
        testId="userland-resource-attrs-section"
      />
    </div>
  );
};
