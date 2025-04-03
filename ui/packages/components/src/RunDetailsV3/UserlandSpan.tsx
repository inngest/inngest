import type { UserlandSpanType } from './types';

const internalPrevixes = ['sys', 'inngest', 'userland', 'sdk'];

export const UserlandSpan = ({ userlandSpan }: { userlandSpan: UserlandSpanType }) => {
  let attrs = null;

  try {
    attrs = JSON.parse(userlandSpan.spanAttrs);
  } catch (error) {
    console.info('Error parsing userlandAttrs', error);
  }

  return attrs ? (
    <div className="border-accent-intense mt-2 flex flex-col border-t text-sm font-medium leading-tight">
      {Object.entries(attrs)
        .filter(([key]) => !internalPrevixes.some((prefix) => key.startsWith(prefix)))
        .map(([key, value], i) => {
          return (
            <div
              key={`userland-span-${i}`}
              className="text-muted mt-2 flex flex-row items-center justify-start gap-2 text-xs"
            >
              <div className="text-muted text-xs">{key}:</div>
              <div className="truncate">{String(value)}</div>
            </div>
          );
        })}
    </div>
  ) : null;
};
