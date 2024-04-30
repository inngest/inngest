import { RunStatusIcon, statusStyles } from '../FunctionRunStatusIcon/RunStatusIcons';
import { Select } from '../Select/Select';
import {
  functionRunStatuses,
  isFunctionRunStatus,
  type FunctionRunStatus,
} from '../types/functionRun';
import { cn } from '../utils/classNames';

type StatusFilterProps = {
  selectedStatuses: FunctionRunStatus[];
  onStatusesChange: (value: FunctionRunStatus[]) => void;
};

export default function StatusFilter({ selectedStatuses, onStatusesChange }: StatusFilterProps) {
  const statusDots = selectedStatuses.map((status) => {
    const isSelected = selectedStatuses.includes(status);
    return (
      <span
        key={status}
        className={cn(
          'inline-block h-[9px] w-[9px] flex-shrink-0 rounded-full border border-slate-50 bg-slate-50 ring-1 ring-inset ring-slate-300 group-hover:border-slate-100 [&:not(:first-child)]:-ml-1',
          isSelected && [statusStyles[status], 'ring-0']
        )}
        aria-hidden="true"
      />
    );
  });

  return (
    <Select
      multiple
      defaultValue={selectedStatuses}
      onChange={(value: string[]) => {
        const newValue: FunctionRunStatus[] = [];
        value.forEach((status) => {
          if (isFunctionRunStatus(status)) {
            newValue.push(status);
          } else {
            console.error(`invalid status: ${status}`);
          }
        });
        onStatusesChange(newValue);
      }}
      label="Status"
    >
      <Select.Button>{statusDots}</Select.Button>
      <Select.Options>
        {functionRunStatuses.map((option) => {
          return (
            <Select.CheckboxOption key={option} option={option}>
              <span className="flex items-center gap-1 lowercase">
                <RunStatusIcon status={option} className="h-2 w-2" />
                <label className="text-sm first-letter:capitalize">{option}</label>
              </span>
            </Select.CheckboxOption>
          );
        })}
      </Select.Options>
    </Select>
  );
}
