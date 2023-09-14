import classNames from '@/utils/classnames';
import MetadataItem, { type MetadataItemProps } from './MetadataItem';

type Props = {
  metadataItems: MetadataItemProps[];
};

export default function MetadataGrid({ metadataItems }: Props) {
  // Each metadata element that has a large size counts as two items
  const items = metadataItems.reduce((count, item) => {
    return count + (item.size === 'large' ? 2 : 1);
  }, 0);

  const columns = items > 2 ? 3 : items === 2 ? 2 : 1;

  const rows = Math.ceil(items / 3);
  let currentIndex = 0;

  return (
    <div className="p-2.5 border rounded-lg border-slate-800/50 bg-slate-950">
      <div className={`grid grid-cols-${columns} grid-rows-${rows} gap-5`}>
        {metadataItems.map((item, index) => {
          const spanIndex = currentIndex;
          if (item.size === 'large') {
            currentIndex += 2; // Increment by 2 for large items.
          } else {
            currentIndex += 1; // Increment by 1 for regular items.
          }
          // Check conditions to exclude the first 3 items and any items in the last row.
          const excludeCondition = spanIndex < metadataItems.length - (metadataItems.length % 3);
          const verticalDividers =
            'before:absolute before:top-0 before:-left-2.5 before:h-full before:border-l before:border-slate-800/50';
          const horizontalDividers =
            'after:absolute after:-bottom-2.5 after:left-0 after:w-full after:border-slate-800/50 after:border-b';

          return (
            <span
              key={index}
              className={classNames(
                'relative overflow-visible',
                spanIndex !== 0 && spanIndex % 3 !== 0 && verticalDividers,
                excludeCondition && horizontalDividers,
                item.size === 'large' ? 'col-span-2' : 'col-span-1',
              )}
            >
              <MetadataItem {...item} />
            </span>
          );
        })}
      </div>
    </div>
  );
}
