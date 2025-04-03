import { ElementWrapper, TextElement } from '../DetailsCard/NewElement';
import type { UserlandSpanType } from './types';

const internalPrevixes = ['sys', 'inngest', 'userland', 'sdk'];

export const UserlandAttrs = ({ userlandSpan }: { userlandSpan: UserlandSpanType }) => {
  let attrs = null;

  try {
    attrs = JSON.parse(userlandSpan.spanAttrs);
  } catch (error) {
    console.info('Error parsing userland span attributes', error);
  }

  return attrs ? (
    <>
      {Object.entries(attrs)
        .filter(([key]) => !internalPrevixes.some((prefix) => key.startsWith(prefix)))
        .map(([key, value], i) => {
          return (
            <ElementWrapper label={key}>
              <TextElement>{String(value)}</TextElement>
            </ElementWrapper>
          );
        })}
    </>
  ) : null;
};
