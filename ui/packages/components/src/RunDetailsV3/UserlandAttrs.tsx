import { ElementWrapper, TextElement } from '../DetailsCard/NewElement';
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
    <div className="flex flex-col gap-2 px-4 py-2">
      {Object.entries(attrs)
        .filter(([key]) => !internalPrevixes.some((prefix) => key.startsWith(prefix)))
        .map(([key, value], i) => {
          return (
            <div className="flex flex-row items-center justify-start gap-2">
              <div className="text-muted text-xs">{key}:</div>
              <div className="truncate">{String(value) || '--'}</div>
            </div>
          );
        })}
    </div>
  ) : null;
};
