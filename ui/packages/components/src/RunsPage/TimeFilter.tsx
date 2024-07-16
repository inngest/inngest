import { forwardRef } from 'react';
import { type Option } from '@inngest/components/Select/Select';

import { DateSelectButton, RangePicker, type DateButtonProps } from '../DatePicker';

const daysAgoArray = ['1', '3', '7', '14', '30'];

type Props = {
  daysAgoMax: number;
  onDaysChange: (value: string) => void;
  selectedDays: string;
};

export function TimeFilter({ daysAgoMax, onDaysChange, selectedDays }: Props) {
  const options: Option[] = daysAgoArray.map((date) => ({
    id: date,
    name: date === '1' ? `Last ${date} day` : `Last ${date} days`,
    disabled: parseInt(date) > daysAgoMax,
  }));

  const selectedValue = options.find((option) => option.id === selectedDays.toString());

  /* TODO: better plan validation and toast when absolute time available */
  // If selected value is disabled, select 3 days instead
  if (selectedValue && selectedValue.disabled) {
    const newSelectedValue = options.find(
      (option) => !option.disabled && parseInt(option.id) === 3
    );
    if (newSelectedValue) {
      onDaysChange(newSelectedValue.id);
    }
  }

  return (
    <RangePicker
      onChange={() => {}}
      triggerComponent={forwardRef<HTMLButtonElement, DateButtonProps>((props, ref) => (
        <DateSelectButton
          {...props}
          ref={ref}
          className={`${props.className || ''} rounded-l-none border-l-0`}
        />
      ))}
    />
    // <RelativeTimeFilter
    //   options={options}
    //   selectedDays={selectedValue}
    //   onDaysChange={(value: Option) => {
    //     const numericId = parseInt(value.id);
    //     if (!isNaN(numericId)) {
    //       onDaysChange(value.id);
    //     }
    //   }}
    // />
  );
}
