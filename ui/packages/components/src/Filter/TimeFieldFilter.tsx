import { Select, type Option } from '../Select/Select';
import {
  FunctionRunTimeFields,
  isFunctionTimeField,
  type FunctionRunTimeField,
} from '../types/functionRun';

type TimeFieldFilterProps = {
  selectedTimeField: FunctionRunTimeField;
  onTimeFieldChange: (value: FunctionRunTimeField) => void;
};

const options: Option[] = FunctionRunTimeFields.map((field) => ({
  id: field,
  name: field.replace(/_/g, ' '),
}));

export default function TimeFieldFilter({
  selectedTimeField,
  onTimeFieldChange,
}: TimeFieldFilterProps) {
  const selectedValue = options.find((option) => option.id === selectedTimeField);

  return (
    <Select
      defaultValue={selectedValue}
      onChange={(value: Option) => {
        if (isFunctionTimeField(value.id)) {
          onTimeFieldChange(value.id);
        }
      }}
      label="Time Field"
      isLabelVisible={false}
      className="w-[6.6rem]"
    >
      <Select.Button>
        <span className="pr-2 text-sm lowercase first-letter:capitalize">
          {selectedValue?.name}
        </span>
      </Select.Button>
      <Select.Options>
        {options.map((option) => {
          return (
            <Select.Option key={option.id} option={option}>
              <span className="inline-flex items-center gap-2 lowercase">
                <label className="text-sm first-letter:capitalize">{option.name}</label>
              </span>
            </Select.Option>
          );
        })}
      </Select.Options>
    </Select>
  );
}
