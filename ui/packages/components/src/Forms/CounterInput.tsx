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
  step?: number;
};

export default function CounterInput({
  value = 0,
  min = 0,
  max = 100,
  onChange,
  step = 1,
}: CounterInputProps) {
  const [err, setErr] = useState<string | null>(null);

  const increment = () => {
    const newValue = value + step;
    if (newValue <= max) {
      setErr(null);
      onChange(newValue);
    }
  };

  const decrement = () => {
    const newValue = value - step;
    if (newValue >= min) {
      setErr(null);
      onChange(newValue);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = parseInt(e.target.value, 10);

    if (isNaN(newValue)) {
      setErr('Value must be a number.');
      return;
    }
    if (newValue < min || newValue > max) {
      setErr(`Value must be between ${min} and ${max}.`);
      return;
    }
    if ((newValue - min) % step !== 0) {
      setErr(`Value must align with intervals of ${step}.`);
      return;
    }

    setErr(null);
    onChange(newValue);
  };

  return (
    <div>
      <div className="flex items-center">
        <Input
          type="number"
          value={value}
          onChange={handleChange}
          className="z-10 w-12 rounded-r-none border-r-0"
          step={step}
        />
        <Button
          kind="secondary"
          appearance="outlined"
          onClick={decrement}
          disabled={value - step < min}
          icon={<RiSubtractFill className="h-4" />}
          className="disabled:border-muted disabled:bg-canvasBase rounded-none border-r-0"
        />
        <Button
          kind="secondary"
          appearance="outlined"
          onClick={increment}
          disabled={value + step > max}
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
