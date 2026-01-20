import { forwardRef } from 'react';
import { subtractDuration } from '@inngest/components/utils/date';

import { DateSelectButton, RangePicker, type DateButtonProps } from '../DatePicker';
import type { RangeChangeProps } from '../DatePicker/RangePicker';

type Props = {
  daysAgoMax: number;
  onDaysChange: (value: RangeChangeProps) => void;
  defaultValue?: RangeChangeProps;
  className?: string;
};

export function TimeFilter({ daysAgoMax, onDaysChange, defaultValue, className }: Props) {
  return (
    <RangePicker
      className={className}
      defaultValue={defaultValue}
      onChange={(range) => {
        onDaysChange(range);
      }}
      daysAgoMax={daysAgoMax}
      upgradeCutoff={subtractDuration(new Date(), { days: daysAgoMax || 7 })}
      triggerComponent={forwardRef<HTMLButtonElement, DateButtonProps>((props, ref) => (
        <DateSelectButton {...props} ref={ref} className={`${props.className || ''} `} />
      ))}
    />
  );
}
