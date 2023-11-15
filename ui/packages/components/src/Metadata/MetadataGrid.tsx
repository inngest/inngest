import { classNames } from '@inngest/components/utils/classNames';

import { MetadataItem, type MetadataItemProps } from './MetadataItem';

type Props = {
  metadataItems: MetadataItemProps[];
  columns?: 2 | 3;
};

export function MetadataGrid({ metadataItems, columns = 3 }: Props) {
  // Each metadata element that has a large size counts as two items
  const items = metadataItems.reduce((count, item) => {
    return count + (item.size === 'large' ? 2 : 1);
  }, 0);

  let gridColumns: number;

  if (items > 2 && columns === 3) {
    gridColumns = 3;
  } else if (items >= 2) {
    gridColumns = 2;
  } else {
    gridColumns = 1;
  }

  const rows = Math.ceil(items / columns);
  let currentIndex = 0;

  return (
    <dl
      className={`dark:bg-slate-910 grid rounded-lg border border-slate-200 bg-white p-2.5 dark:border-slate-800 grid-cols-${gridColumns} grid-rows-${rows} gap-5`}
    >
      {metadataItems.map((item, index) => {
        const spanIndex = currentIndex;
        if (item.size === 'large') {
          currentIndex += 2; // Increment by 2 for large items.
        } else {
          currentIndex += 1; // Increment by 1 for regular items.
        }
        // Check conditions to exclude the first items if there is only one row and any items in the last row.
        const lastOrOnlyRow =
          (metadataItems.length <= gridColumns &&
            currentIndex - spanIndex < metadataItems.length) ||
          spanIndex >= metadataItems.length - (metadataItems.length % gridColumns);
        const verticalDividers =
          'before:absolute before:top-0 before:-left-2.5 before:h-full before:border-l dark:before:border-slate-800/50 before:border-slate-200';
        const horizontalDividers =
          'after:absolute after:-bottom-2.5 after:left-0 after:w-full dark:after:border-slate-800/50 after:border-slate-200 after:border-b';

        return (
          <MetadataItem
            key={index}
            className={classNames(
              'relative overflow-visible',
              spanIndex !== 0 && spanIndex % gridColumns !== 0 && verticalDividers,
              !lastOrOnlyRow && horizontalDividers,
              item.size === 'large' ? 'col-span-2' : 'col-span-1'
            )}
            {...item}
          />
        );
      })}
    </dl>
  );
}
