import { forwardRef } from 'react';
import { subtractDuration } from '@inngest/components/utils/date';
import type { Duration } from 'date-fns';

import { DateSelectButton, RangePicker, type DateButtonProps } from '../DatePicker';
import type { RangeChangeProps } from '../DatePicker/RangePicker';

type Props = {
  daysAgoMax: number;
  onDaysChange: (value: RangeChangeProps) => void;
  defaultValue?: RangeChangeProps;
  minDuration?: Duration;
  className?: string;
};

export function TimeFilter({
  daysAgoMax,
  onDaysChange,
  defaultValue,
  minDuration,
  className,
}: Props) {
  return (
    <RangePicker
      className={className}
      defaultValue={defaultValue}
      onChange={(range) => {
        onDaysChange(range);
      }}
      daysAgoMax={daysAgoMax}
      minDuration={minDuration}
      upgradeCutoff={subtractDuration(new Date(), { days: daysAgoMax || 7 })}
      triggerComponent={forwardRef<HTMLButtonElement, DateButtonProps>((props, ref) => (
        <DateSelectButton {...props} ref={ref} className={`${props.className || ''} `} />
      ))}
    />
  );
}
