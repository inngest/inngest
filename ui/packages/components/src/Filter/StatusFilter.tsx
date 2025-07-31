import { useRef, useState } from 'react';

import { Select, type Option } from '../Select/Select';
import { StatusDot } from '../Status/StatusDot';
import { getStatusBackgroundClass, getStatusBorderClass } from '../Status/statusClasses';
import { cn } from '../utils/classNames';

type StatusFilterProps<T extends string> = {
  selectedStatuses: T[];
  onStatusesChange: (value: T[]) => void;
  availableStatuses: T[];
  isValidStatus: (status: string) => status is T;
};

export default function StatusFilter<T extends string>({
  selectedStatuses,
  onStatusesChange,
  availableStatuses,
  isValidStatus,
}: StatusFilterProps<T>) {
  const [temporarySelectedStatuses, setTemporarySelectedStatuses] = useState(selectedStatuses);
  const comboboxRef = useRef<HTMLButtonElement>(null);

  const options: Option[] = availableStatuses.map((status: string) => ({
    id: status,
    name: status,
  }));
  const selectedValues = options.filter((option) =>
    temporarySelectedStatuses.some((status) => isValidStatus(status) && status === option.id)
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
        const newValue: T[] = [];
        value.forEach((status) => {
          if (isValidStatus(status.id)) {
            newValue.push(status.id);
          } else {
            console.error(`invalid status: ${status.id}`);
          }
        });
        setTemporarySelectedStatuses(newValue);
      }}
      label="Status"
      isLabelVisible
      size="small"
      className="bg-modalBase"
    >
      <Select.Button isLabelVisible ref={comboboxRef} size="small">
        <div className="w-7 text-left">
          {temporarySelectedStatuses.length > 0 && !areAllStatusesSelected && (
            <span>{statusDots}</span>
          )}
          {(temporarySelectedStatuses.length === 0 || areAllStatusesSelected) && <span>All</span>}
        </div>
      </Select.Button>
      <Select.Options>
        {options.map((option) => {
          if (!isValidStatus(option.id)) return;
          return (
            <Select.CheckboxOption key={option.id} option={option}>
              <span className="flex items-center gap-1 lowercase">
                <StatusDot status={option.id} size="small" />
                <label className="first-letter:capitalize">{option.name}</label>
              </span>
            </Select.CheckboxOption>
          );
        })}
        <Select.Footer onApply={handleApply} disabledApply={isDisabledApply} />
      </Select.Options>
    </Select>
  );
}
