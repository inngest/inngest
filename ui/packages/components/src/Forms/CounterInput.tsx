'use client';

import { useState } from 'react';
import { RiAddFill, RiAlertFill, RiSubtractFill } from '@remixicon/react';

import { Button } from '../Button';
import { Input } from './Input';

type CounterInputProps = {
  value?: number;
  min?: number;
  max?: number;
  onChange: (value: number) => void;
  onValid: (valid: boolean) => void;
  step?: number;
};

export default function CounterInput({
  value = 0,
  min = 0,
  max = 100,
  onChange,
  onValid,
  step = 1,
}: CounterInputProps) {
  const [err, setErr] = useState<string | null>(null);

  const increment = () => {
    // NaN indicates invalid or empty input:
    if (isNaN(value)) {
      value = 0;
    }
    // use min() to be sure the result is <= max, even after any rounding.
    // if the result is unaligned with the step interval (due to user input), adjust it downward to be aligned.
    // this guarantees the input value is always valid after the user clicks +.
    let newValue = Math.min(value + step, max);
    if (newValue != max && (newValue - min) % step !== 0) {
      newValue -= (newValue - min) % step;
    }
    setErr(null);
    onValid(true);
    onChange(newValue);
  };

  const decrement = () => {
    // NaN indicates invalid or empty input:
    if (isNaN(value)) {
      value = 0;
    }
    // use max() to be sure the result is >= min, even after any rounding.
    // if the result is unaligned with the step interval (due to user input), adjust it upward to be aligned.
    // this guarantees the input value is always valid after the user clicks -.
    let newValue = Math.max(value - step, min);
    if (newValue != min && (newValue - min) % step !== 0) {
      newValue += step - ((newValue - min) % step);
    }
    setErr(null);
    onValid(true);
    onChange(newValue);
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = parseInt(e.target.value, 10);
    if (isNaN(newValue)) {
      onValid(false);
      setErr('Value must be a number.');
    } else if (newValue < min || newValue > max) {
      onValid(false);
      setErr(`Value must be between ${min} and ${max}.`);
    } else if ((newValue - min) % step !== 0) {
      onValid(false);
      setErr(`Value must align with intervals of ${step}.`);
    } else {
      onValid(true);
      setErr(null);
    }
    onChange(newValue);
  };

  return (
    <div>
      <div className="flex items-center">
        <Input
          type="number"
          value={isNaN(value) ? '' : value}
          onChange={handleChange}
          className="z-10 w-16 rounded-r-none border-r-0"
          step={step}
        />
        <Button
          kind="secondary"
          appearance="outlined"
          onClick={decrement}
          disabled={isNaN(value) || value == min}
          icon={<RiSubtractFill className="h-4" />}
          className="disabled:border-muted disabled:bg-canvasBase rounded-none border-r-0"
        />
        <Button
          kind="secondary"
          appearance="outlined"
          onClick={increment}
          disabled={value == max}
          icon={<RiAddFill className="h-4" />}
          className="disabled:border-muted disabled:bg-canvasBase rounded-l-none border-l-0"
        />
        {err && (
          <p className="text-error ml-2 text-xs">
            <RiAlertFill className="-mt-0.5 inline h-4" /> {err}
          </p>
        )}
      </div>
    </div>
  );
}
