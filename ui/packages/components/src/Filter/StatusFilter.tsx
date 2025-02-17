import { useRef, useState } from 'react';

import { Select, type Option } from '../Select/Select';
import { StatusDot } from '../Status/StatusDot';
import { getStatusBackgroundClass, getStatusBorderClass } from '../Status/statusClasses';
import {
  functionRunStatuses,
  isFunctionRunStatus,
  type FunctionRunStatus,
} from '../types/functionRun';
import { cn } from '../utils/classNames';

type StatusFilterProps = {
  selectedStatuses: FunctionRunStatus[];
  onStatusesChange: (value: FunctionRunStatus[]) => void;
  functionIsPaused?: boolean;
};

export default function StatusFilter({
  selectedStatuses,
  onStatusesChange,
  functionIsPaused,
}: StatusFilterProps) {
  const [temporarySelectedStatuses, setTemporarySelectedStatuses] = useState(selectedStatuses);
  const comboboxRef = useRef<HTMLButtonElement>(null);
  const availableStatuses: FunctionRunStatus[] = functionRunStatuses.filter((status) => {
    if (status === 'PAUSED') {
      return !!functionIsPaused;
    } else if (status === 'RUNNING') {
      return !functionIsPaused;
      // Hide skipped runs from filter
    } else if (status === 'SKIPPED') {
      return false;
    }
    return true;
  });
  const options: Option[] = availableStatuses.map((status: FunctionRunStatus) => ({
    id: status,
    name: status,
  }));
  const selectedValues = options.filter((option) =>
    temporarySelectedStatuses.some((status) => isFunctionRunStatus(status) && status === option.id)
  );
  const areAllStatusesSelected = availableStatuses.every((status) =>
    temporarySelectedStatuses.includes(status)
  );
  const statusDots = temporarySelectedStatuses.map((status) => {
    const isSelected = temporarySelectedStatuses.includes(status);
    return (
      <span
        key={status}
        className={cn(
          'border-subtle bg-canvasBase group-hover:border-subtle inline-block h-[9px] w-[9px] flex-shrink-0 rounded-full border [&:not(:first-child)]:-ml-1',
          isSelected && [getStatusBackgroundClass(status), getStatusBorderClass(status), 'ring-0']
        )}
        aria-hidden="true"
      />
    );
  });

  const handleApply = () => {
    onStatusesChange(temporarySelectedStatuses);
    // Close the Select dropdown
    if (comboboxRef.current) {
      comboboxRef.current.click();
    }
  };

  const isSelectionChanged = () => {
    if (temporarySelectedStatuses.length !== selectedStatuses.length) return true;
    const tempSet = new Set(temporarySelectedStatuses);
    return selectedStatuses.some((status) => !tempSet.has(status));
  };

  const isDisabledApply = !isSelectionChanged();

  return (
    <Select
      multiple
      value={selectedValues}
      onChange={(value: Option[]) => {
        const newValue: FunctionRunStatus[] = [];
        value.forEach((status) => {
          if (isFunctionRunStatus(status.id)) {
            newValue.push(status.id);
          } else {
            console.error(`invalid status: ${status.id}`);
          }
        });
        setTemporarySelectedStatuses(newValue);
      }}
      label="Status"
      isLabelVisible
    >
      <Select.Button isLabelVisible ref={comboboxRef}>
        <div className="w-7 text-left">
          {temporarySelectedStatuses.length > 0 && !areAllStatusesSelected && (
            <span>{statusDots}</span>
          )}
          {(temporarySelectedStatuses.length === 0 || areAllStatusesSelected) && <span>All</span>}
        </div>
      </Select.Button>
      <Select.Options>
        {options.map((option) => {
          if (!isFunctionRunStatus(option.id)) return;
          return (
            <Select.CheckboxOption key={option.id} option={option}>
              <span className="flex items-center gap-1 lowercase">
                <StatusDot status={option.id} className="h-2 w-2" />
                <label className="text-sm first-letter:capitalize">{option.name}</label>
              </span>
            </Select.CheckboxOption>
          );
        })}
        <Select.Footer onApply={handleApply} disabledApply={isDisabledApply} />
      </Select.Options>
    </Select>
  );
}
