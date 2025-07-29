'use client';

import { Select } from '@inngest/components/Select/Select';
import { StatusDot } from '@inngest/components/Status/StatusDot';

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
      <Select.Button className="w-[136px]" size="small">
        <div className="text-basis flex flex-row items-center gap-2">
          <StatusDot status={archived ? 'ARCHIVED' : 'ACTIVE'} size="small" />
          {archived ? 'Archived' : 'Active'}
        </div>
      </Select.Button>
      <Select.Options>
        <Select.Option key={activeOption.id} option={activeOption}>
          <div className="text-basis flex flex-row items-center gap-2">
            <StatusDot status="ACTIVE" size="small" />
            {activeOption.name}
          </div>
        </Select.Option>

        <Select.Option key={archivedOption.id} option={archivedOption}>
          <div className="text-basis flex flex-row items-center gap-2">
            <StatusDot status="ARCHIVED" size="small" />
            {archivedOption.name}
          </div>
        </Select.Option>
      </Select.Options>
    </Select>
  );
}
