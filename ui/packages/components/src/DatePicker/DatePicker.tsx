import { useState } from 'react';

import { Button } from '../Button';
import { Popover, PopoverClose, PopoverContent, PopoverTrigger } from '../Popover';
import { Switch, SwitchLabel, SwitchWrapper } from '../Switch';
import { Calendar } from './Calendar';
import { DateInputButton } from './DateInputButton';
import { TimeInput } from './TimeInput';

export function DatePicker() {
  const [buttonCopy, setButtonCopy] = useState<string>('Enter Date and Time');
  const [selectedDay, setSelectedDay] = useState<Date>();
  const [calendarOpen, setCalendarOpen] = useState(false);
  const [is24HourFormat, setIs24HourFormat] = useState(false);

  function handleApply() {
    if (selectedDay) {
      const string = selectedDay.toISOString();
      setButtonCopy(string);
    } else {
      setButtonCopy('Enter Date and Time');
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
          <TimeInput is24Format={is24HourFormat} />
        </div>
        <footer className="p-4">
          <p className="pb-2 text-xs font-medium text-red-600">Error</p>
          <div className="flex justify-end gap-2 text-right">
            <PopoverClose asChild>
              <Button appearance="outlined" label="Cancel" />
            </PopoverClose>
            <Button kind="primary" label="Apply" btnAction={handleApply} />
          </div>
        </footer>
      </PopoverContent>
    </Popover>
  );
}
