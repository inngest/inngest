import { useState } from 'react';
import { ZodError, type ZodSchema } from 'zod';

import {
  getHourSchema,
  millisecondsSchema,
  minutesSchema,
  periodSchema,
  secondsSchema,
} from './timeSchemas';

type TimeInputProps = {
  is24Format?: boolean;
};

export function TimeInput({ is24Format = false }: TimeInputProps) {
  const [hourInput, setHourInput] = useState('');
  const [minuteInput, setMinuteInput] = useState('');
  const [secondInput, setSecondInput] = useState('');
  const [millisecondInput, setMillisecondInput] = useState('');
  const [periodInput, setPeriodInput] = useState('');

  type HandleTimeChangeProps = {
    e: React.ChangeEvent<HTMLInputElement>;
    schema: ZodSchema<any>;
    setInputValue: React.Dispatch<React.SetStateAction<string>>;
  };

  const handleTimeChange = ({ e, schema, setInputValue }: HandleTimeChangeProps) => {
    const value = e.target.value;
    try {
      schema.parse(value);
      setInputValue(value);
    } catch (error) {
      if (!(error instanceof ZodError)) {
        return;
      }
      const errorCode = error.errors?.[0]?.code;
      if (value === '') {
        setInputValue('');
      } else if (errorCode === 'too_big' || errorCode === 'too_small') {
        setInputValue(value);
      }
    }
  };

  const handlePeriodChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const inputPeriod = e.target.value;
    const parsedInputPeriod = inputPeriod.toUpperCase();
    try {
      periodSchema.parse(parsedInputPeriod);
      setPeriodInput(parsedInputPeriod);
    } catch (error) {
      if (parsedInputPeriod === '') {
        setPeriodInput('');
      } else if (
        parsedInputPeriod.length === 1 &&
        (parsedInputPeriod.startsWith('A') || parsedInputPeriod.startsWith('P'))
      ) {
        setPeriodInput(parsedInputPeriod);
      }
    }
  };

  return (
    <div className="flex h-8 items-center rounded-lg border-2 border-transparent bg-white px-3.5 text-sm leading-none placeholder-slate-500 transition-all has-[:focus]:border-indigo-500">
      <input
        className="w-7 px-0.5 text-center focus:outline-none"
        placeholder="HH"
        aria-label="Point in time (Hours)"
        maxLength={2}
        onChange={(e) =>
          handleTimeChange({ e, schema: getHourSchema(is24Format), setInputValue: setHourInput })
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
      {!is24Format && (
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
