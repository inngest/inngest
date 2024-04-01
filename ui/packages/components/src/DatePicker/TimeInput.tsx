import { useEffect, useState } from 'react';
import { format } from 'date-fns';
import { ZodError, type ZodSchema } from 'zod';

import { cn } from '../utils/classNames';
import {
  getHourSchema,
  millisecondsSchema,
  minutesSchema,
  periodSchema,
  secondsSchema,
} from './timeSchemas';

type HandleTimeChangeProps = {
  e: React.ChangeEvent<HTMLInputElement>;
  schema: ZodSchema<any>;
  setInputValue: React.Dispatch<React.SetStateAction<string>>;
};

type TimeInputProps = {
  is24HourFormat?: boolean;
  selectedTime?: Date;
  onSelect: React.Dispatch<React.SetStateAction<Date | undefined>>;
  setIsValidTime: React.Dispatch<React.SetStateAction<boolean>>;
  isValidTime: boolean;
};

export function TimeInput({
  selectedTime,
  onSelect,
  is24HourFormat = false,
  setIsValidTime,
  isValidTime,
}: TimeInputProps) {
  const [hourInput, setHourInput] = useState(
    selectedTime ? format(selectedTime, is24HourFormat ? 'HH' : 'hh') : ''
  );
  const [minuteInput, setMinuteInput] = useState(selectedTime ? format(selectedTime, 'mm') : '');
  const [secondInput, setSecondInput] = useState(selectedTime ? format(selectedTime, 'ss') : '');
  const [millisecondInput, setMillisecondInput] = useState(
    selectedTime ? format(selectedTime, 'SSS') : ''
  );
  const [periodInput, setPeriodInput] = useState(
    selectedTime && !is24HourFormat ? format(selectedTime, 'a') : ''
  );

  useEffect(() => {
    if (!isValidTime) {
      onSelect(undefined);
      return;
    }
    // Aggregates the multiple input time parts and combines in one date
    const newTimeDate = new Date();
    newTimeDate.setHours(parseInt(hourInput));
    newTimeDate.setMinutes(parseInt(minuteInput));
    newTimeDate.setSeconds(parseInt(secondInput));
    newTimeDate.setMilliseconds(parseInt(millisecondInput));
    if (periodInput && !is24HourFormat) {
      const hour = newTimeDate.getHours();
      if (periodInput.toUpperCase() === 'PM' && hour < 12) {
        newTimeDate.setHours(hour + 12);
      } else if (periodInput.toUpperCase() === 'AM' && hour === 12) {
        newTimeDate.setHours(0);
      }
    }
    if (isNaN(newTimeDate.getTime())) {
      return;
    }
    onSelect(newTimeDate);
  }, [hourInput, minuteInput, secondInput, millisecondInput, periodInput]);

  useEffect(() => {
    // Changes the hour and period when switch between 24h and AM/PM format
    setHourInput(selectedTime ? format(selectedTime, is24HourFormat ? 'HH' : 'hh') : '');
    setPeriodInput(selectedTime && !is24HourFormat ? format(selectedTime, 'a') : '');
  }, [is24HourFormat]);

  const handleTimeChange = ({ e, schema, setInputValue }: HandleTimeChangeProps) => {
    const value = e.target.value;
    if (value.trim() === '') {
      setInputValue('');
      setIsValidTime(false);
      return;
    }
    try {
      schema.parse(value);
      setInputValue(value);
      setIsValidTime(true);
    } catch (error) {
      setIsValidTime(false);
      if (!(error instanceof ZodError)) {
        return;
      }
      const errorCode = error.errors?.[0]?.code;
      if (errorCode === 'too_big' || errorCode === 'too_small') {
        setInputValue(value);
      }
    }
  };

  const handlePeriodChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const inputPeriod = e.target.value;
    if (inputPeriod.trim() === '') {
      setPeriodInput('');
      setIsValidTime(false);
      return;
    }
    const parsedInputPeriod = inputPeriod.toUpperCase();
    try {
      periodSchema.parse(parsedInputPeriod);
      setPeriodInput(parsedInputPeriod);
      setIsValidTime(true);
    } catch (error) {
      setIsValidTime(false);
      if (
        parsedInputPeriod.length === 1 &&
        (parsedInputPeriod.startsWith('A') || parsedInputPeriod.startsWith('P'))
      ) {
        setPeriodInput(parsedInputPeriod);
      }
    }
  };

  return (
    <div
      className={cn(
        'flex h-8 items-center rounded-lg border-2 border-transparent bg-white px-3.5 text-sm leading-none placeholder-slate-500 transition-all has-[:focus]:border-indigo-500',
        !isValidTime && 'has-[:focus]:border-rose-500'
      )}
    >
      <input
        className="w-7 px-0.5 text-center focus:outline-none"
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
        value={hourInput}
      />
      <span className="px-0.5">:</span>
      <input
        className="w-7 px-0.5 text-center focus:outline-none"
        placeholder="mm"
        aria-label="Point in time (Minutes)"
        maxLength={2}
        onChange={(e) =>
          handleTimeChange({ e, schema: minutesSchema, setInputValue: setMinuteInput })
        }
        value={minuteInput}
      />
      <span className="px-0.5">:</span>
      <input
        className="w-7 px-0.5 text-center focus:outline-none"
        placeholder="ss"
        aria-label="Point in time (Seconds)"
        maxLength={2}
        onChange={(e) =>
          handleTimeChange({ e, schema: secondsSchema, setInputValue: setSecondInput })
        }
        value={secondInput}
      />
      <span className="px-0.5">.</span>
      <input
        className="w-9 px-0.5 text-center focus:outline-none"
        placeholder="sss"
        aria-label="Point in time (Milliseconds)"
        maxLength={3}
        onChange={(e) =>
          handleTimeChange({ e, schema: millisecondsSchema, setInputValue: setMillisecondInput })
        }
        value={millisecondInput}
      />
      {!is24HourFormat && (
        <input
          className="w-7 pl-0.5 focus:outline-none"
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
