'use client';

import Link from 'next/link';
import { Select } from '@inngest/components/Select/Select';

export const ActionsMenu = ({ envSlug, archived }: { envSlug: string; archived: boolean }) => {
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
        <div className="text-basis mr-2 flex flex-row items-center text-xs font-medium leading-tight">
          All actions
        </div>
      </Select.Button>
      <Select.Options>
        <Link href={`/env/${envSlug}/apps`} prefetch={true}>
          <Select.Option key={activeOption.id} option={activeOption}>
            <div className="text-basis flex flex-row items-center text-xs font-medium">
              {activeOption.name}
            </div>
          </Select.Option>
        </Link>
        <Link href={`/env/${envSlug}/apps?archived=true`} prefetch={true}>
          <Select.Option key={archivedOption.id} option={archivedOption}>
            <div className="text-basis flex flex-row items-center text-xs font-medium">
              {archivedOption.name}
            </div>
          </Select.Option>
        </Link>
      </Select.Options>
    </Select>
  );
};
