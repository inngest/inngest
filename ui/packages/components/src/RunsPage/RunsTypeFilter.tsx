import { Select, type Option } from '../Select/Select';
import { isRunType, runTypes, type RunType } from '../types/functionRun';

type RunsTypeFilterProps = {
  selectedRunType: RunType | undefined;
  onRunTypeChange: (value: RunType | undefined) => void;
};

const ALL_ID = 'ALL';

const options: Option[] = [
  { id: ALL_ID, name: 'ALL' },
  ...runTypes.map((field) => ({ id: field, name: field })),
];

export default function RunsTypeFilter({ selectedRunType, onRunTypeChange }: RunsTypeFilterProps) {
  const selectedValue =
    options.find((option) => option.id === (selectedRunType ?? ALL_ID)) ?? options[0];

  return (
    <Select
      value={selectedValue}
      onChange={(value: Option) => {
        onRunTypeChange(isRunType(value.id) ? value.id : undefined);
      }}
      label="Type"
      isLabelVisible
      className="bg-modalBase min-w-[90px]"
      size="small"
    >
      <Select.Button isLabelVisible size="small">
        <span className="lowercase first-letter:capitalize">{selectedValue?.name}</span>
      </Select.Button>
      <Select.Options>
        {options.map((option) => {
          return (
            <Select.Option key={option.id} option={option}>
              <span className="inline-flex items-center gap-2 lowercase">
                <label className="first-letter:capitalize">{option.name}</label>
              </span>
            </Select.Option>
          );
        })}
      </Select.Options>
    </Select>
  );
}
