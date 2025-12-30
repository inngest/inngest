import type { UserlandSpanType } from './types';

const internalPrevixes = ['sys', 'inngest', 'userland', 'sdk'];

export const UserlandAttrs = ({ userlandSpan }: { userlandSpan: UserlandSpanType }) => {
  let attrs = null;

  try {
    attrs = userlandSpan.spanAttrs && JSON.parse(userlandSpan.spanAttrs);
  } catch (error) {
    console.info('Error parsing userland span attributes', error);
  }

  return attrs ? (
    <div className="h-full overflow-y-auto">
      <div className="mb-4 mt-2 flex max-h-full flex-col gap-2">
        <div className="text-muted bg-canvasSubtle sticky top-0 flex flex-row px-4 py-2 text-sm font-medium leading-tight">
          <div className="w-72">Key</div>
          <div className="">Value</div>
        </div>
        {Object.entries(attrs)
          .filter(([key]) => !internalPrevixes.some((prefix) => key.startsWith(prefix)))
          .map(([key, value]) => {
            return (
              <div
                key={`userland-span-attr-${key}`}
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
    </div>
  ) : null;
};
