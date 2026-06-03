import { Select, type Option } from '../Select/Select';

type RunsTypeFilterProps = {
  excludeDeferred: boolean;
  onExcludeDeferredChange: (value: boolean) => void;
};

const options: Option[] = [
  { id: 'all', name: 'All' },
  { id: 'primary', name: 'Primary (no deferred runs)' },
];

export default function RunsTypeFilter({
  excludeDeferred,
  onExcludeDeferredChange,
}: RunsTypeFilterProps) {
  const selectedValue = options.find((o) => o.id === (excludeDeferred ? 'primary' : 'all'));

  return (
    <Select
      value={selectedValue}
      onChange={(value: Option) => {
        onExcludeDeferredChange(value.id === 'primary');
      }}
      label="Run type"
      isLabelVisible={true}
      className="bg-modalBase"
      size="small"
    >
      <Select.Button size="small">
        <span>{selectedValue?.name}</span>
      </Select.Button>
      <Select.Options>
        {options.map((option) => (
          <Select.Option key={option.id} option={option}>
            <span>{option.name}</span>
          </Select.Option>
        ))}
      </Select.Options>
    </Select>
  );
}
