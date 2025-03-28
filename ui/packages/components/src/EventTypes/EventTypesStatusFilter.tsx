'use client';

import { Select } from '@inngest/components/Select/Select';

const StatusIcon = ({ className }: { className: string }) => (
  <span className={`block h-2 w-2 shrink-0 rounded-full ${className}`} />
);

export default function EventTypesStatusFilter({
  archived,
  onStatusChange,
}: {
  pathCreator: string;
  archived: boolean;
  onStatusChange: (archived: boolean) => void;
}) {
  const activeOption = { id: 'active', name: 'Active events' };
  const archivedOption = { id: 'archived', name: 'Archived events' };
  return (
    <Select
      onChange={(value) => onStatusChange(value.id === 'archived')}
      isLabelVisible={false}
      label="Select event status"
      multiple={false}
      value={archived ? archivedOption : activeOption}
    >
      <Select.Button className="h-[28px] w-[136px] px-2 py-1">
        <div className="text-basis mr-1 flex flex-row items-center text-xs font-medium leading-tight">
          <StatusIcon className={`mr-1 ${archived ? 'bg-surfaceMuted' : 'bg-primary-moderate'}`} />
          {archived ? 'Archived events' : 'Active events'}
        </div>
      </Select.Button>
      <Select.Options>
        <Select.Option key={activeOption.id} option={activeOption}>
          <div className="text-basis flex flex-row items-center text-xs font-medium">
            <StatusIcon className="bg-primary-moderate mr-1" />
            {activeOption.name}
          </div>
        </Select.Option>

        <Select.Option key={archivedOption.id} option={archivedOption}>
          <div className="text-basis flex flex-row items-center text-xs font-medium">
            <StatusIcon className="bg-surfaceMuted mr-1" />
            {archivedOption.name}
          </div>
        </Select.Option>
      </Select.Options>
    </Select>
  );
}
