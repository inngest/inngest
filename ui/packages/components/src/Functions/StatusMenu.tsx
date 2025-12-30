import { Select } from '@inngest/components/Select/Select';
import { StatusDot } from '@inngest/components/Status/StatusDot';

export default function FunctionsStatusFilter({
  archived,
  onStatusChange,
}: {
  archived: boolean;
  onStatusChange: (archived: boolean) => void;
}) {
  const activeOption = { id: 'active', name: 'Active functions' };
  const archivedOption = { id: 'archived', name: 'Archived functions' };
  return (
    <Select
      onChange={(value) => onStatusChange(value.id === 'archived')}
      isLabelVisible
      label="Status"
      multiple={false}
      value={archived ? archivedOption : activeOption}
      size="small"
    >
      <Select.Button size="small">
        <div className="flex flex-row items-center gap-2">
          <StatusDot status={archived ? 'ARCHIVED' : 'ACTIVE'} size="small" />
          {archived ? 'Archived' : 'Active'}
        </div>
      </Select.Button>
      <Select.Options>
        <Select.Option key={activeOption.id} option={activeOption}>
          <div className="flex flex-row items-center gap-2">
            <StatusDot status="ACTIVE" size="small" />
            {activeOption.name}
          </div>
        </Select.Option>

        <Select.Option key={archivedOption.id} option={archivedOption}>
          <div className="flex flex-row items-center gap-2">
            <StatusDot status="ARCHIVED" size="small" />
            {archivedOption.name}
          </div>
        </Select.Option>
      </Select.Options>
    </Select>
  );
}
