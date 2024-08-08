'use client';

import Link from 'next/link';
import { Select } from '@inngest/components/Select/Select';

const StatusIcon = ({ className }: { className: string }) => (
  <span className={`block h-2 w-2 shrink-0 rounded-full ${className}`} />
);

export const StatusMenu = ({ envSlug, archived }: { envSlug: string; archived: boolean }) => {
  const activeOption = { id: 'active', name: 'Active apps' };
  const archivedOption = { id: 'archived', name: 'Archived apps' };
  return (
    <Select
      onChange={() => null}
      isLabelVisible={false}
      label="Pause runs"
      multiple={false}
      value={archived ? archivedOption : activeOption}
      className="mb-5"
    >
      <Select.Button className="w-[124px] px-4">
        <div className="text-basis mr-2 flex flex-row items-center text-sm font-medium leading-tight">
          <StatusIcon className={`mr-2 ${archived ? 'bg-accent-subtle' : 'bg-primary-moderate'}`} />
          {archived ? 'Archived' : 'Active'}
        </div>
      </Select.Button>
      <Select.Options>
        <Link href={`/env/${envSlug}/apps`} prefetch={true}>
          <Select.Option key={activeOption.id} option={activeOption}>
            <div className="text-basis flex flex-row items-center text-sm font-medium">
              <StatusIcon className="bg-primary-moderate mr-2" />
              {activeOption.name}
            </div>
          </Select.Option>
        </Link>
        <Link href={`/env/${envSlug}/apps?archived=true`} prefetch={true}>
          <Select.Option key={archivedOption.id} option={archivedOption}>
            <div className="text-basis flex flex-row items-center text-sm font-medium">
              <StatusIcon className="bg-accent-subtle mr-2" />
              {archivedOption.name}
            </div>
          </Select.Option>
        </Link>
      </Select.Options>
    </Select>
  );
};
