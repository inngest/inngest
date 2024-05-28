import { useEffect, useState } from 'react';

import { Switch, SwitchLabel, SwitchWrapper } from '../Switch';
import { combineDayAndTime } from '../utils/date';
import { Calendar } from './Calendar';
import { DateTimeInput } from './DateTimeInput';

type InternalPickerProps = {
  defaultValue: Date;
  onChange: (value: Date | undefined) => void;
};

export const DateTimePicker = ({ defaultValue, onChange }: InternalPickerProps) => {
  const [dateTime, setDateTime] = useState<Date | undefined>(defaultValue);
  const [is24HourFormat, setIs24HourFormat] = useState(false);
  const [isValidTime, setIsValidTime] = useState(true);
  const [calendarDate, setCalendarDate] = useState<Date | undefined>(defaultValue);
  const [inputDate, setInputDate] = useState<Date | undefined>(defaultValue);

  useEffect(() => {
    onChange(dateTime);
  }, [dateTime]);

  return (
    <div>
      <div className="mt-2 p-2">
        <Calendar
          month={calendarDate}
          selected={calendarDate}
          onSelect={(day) => {
            if (day) {
              const d = combineDayAndTime({ day, time: dateTime || new Date() });
              setCalendarDate(d);
              setInputDate(d);
              setDateTime(d);
            }
          }}
          onMonthChange={(month) => {
            if (month) {
              const d = combineDayAndTime({ day: month, time: dateTime || new Date() });
              setCalendarDate(d);
              setInputDate(d);
              setDateTime(d);
            }
          }}
        />
      </div>
      <div className="w-full border-b border-t border-slate-200 p-4">
        <div className="flex items-center justify-between pb-4">
          <p className="text-sm font-medium">{Intl.DateTimeFormat().resolvedOptions().timeZone}</p>
          <SwitchWrapper>
            <Switch
              checked={is24HourFormat}
              onCheckedChange={() => {
                setIs24HourFormat((prev) => !prev);
              }}
              id="24hr"
            />
            <SwitchLabel htmlFor="24hr">24hr</SwitchLabel>
          </SwitchWrapper>
        </div>
        <DateTimeInput
          is24HourFormat={is24HourFormat}
          selectedDateTime={inputDate}
          onSelect={(d) => {
            if (d) {
              setCalendarDate(d);
              setDateTime(d);
            }
          }}
          setIsValidTime={setIsValidTime}
          isValidTime={isValidTime}
        />
      </div>
    </div>
  );
};
