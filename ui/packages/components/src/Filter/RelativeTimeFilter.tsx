import { Select } from '../Select/Select';

const datesArray = [1, 3, 7, 14, 30];

type RelativeTimeFilterProps = {
  selectedDays: number;
  onDaysChange: (value: number) => void;
};

export default function RelativeTimeFilter({
  selectedDays,
  onDaysChange,
}: RelativeTimeFilterProps) {
  return (
    <Select
      defaultValue={selectedDays}
      //@ts-ignore
      onChange={onDaysChange}
      label="Last Days"
      isLabelVisible={false}
    >
      <Select.Button>
        <span className="pr-2 text-sm lowercase first-letter:capitalize">
          {`Last ${selectedDays} days`}
        </span>
      </Select.Button>
      <Select.Options>
        {datesArray.map((option) => {
          return (
            <Select.Option key={option} option={option}>
              <span className="inline-flex items-center gap-2 lowercase">
                <label className="text-sm first-letter:capitalize">{`Last ${option} days`}</label>
              </span>
            </Select.Option>
          );
        })}
      </Select.Options>
    </Select>
  );
}
