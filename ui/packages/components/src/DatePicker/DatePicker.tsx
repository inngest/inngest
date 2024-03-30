import { useEffect, useState } from 'react';

import { Button } from '../Button';
import { Popover, PopoverClose, PopoverContent, PopoverTrigger } from '../Popover';
import { Switch, SwitchLabel, SwitchWrapper } from '../Switch';
import { combineDayAndTime, formatDayString, formatTimeString } from '../utils/date';
import { Calendar } from './Calendar';
import { DateInputButton } from './DateInputButton';
import { TimeInput } from './TimeInput';

type DatePickerProps = {
  datePickerDate?: Date;
  setDatePickerDate: React.Dispatch<React.SetStateAction<Date | undefined>>;
};

export function DatePicker({ datePickerDate, setDatePickerDate }: DatePickerProps) {
  const [buttonCopy, setButtonCopy] = useState<string>(
    datePickerDate?.toISOString() || 'Enter Date and Time'
  );
  const [selectedDay, setSelectedDay] = useState<Date | undefined>(datePickerDate);
  const [selectedTime, setSelectedTime] = useState<Date | undefined>(datePickerDate);
  const [calendarOpen, setCalendarOpen] = useState(false);
  const [is24HourFormat, setIs24HourFormat] = useState(false);
  const [dayString, setDayString] = useState<string>('');
  const [timeString, setTimeString] = useState<string>('');

  useEffect(() => {
    // Reset selected day and time when the popover is closed
    if (!calendarOpen) {
      setSelectedDay(datePickerDate);
      setSelectedTime(datePickerDate);
    }
  }, [calendarOpen, datePickerDate]);

  useEffect(() => {
    // To do: date validation
    // Generates the day and time string for the footer
    const dateString = selectedDay ? formatDayString(selectedDay) : '';
    setDayString(dateString);
    const timeString = selectedTime
      ? formatTimeString({ date: selectedTime, is24HourFormat: is24HourFormat })
      : '';
    setTimeString(timeString);
  }, [selectedDay, selectedTime, is24HourFormat]);

  function handleApply() {
    // To do: date validation
    if (selectedDay && selectedTime) {
      const combinedDate = combineDayAndTime({ day: selectedDay, time: selectedTime });
      setDatePickerDate(combinedDate);
      const string = combinedDate?.toISOString();
      if (string) {
        setButtonCopy(string);
      }
    }
    setCalendarOpen(false);
  }

  return (
    <Popover open={calendarOpen} onOpenChange={setCalendarOpen}>
      <PopoverTrigger asChild>
        <DateInputButton>{buttonCopy}</DateInputButton>
      </PopoverTrigger>
      <PopoverContent>
        <div className="p-4">
          <Calendar selected={selectedDay} onSelect={setSelectedDay} />
        </div>
        <div className="bg-slate-300 p-4">
          <div className="flex items-center justify-between pb-4">
            <p className="text-sm font-medium">UTC</p>
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
          <TimeInput
            is24HourFormat={is24HourFormat}
            selectedTime={selectedTime}
            onSelect={setSelectedTime}
          />
        </div>
        <footer className="p-4">
          {/* To do: error handling */}
          {/* <p className="pb-2 text-xs font-medium text-rose-600">Error</p> */}
          <div className="flex items-center justify-between">
            <div>
              <time className="block text-sm text-slate-600">{dayString}</time>
              <time className="text-sm text-slate-600">{timeString}</time>
            </div>
            <div className="flex justify-end gap-2 text-right">
              <PopoverClose asChild>
                <Button appearance="outlined" label="Cancel" />
              </PopoverClose>
              <Button kind="primary" label="Apply" btnAction={handleApply} />
            </div>
          </div>
        </footer>
      </PopoverContent>
    </Popover>
  );
}
