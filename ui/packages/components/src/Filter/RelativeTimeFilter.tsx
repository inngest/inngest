import { Badge } from '../Badge';
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
      className="w-[7.5rem]"
    >
      <Select.Button>
        <span className="pr-2 text-sm lowercase first-letter:capitalize">{selectedDays?.name}</span>
      </Select.Button>
      <Select.Options>
        {options.map((option) => {
          return (
            <Select.Option key={option.id} option={option}>
              <span className="inline-flex w-full items-center justify-between gap-2">
                <label className="text-sm lowercase first-letter:capitalize">{option.name}</label>
                {option.disabled && (
                  <Badge className="border-indigo-500 text-indigo-500" kind="outlined">
                    Upgrade Plan
                  </Badge>
                )}
              </span>
            </Select.Option>
          );
        })}
      </Select.Options>
    </Select>
  );
}
