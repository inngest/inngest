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
            <Select.CustomOption
              key={option}
              value={option}
              className="ui-active:bg-blue-50 flex select-none items-center justify-between px-2 py-4 focus:outline-none"
            >
              {({ selected }: { selected: boolean }) => (
                <span className="inline-flex items-center gap-2 lowercase">
                  <span className="inline-flex items-center gap-2 lowercase">
                    <input
                      type="checkbox"
                      id={option}
                      checked={selected}
                      readOnly
                      className="h-[15px] w-[15px] rounded border-slate-300 text-indigo-500 drop-shadow-sm checked:border-indigo-500 checked:drop-shadow-none"
                    />
                    <span className="flex items-center gap-1">
                      <RunStatusIcon status={option} className="h-2 w-2" />
                      <label className="text-sm first-letter:capitalize">{option}</label>
                    </span>
                  </span>
                </span>
              )}
            </Select.CustomOption>
          );
        })}
      </Select.Options>
    </Select>
  );
}
