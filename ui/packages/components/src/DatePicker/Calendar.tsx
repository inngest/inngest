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
  root: 'text-basis bg-canvasBase',
  months: 'flex gap-4 relative justify-center',
  caption_label: 'text-lg font-medium',
  nav_button:
    'inline-flex justify-center items-center absolute top-0 w-6 h-6 rounded-full text-subtle hover:bg-canvasSubtle',
  nav_button_next: 'right-2',
  nav_button_previous: 'left-2',
  table: 'border-collapse border-spacing-0 text-sm',
  head_cell: 'w-11 pt-6 pb-4 align-middle text-center font-medium text-subtle',
  cell: 'w-11 h-11 align-middle text-center border-0 font-medium',
  day: 'rounded-full w-10 h-10 transition-colors hover:bg-primary-xSubtle aria-selected:hover:text-basis aria-selected:hover:bg-primary-xSubtle hover:text-basis focus:outline-none focus-visible:ring focus-visible:ring-primary-subtle',
  day_selected: 'text-onContrast bg-primary-moderate',
  day_today: 'font-semibold',
  day_disabled: 'text-disabled',
  day_outside: 'text-disabled',
};
