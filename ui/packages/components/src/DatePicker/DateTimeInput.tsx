import { useEffect, useRef, useState, type MutableRefObject } from 'react';
import { format as formatDate, getDaysInMonth, isValid, parseISO } from 'date-fns';
import { ZodError, type ZodSchema } from 'zod';

import { cn } from '../utils/classNames';
import {
  getHourSchema,
  millisecondsSchema,
  minutesSchema,
  monthSchema,
  periodSchema,
  secondsSchema,
  yearSchema,
} from './timeSchemas';

type HandleTimeChangeProps = {
  e: React.ChangeEvent<HTMLInputElement>;
  schema: ZodSchema<any>;
  setInputValue: React.Dispatch<React.SetStateAction<string>>;
};

type TimeInputProps = {
  is24HourFormat?: boolean;
  selectedDateTime?: Date;
  onSelect: React.Dispatch<React.SetStateAction<Date | undefined>>;
  setValid: React.Dispatch<React.SetStateAction<boolean>>;
  valid: boolean;
  error?: string;
};

const isNumeric = (num: any) =>
  (typeof num === 'number' || (typeof num === 'string' && num.trim() !== '')) &&
  !isNaN(num as number);

const formatWithDefault = (format: string, defaultValue: string, date?: Date) =>
  date ? formatDate(date, format) : defaultValue;

