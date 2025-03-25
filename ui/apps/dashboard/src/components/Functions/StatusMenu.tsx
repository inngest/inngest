'use client';

import NextLink from 'next/link';
import { Select } from '@inngest/components/Select/Select';

const StatusIcon = ({ className }: { className: string }) => (
  <span className={`block h-2 w-2 shrink-0 rounded-full ${className}`} />
);

export const StatusMenu = ({ envSlug, archived }: { envSlug: string; archived: boolean }) => {
  const activeOption = { id: 'active', name: 'Active functions' };
  const archivedOption = { id: 'archived', name: 'Archived functions' };
  return (
    <Select
      onChange={() => null}
      isLabelVisible={false}
      label="Pause runs"
      multiple={false}
      value={archived ? archivedOption : activeOption}
      className="z-20 mr-3 h-[30px]"
    >
      <Select.Button className="h-[28px] w-[142px] py-1 pl-2 pr-3">
        <div className="mr-2 flex flex-row items-center text-sm">
          <StatusIcon className={`mr-2 ${archived ? 'bg-surfaceMuted' : 'bg-primary-moderate'}`} />
          {archived ? 'Archived' : 'Active'}
        </div>
      </Select.Button>

      <Select.Options>
        <NextLink href={`/env/${envSlug}/functions`}>
          <Select.Option key={activeOption.id} option={activeOption}>
            <div className="text-basis flex flex-row items-center text-sm font-medium">
              <StatusIcon className="bg-primary-moderate mr-2" />
              {activeOption.name}
            </div>
          </Select.Option>
        </NextLink>
        <NextLink href={`/env/${envSlug}/functions?archived=true`}>
          <Select.Option key={archivedOption.id} option={archivedOption}>
            <div className="text-basis flex flex-row items-center text-sm font-medium">
              <StatusIcon className="bg-surfaceMuted mr-2" />
              {archivedOption.name}
            </div>
          </Select.Option>
        </NextLink>
      </Select.Options>
    </Select>
  );
};
