import { DayPicker, type ClassNames } from 'react-day-picker';

type CalendarProps = {
  month?: Date;
  selected?: Date;
  onSelect?: (date: Date | undefined) => void;
  onMonthChange?: (month: Date) => void;
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

const classNames: Partial<ClassNames> = {
  month_caption: 'flex justify-center items-center h-6',
  root: 'text-basis bg-canvasBase',
  months: 'flex gap-4 relative justify-center',
  caption_label: 'text-lg font-medium',
  button_next:
    'inline-flex justify-center items-center absolute top-0 right-2 w-6 h-6 rounded-full text-subtle hover:bg-canvasSubtle',
  button_previous:
    'inline-flex justify-center items-center absolute top-0 left-2 w-6 h-6 rounded-full text-subtle hover:bg-canvasSubtle',
  month_grid: 'border-collapse border-spacing-0 text-sm',
  weekday: 'w-11 pt-6 pb-4 align-middle text-center font-medium text-subtle',
  day: 'w-11 h-11 align-middle text-center border-0 font-medium',
  day_button:
    'rounded-full w-10 h-10 transition-colors hover:bg-primary-xSubtle aria-selected:bg-primary-moderate aria-selected:text-onContrast aria-selected:hover:bg-primary-xSubtle aria-selected:hover:text-basis focus:outline-none focus-visible:ring focus-visible:ring-primary-subtle',
  selected: '',
  today: 'font-semibold',
  disabled: 'text-disabled',
  outside: 'text-disabled',
};
