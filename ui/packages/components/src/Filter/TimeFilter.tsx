import { Select } from '../Select/Select';
import {
  FunctionRunTimeFields,
  isFunctionTimeField,
  type FunctionRunTimeField,
} from '../types/functionRun';

type TimeFilterProps = {
  selectedTimeField?: FunctionRunTimeField;
  onTimeFieldChange: (value: FunctionRunTimeField) => void;
};

function replaceUnderscoreWithSpace(option: FunctionRunTimeField) {
  return option.replace(/_/g, ' ');
}

export default function TimeFilter({
  selectedTimeField = 'QUEUED_AT',
  onTimeFieldChange,
}: TimeFilterProps) {
  return (
    <Select
      defaultValue={selectedTimeField}
      onChange={(value: string) => {
        if (isFunctionTimeField(value)) {
          onTimeFieldChange(value);
        }
      }}
      label="Status"
      isLabelVisible={false}
    >
      <Select.Button>
        <span className="pr-2 text-sm lowercase first-letter:capitalize">
          {replaceUnderscoreWithSpace(selectedTimeField)}
        </span>
      </Select.Button>
      <Select.Options>
        {FunctionRunTimeFields.map((option) => {
          return (
            <Select.Option key={option} option={option}>
              <span className="inline-flex items-center gap-2 lowercase">
                <label className="text-sm first-letter:capitalize">
                  {replaceUnderscoreWithSpace(option)}
                </label>
              </span>
            </Select.Option>
          );
        })}
      </Select.Options>
    </Select>
  );
}
