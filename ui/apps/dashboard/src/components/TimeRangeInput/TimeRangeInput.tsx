'use client';

import { useState } from 'react';

import cn from '@/utils/cn';
import { TimeInput } from './TimeInput';

type TimeRange = {
  start: Date;
  end: Date;
};

type Props = {
  onChange: (timeRange: TimeRange | undefined) => void;
};

export function TimeRangeInput({ onChange }: Props) {
  const [startDateTime, setStartDateTime] = useState<Date>();
  const [startDateTimeError, setStartDateTimeError] = useState<string>();
  const [endDateTime, setEndDateTime] = useState<Date>();
  const [endDateTimeError, setEndDateTimeError] = useState<string | undefined>();

  function onStartDateTimeChange(newStartDateTime: Date | undefined) {
    setStartDateTime(newStartDateTime);
    setStartDateTimeError(undefined);
    setEndDateTimeError(undefined);
    if (newStartDateTime && endDateTime && newStartDateTime > endDateTime) {
      setStartDateTimeError('Start time must be before end time');
      return;
    }

    if (endDateTime && newStartDateTime) {
      onChange({ start: newStartDateTime, end: endDateTime });
      return;
    }
    onChange(undefined);
  }

  function onEndDateTimeChange(newEndDateTime: Date | undefined) {
    setEndDateTime(newEndDateTime);
    setStartDateTimeError(undefined);
    setEndDateTimeError(undefined);
    if (startDateTime && newEndDateTime && newEndDateTime < startDateTime) {
      setEndDateTimeError('End time must be after start time');
      return;
    }

    if (startDateTime && newEndDateTime) {
      onChange({ start: startDateTime, end: newEndDateTime });
      return;
    }
    onChange(undefined);
  }

  return (
    <div className="space-y-1">
      <div className="flex items-start gap-2">
        <div className="min-w-[190px]">
          <TimeInput onChange={onStartDateTimeChange} placeholder="start time" required />
          <p
            className={cn(
              'mt-1 pl-2 text-xs',
              startDateTimeError ? 'text-red-500' : 'text-slate-400'
            )}
          >
            {startDateTimeError
              ? startDateTimeError
              : startDateTime
              ? startDateTime.toISOString()
              : '-'}
          </p>
        </div>
        <span className="mt-1.5 text-sm font-medium text-slate-400">to</span>
        <div className="min-w-[190px]">
          <TimeInput onChange={onEndDateTimeChange} placeholder="end time" required />
          <p
            className={cn(
              'mt-1 pl-2 text-xs',
              endDateTimeError ? 'text-red-500' : 'text-slate-400'
            )}
          >
            {endDateTimeError ? endDateTimeError : endDateTime ? endDateTime.toISOString() : '-'}
          </p>
        </div>
      </div>
    </div>
  );
}
