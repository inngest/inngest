import React from 'react';
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/react/20/solid';
import {
  Calendar as AriaCalendar,
  CalendarGridHeader as AriaCalendarGridHeader,
  Button,
  CalendarCell,
  CalendarGrid,
  CalendarGridBody,
  CalendarHeaderCell,
  Heading,
  Text,
  useLocale,
  type CalendarProps as AriaCalendarProps,
  type DateValue,
} from 'react-aria-components';
import { tv } from 'tailwind-variants';

export const focusRing = tv({
  base: 'outline outline-indigo-600 dark:outline-indigo-500 forced-colors:outline-[Highlight] outline-offset-2',
  variants: {
    isFocusVisible: {
      false: 'outline-0',
      true: 'outline-2',
    },
  },
});

const cellStyles = tv({
  extend: focusRing,
  base: 'text-sm font-medium size-11 cursor-default rounded-full flex items-center justify-center forced-color-adjust-none',
  variants: {
    isSelected: {
      false:
        'text-slate-900 dark:text-slate-200 hover:bg-indigo-200 dark:hover:bg-slate-700 pressed:bg-gray-200 dark:pressed:bg-slate-600',
      true: 'bg-indigo-600 invalid:bg-red-600 text-white forced-colors:bg-[Highlight] forced-colors:invalid:bg-[Mark] forced-colors:text-[HighlightText]',
    },
    isDisabled: {
      true: 'text-slate-400 dark:text-slate-600 forced-colors:text-[GrayText]',
    },
  },
});

export interface CalendarProps<T extends DateValue>
  extends Omit<AriaCalendarProps<T>, 'visibleDuration'> {
  errorMessage?: string;
}

export function Calendar<T extends DateValue>({ errorMessage, ...props }: CalendarProps<T>) {
  return (
    <AriaCalendar {...props}>
      <CalendarHeader />
      <CalendarGrid weekdayStyle="long">
        <CalendarGridHeader />
        <CalendarGridBody>
          {(date) => <CalendarCell date={date} className={cellStyles} />}
        </CalendarGridBody>
      </CalendarGrid>
      {errorMessage && (
        <Text slot="errorMessage" className="text-sm text-red-600">
          {errorMessage}
        </Text>
      )}
    </AriaCalendar>
  );
}

export function CalendarHeader() {
  const { direction } = useLocale();

  return (
    <header className="flex w-full items-center gap-1 px-1 pb-4">
      <Button slot="previous">
        {direction === 'rtl' ? (
          <ChevronRightIcon aria-hidden className="size-6" />
        ) : (
          <ChevronLeftIcon aria-hidden className="size-6" />
        )}
      </Button>
      <Heading className="mx-2 flex-1 text-center text-lg font-medium text-slate-900 dark:text-slate-200" />
      <Button slot="next">
        {direction === 'rtl' ? (
          <ChevronLeftIcon aria-hidden className="size-6" />
        ) : (
          <ChevronRightIcon aria-hidden className="size-6" />
        )}
      </Button>
    </header>
  );
}

export function CalendarGridHeader() {
  return (
    <AriaCalendarGridHeader>
      {(day) => (
        <CalendarHeaderCell className="text-sm font-medium text-slate-500">
          {day.substring(0, 2)}
        </CalendarHeaderCell>
      )}
    </AriaCalendarGridHeader>
  );
}
