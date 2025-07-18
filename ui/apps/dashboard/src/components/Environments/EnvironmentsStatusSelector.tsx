'use client';

import { Select } from '@inngest/components/Select/Select';

type EnvironmentsStatusSelectorProps = {
  archived: boolean;
  onChange: (archived: boolean) => void;
};

const ACTIVE_OPTION = { id: 'active', name: 'Active environments' };
const ARCHIVED_OPTION = { id: 'archived', name: 'Archived environments' };

export function EnvironmentsStatusSelector({
  archived,
  onChange,
}: EnvironmentsStatusSelectorProps) {
  return (
    <Select
      onChange={(value) => onChange(value.id === 'archived')}
      isLabelVisible={false}
      label="Select environment status"
      multiple={false}
      value={archived ? ARCHIVED_OPTION : ACTIVE_OPTION}
    >
      <Select.Button className="h-[28px] w-[200px] shrink-0 px-2 py-1">
        <div className="text-basis mr-1 flex flex-row items-center overflow-hidden whitespace-nowrap text-sm font-medium leading-tight">
          <StatusIcon className={`mr-2 ${archived ? 'bg-surfaceMuted' : 'bg-primary-moderate'}`} />
          {archived ? 'Archived environments' : 'Active environments'}
        </div>
      </Select.Button>
      <Select.Options>
        <Select.Option key={ACTIVE_OPTION.id} option={ACTIVE_OPTION}>
          <div className="text-basis flex flex-row items-center text-sm font-medium">
            <StatusIcon className="bg-primary-moderate mr-1" />
            {ACTIVE_OPTION.name}
          </div>
        </Select.Option>

        <Select.Option key={ARCHIVED_OPTION.id} option={ARCHIVED_OPTION}>
          <div className="text-basis flex flex-row items-center text-sm font-medium">
            <StatusIcon className="bg-surfaceMuted mr-1" />
            {ARCHIVED_OPTION.name}
          </div>
        </Select.Option>
      </Select.Options>
    </Select>
  );
}

function StatusIcon({ className }: { className: string }) {
  return <span className={`block h-2 w-2 shrink-0 rounded-full ${className}`} />;
}
