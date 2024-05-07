import { Select, type Option } from '../Select/Select';

type RelativeTimeFilterProps = {
  options: Option[];
  selectedDays?: Option;
  onDaysChange: (value: Option) => void;
};

export default function RelativeTimeFilter({
  selectedDays,
  onDaysChange,
  options,
}: RelativeTimeFilterProps) {
  return (
    <Select
      defaultValue={selectedDays}
      onChange={onDaysChange}
      label="Last Days"
      isLabelVisible={false}
    >
      <Select.Button>
        <span className="pr-2 text-sm lowercase first-letter:capitalize">{selectedDays?.name}</span>
      </Select.Button>
      <Select.Options>
        {options.map((option) => {
          return (
            <Select.Option key={option.id} option={option}>
              <span className="inline-flex items-center gap-2 lowercase">
                <label className="text-sm first-letter:capitalize">{option.name}</label>
                {option.disabled && 'Upgrade Plan'}
              </span>
            </Select.Option>
          );
        })}
      </Select.Options>
    </Select>
  );
}
