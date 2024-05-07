import { RunStatusIcon, statusStyles } from '../FunctionRunStatusIcon/RunStatusIcons';
import { Select, type Option } from '../Select/Select';
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

const options: Option[] = functionRunStatuses.map((status: FunctionRunStatus) => ({
  id: status,
  name: status,
}));

export default function StatusFilter({ selectedStatuses, onStatusesChange }: StatusFilterProps) {
  const selectedValues = options.filter((option) =>
    selectedStatuses.some((status) => isFunctionRunStatus(status) && status === option.id)
  );
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
      defaultValue={selectedValues}
      onChange={(value: Option[]) => {
        const newValue: FunctionRunStatus[] = [];
        value.forEach((status) => {
          if (isFunctionRunStatus(status.id)) {
            newValue.push(status.id);
          } else {
            console.error(`invalid status: ${status.id}`);
          }
        });
        onStatusesChange(newValue);
      }}
      label="Status"
    >
      <Select.Button>
        {selectedStatuses.length > 0 && <span className="pr-2">{statusDots}</span>}
      </Select.Button>
      <Select.Options>
        {options.map((option) => {
          if (!isFunctionRunStatus(option.id)) return;
          return (
            <Select.CheckboxOption key={option.id} option={option}>
              <span className="flex items-center gap-1 lowercase">
                <RunStatusIcon status={option.id} className="h-2 w-2" />
                <label className="text-sm first-letter:capitalize">{option.name}</label>
              </span>
            </Select.CheckboxOption>
          );
        })}
      </Select.Options>
    </Select>
  );
}
