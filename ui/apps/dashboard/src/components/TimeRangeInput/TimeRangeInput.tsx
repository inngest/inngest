'use client';

import { useState } from 'react';

import { TimeInput } from './TimeInput';

type TimeRange = {
  start: Date;
  end: Date;
};

type Props = {
  onChange: (timeRange: TimeRange) => void;
};

export function TimeRangeInput({ onChange }: Props) {
  const [startDateTime, setStartDateTime] = useState<Date>();
  const [startDateTimeError, setStartDateTimeError] = useState<string>();
  const [endDateTime, setEndDateTime] = useState<Date>();
  const [endDateTimeError, setEndDateTimeError] = useState<string | undefined>();

  function onStartDateTimeChange(newStartDateTime: Date) {
    setStartDateTime(newStartDateTime);
    setStartDateTimeError(undefined);
    setEndDateTimeError(undefined);
    if (endDateTime && newStartDateTime > endDateTime) {
      setStartDateTimeError('Start time must be before end time.');
      return;
    }

    if (endDateTime) {
      onChange({ start: newStartDateTime, end: endDateTime });
    }
  }

  function onEndDateTimeChange(newEndDateTime: Date) {
    setEndDateTime(newEndDateTime);
    setStartDateTimeError(undefined);
    setEndDateTimeError(undefined);
    if (startDateTime && newEndDateTime < startDateTime) {
      setEndDateTimeError('End time must be after start time.');
      return;
    }

    if (startDateTime) {
      onChange({ start: startDateTime, end: newEndDateTime });
    }
  }

  return (
    <div className="space-y-1">
      <div className="flex items-center gap-2">
        <TimeInput onChange={onStartDateTimeChange} required />
        <span className="text-sm font-medium text-slate-400">to</span>
        <TimeInput onChange={onEndDateTimeChange} required />
      </div>
      {startDateTimeError && <p className="text-sm text-red-500">{startDateTimeError}</p>}
      {endDateTimeError && <p className="text-right text-sm text-red-500">{endDateTimeError}</p>}
    </div>
  );
}
