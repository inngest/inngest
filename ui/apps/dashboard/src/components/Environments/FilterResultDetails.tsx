'use client';

type FilterResultDetailsProps = {
  hasFilter: boolean;
  size: number;
};

export function FilterResultDetails({ hasFilter, size }: FilterResultDetailsProps) {
  return (
    <div className="flex items-center px-3 py-2 max-[625px]:hidden">
      <span className="text-light whitespace-nowrap text-sm">
        {size} {hasFilter ? 'Filtered' : 'Total'} {size === 1 ? 'environment' : 'environments'}
      </span>
    </div>
  );
}
