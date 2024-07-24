import { forwardRef } from 'react';
import { subtractDuration } from '@inngest/components/utils/date';

import { DateSelectButton, RangePicker, type DateButtonProps } from '../DatePicker';
import type { RangeChangeProps } from '../DatePicker/RangePicker';

type Props = {
  daysAgoMax: number;
  onDaysChange: (value: RangeChangeProps) => void;
  defaultValue?: RangeChangeProps;
};

export function TimeFilter({ daysAgoMax, onDaysChange, defaultValue }: Props) {
  return (
    <RangePicker
      defaultValue={defaultValue}
      onChange={(range) => {
        onDaysChange(range);
      }}
      upgradeCutoff={subtractDuration(new Date(), { days: daysAgoMax || 7 })}
      triggerComponent={forwardRef<HTMLButtonElement, DateButtonProps>((props, ref) => (
        <DateSelectButton
          {...props}
          ref={ref}
          className={`${props.className || ''} rounded-l-none border-l-0`}
        />
      ))}
    />
  );
}
