import { useState } from 'react';
import CalendarIcon from '@heroicons/react/20/solid/CalendarIcon';
import { DayPicker, type DayPickerDefaultProps } from 'react-day-picker';

import { Button } from '../Button';
import { Popover, PopoverClose, PopoverContent, PopoverTrigger } from '../Popover';
import { Switch, SwitchLabel, SwitchWrapper } from '../Switch';

export function DatePicker({}) {
  const [buttonCopy, setButtonCopy] = useState<string>('Enter Date and Time');
  const [selectedDay, setSelectedDay] = useState<Date>();
  const [selectedTime, setSelectedTime] = useState<string>('');
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
        <button className="h-8 rounded-lg border border-slate-300 px-3.5 text-sm leading-none placeholder-slate-500 shadow outline-2 outline-offset-2 outline-indigo-500 transition-all focus:outline">
          <span className="flex items-center gap-2">
            <CalendarIcon className="h-6 w-6" />
            {buttonCopy}
          </span>
        </button>
      </PopoverTrigger>
      <PopoverContent>
        <div className="p-4">
          <DayPicker
            classNames={classNames}
            selected={selectedDay}
            onSelect={setSelectedDay}
            mode="single"
            showOutsideDays
            fixedWeeks
          />
        </div>
        <div className="bg-slate-300 p-4">
          <div className="flex items-center justify-between pb-4">
            <p>UTC</p>
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
          <input type="time" step="any" id="timeInput" />
        </div>
        <footer className="p-4">
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

const classNames: DayPickerDefaultProps['classNames'] = {
  caption: 'flex justify-center items-center h-6',
  root: 'text-slate-900',
  months: 'flex gap-4 relative',
  caption_label: 'text-lg font-medium',
  nav_button:
    'inline-flex justify-center items-center absolute top-0 w-6 h-6 rounded-full text-slate-900 hover:bg-gray-100',
  nav_button_next: 'right-0',
  nav_button_previous: 'left-0',
  table: 'border-collapse border-spacing-0 text-sm',
  head_cell: 'w-11 pt-6 pb-4 align-middle text-center font-medium text-slate-500',
  cell: 'w-11 h-11 align-middle text-center border-0 font-medium',
  day: 'rounded-full w-10 h-10 transition-colors hover:bg-indigo-100 aria-selected:hover:text-white aria-selected:hover:bg-indigo-600 hover:text-slate-500 focus:outline-none focus-visible:ring focus-visible:ring-indigo-500',
  day_selected: 'text-white bg-indigo-600',
  day_today: 'font-semibold',
  day_disabled: 'text-slate-400',
  day_outside: 'text-slate-400',
};
