import {
  DayPicker,
  type DayPickerDefaultProps,
  type MonthChangeEventHandler,
  type SelectSingleEventHandler,
} from 'react-day-picker';

type CalendarProps = {
  month?: Date;
  selected?: Date;
  onSelect?: SelectSingleEventHandler;
  onMonthChange?: MonthChangeEventHandler;
};

export function Calendar({ selected, onSelect, month, onMonthChange }: CalendarProps) {
  return (
    <DayPicker
      classNames={classNames}
      selected={selected}
      onSelect={onSelect}
      month={month}
      onMonthChange={onMonthChange}
      mode="single"
      showOutsideDays
      fixedWeeks
    />
  );
}

const classNames: DayPickerDefaultProps['classNames'] = {
  caption: 'flex justify-center items-center h-6',
  root: 'text-slate-900 bg-white',
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
