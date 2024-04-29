import { RunStatusIcon, statusStyles } from '../FunctionRunStatusIcon/RunStatusIcons';
import { Select } from '../Select/Select';
import { functionRunStatuses, type FunctionRunStatus } from '../types/functionRun';
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
      onChange={onStatusesChange}
      options={functionRunStatuses}
      label="Status"
    >
      <Select.Button>{statusDots}</Select.Button>
      <Select.Options options={functionRunStatuses} multiple>
        {(option: string) => <RunStatusIcon status={option} className="h-2 w-2" />}
      </Select.Options>
    </Select>
  );
}