export function DateTimeInput({
  selectedDateTime,
  onSelect,
  is24HourFormat = false,
  setValid,
  valid,
}: TimeInputProps) {
  const daysRef = useRef<HTMLInputElement>(null);
  const yearsRef = useRef<HTMLInputElement | null>(null);
  const hoursRef = useRef<HTMLInputElement | null>(null);
  const minutesRef = useRef<HTMLInputElement | null>(null);
  const secondsRef = useRef<HTMLInputElement | null>(null);
  const millisecondsRef = useRef<HTMLInputElement | null>(null);
  const meridiemRef = useRef<HTMLInputElement | null>(null);

  const [dayInput, setDayInput] = useState('00');
  const [monthInput, setMonthInput] = useState('00');
  const [yearInput, setYearInput] = useState('0000');
  const [hourInput, setHourInput] = useState('00');
  const [minuteInput, setMinuteInput] = useState('00');
  const [secondInput, setSecondInput] = useState('00');
  const [millisecondInput, setMillisecondInput] = useState('000');
  const [periodInput, setPeriodInput] = useState(is24HourFormat ? 'AM' : '');

  const populateFields = (date: Date) => {
    setDayInput(formatWithDefault('dd', '00', date));
    setMonthInput(formatWithDefault('MM', '00', date));
    setYearInput(formatWithDefault('yyyy', '0000', date));
    setHourInput(formatWithDefault(is24HourFormat ? 'HH' : 'hh', '00', date));
    setMinuteInput(formatWithDefault('mm', '00', date));
    setSecondInput(formatWithDefault('ss', '00', date));
    setMillisecondInput(formatWithDefault('SSS', '000', date));
    setPeriodInput(!is24HourFormat ? formatWithDefault('a', '', date) : 'AM');
  };

  useEffect(() => {
    //
    // keeps the input fields in sycn with the calendar widget
    selectedDateTime && populateFields(selectedDateTime);
  }, [selectedDateTime]);

  useEffect(() => {
    if (yearInput === '0000' || monthInput === '00' || dayInput == '00') {
      onSelect(undefined);
      return;
    }
    // Aggregates the multiple input time parts and combines in one date
    const newDate = new Date(parseInt(yearInput), parseInt(monthInput) - 1, parseInt(dayInput));

    newDate.setHours(parseInt(hourInput));
    newDate.setMinutes(parseInt(minuteInput));
    newDate.setSeconds(parseInt(secondInput));
    newDate.setMilliseconds(parseInt(millisecondInput));

    if (periodInput && !is24HourFormat) {
      const hour = newDate.getHours();
      if (periodInput.toUpperCase() === 'PM' && hour < 12) {
        newDate.setHours(hour + 12);
      } else if (periodInput.toUpperCase() === 'AM' && hour === 12) {
        newDate.setHours(0);
      }
    }

    if (!isValid(newDate)) {
      setValid(false);
      return;
    }
    onSelect(newDate);
  }, [
    monthInput,
    dayInput,
    yearInput,
    hourInput,
    minuteInput,
    secondInput,
    millisecondInput,
    periodInput,
  ]);

  useEffect(() => {
    // Changes the hour and period when switch between 24h and AM/PM format
    setHourInput(
      selectedDateTime ? formatDate(selectedDateTime, is24HourFormat ? 'HH' : 'hh') : '00'
    );
    setPeriodInput(selectedDateTime && !is24HourFormat ? formatDate(selectedDateTime, 'a') : 'AM');
  }, [is24HourFormat]);

  const handleTimeChange = ({ e, schema, setInputValue }: HandleTimeChangeProps) => {
    const value = e.target.value;
    if (value.trim() === '') {
      setInputValue('');
      setValid(false);
      return;
    }
    try {
      schema.parse(value);
      setInputValue(value);
      setValid(true);
    } catch (error) {
      setValid(false);
      if (!(error instanceof ZodError)) {
        return;
      }
      const errorCode = error.errors?.[0]?.code;
      if (errorCode === 'too_big' || errorCode === 'too_small') {
        setInputValue(value);
      }
    }
  };

  const handleDayChange = ({
    e,
    setInputValue,
  }: {
    e: React.ChangeEvent<HTMLInputElement>;

    setInputValue: React.Dispatch<React.SetStateAction<string>>;
  }) => {
    const value = e.target.value;
    if (value.trim() === '') {
      setInputValue('');
      setValid(false);
      return;
    }
    const days = getDaysInMonth(new Date(parseInt(yearInput), parseInt(monthInput) - 1));

    setValid(isNumeric(value) && Number(value) >= 1 && Number(value) <= days);
    setInputValue(value);
  };

  const handlePeriodChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const inputPeriod = e.target.value;
    if (inputPeriod.trim() === '') {
      setPeriodInput('');
      setValid(false);
      return;
    }
    const parsedInputPeriod = inputPeriod.toUpperCase();
    try {
      periodSchema.parse(parsedInputPeriod);
      setPeriodInput(parsedInputPeriod);
      setValid(true);
    } catch (error) {
      setValid(false);
      if (
        parsedInputPeriod.length === 1 &&
        (parsedInputPeriod.startsWith('A') || parsedInputPeriod.startsWith('P'))
      ) {
        setPeriodInput(parsedInputPeriod);
      }
    }
  };

  const focusNext = (
    e: React.SyntheticEvent<HTMLInputElement>,
    length: number,
    ref: MutableRefObject<HTMLInputElement | null>
  ) => {
    const target = e.target as HTMLInputElement;
    if (
      target.selectionStart === length &&
      target.selectionEnd === length &&
      target.value?.length === length
    ) {
      ref.current?.focus();
      ref.current?.select();
    }
  };

  return (
    <div
      className={cn(
        'border-muted has-[:focus]:border-muted bg-canvasBase text-basis placeholder-disabled flex h-8 items-center rounded-md border px-3.5 text-sm leading-none transition-all',
        !valid && 'border-error has-[:focus]:border-error'
      )}
      onPaste={(e: React.ClipboardEvent<HTMLDivElement>) => {
        const raw = e?.clipboardData?.getData('text') || '';
        const d = parseISO(raw);

        //
        // Don't interfere with things that might be valid ISO dates that that are actually
        // single field pastes, e.g. 2025
        if (raw.length > 5 && isValid(d)) {
          e.preventDefault();
          populateFields(d);
          onSelect(d);
        }
      }}
    >
      <input
        className="bg-canvasBase w-7 px-0.5 text-center focus:outline-none"
        placeholder="MM"
        aria-label="Month"
        maxLength={2}
        onChange={(e) => handleTimeChange({ e, schema: monthSchema, setInputValue: setMonthInput })}
        onKeyUp={(e: React.SyntheticEvent<HTMLInputElement>) => focusNext(e, 2, daysRef)}
        value={monthInput}
      />
      <span className="px-0.5">/</span>
      <input
        ref={daysRef}
        className="bg-canvasBase w-7 px-0.5 text-center focus:outline-none"
        placeholder="dd"
        aria-label="Day"
        maxLength={2}
        onChange={(e) => handleDayChange({ e, setInputValue: setDayInput })}
        onKeyUp={(e: React.SyntheticEvent<HTMLInputElement>) => focusNext(e, 2, yearsRef)}
        value={dayInput}
      />
      <span className="px-0.5">/</span>
      <input
        ref={yearsRef}
        className="bg-canvasBase w-9 px-0.5 text-center focus:outline-none"
        placeholder="yyyy"
        aria-label="Year"
        maxLength={4}
        onChange={(e) => handleTimeChange({ e, schema: yearSchema, setInputValue: setYearInput })}
        onKeyUp={(e: React.SyntheticEvent<HTMLInputElement>) => focusNext(e, 4, hoursRef)}
        value={yearInput}
      />
      <span className="pr-0.5">,</span>
      <input
        ref={hoursRef}
        className="bg-canvasBase w-7 px-0.5 text-center focus:outline-none"
        placeholder="HH"
        aria-label="Point in time (Hours)"
        maxLength={2}
        onChange={(e) =>
          handleTimeChange({
            e,
            schema: getHourSchema(is24HourFormat),
            setInputValue: setHourInput,
          })
        }
        onKeyUp={(e: React.SyntheticEvent<HTMLInputElement>) => focusNext(e, 2, minutesRef)}
        value={hourInput}
      />
      <span className="px-0.5">:</span>
      <input
        ref={minutesRef}
        className="bg-canvasBase w-7 px-0.5 text-center focus:outline-none"
        placeholder="mm"
        aria-label="Point in time (Minutes)"
        maxLength={2}
        onChange={(e) =>
          handleTimeChange({ e, schema: minutesSchema, setInputValue: setMinuteInput })
        }
        onKeyUp={(e: React.SyntheticEvent<HTMLInputElement>) => focusNext(e, 2, secondsRef)}
        value={minuteInput}
      />
      <span className="px-0.5">:</span>
      <input
        ref={secondsRef}
        className="bg-canvasBase w-7 px-0.5 text-center focus:outline-none"
        placeholder="ss"
        aria-label="Point in time (Seconds)"
        maxLength={2}
        onChange={(e) =>
          handleTimeChange({ e, schema: secondsSchema, setInputValue: setSecondInput })
        }
        onKeyUp={(e: React.SyntheticEvent<HTMLInputElement>) => focusNext(e, 2, millisecondsRef)}
        value={secondInput}
      />
      <span className="px-0.5">.</span>
      <input
        ref={millisecondsRef}
        className="bg-canvasBase w-9 px-0.5 text-center focus:outline-none"
        placeholder="sss"
        aria-label="Point in time (Milliseconds)"
        maxLength={3}
        onChange={(e) =>
          handleTimeChange({ e, schema: millisecondsSchema, setInputValue: setMillisecondInput })
        }
        onKeyUp={(e: React.SyntheticEvent<HTMLInputElement>) =>
          !is24HourFormat && focusNext(e, 3, meridiemRef)
        }
        value={millisecondInput}
      />
      {!is24HourFormat && (
        <input
          ref={meridiemRef}
          className="bg-canvasBase w-7 pl-0.5 focus:outline-none"
          placeholder="AM"
          aria-label="Point in time (Period)"
          maxLength={2}
          onChange={handlePeriodChange}
          value={periodInput}
        />
      )}
    </div>
  );
}
