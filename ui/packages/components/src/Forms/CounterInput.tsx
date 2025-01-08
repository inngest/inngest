'use client';

import React, { useState } from 'react';

import { Button } from '../Button';
import { Input } from './Input';

type CounterInputProps = {
  initialValue?: number;
  min?: number;
  max?: number;
};

export default function CounterInput({ initialValue = 0, min = 0, max = 100 }: CounterInputProps) {
  const [value, setValue] = useState(initialValue);

  const increment = () => {
    if (value < max) {
      setValue(value + 1);
    }
  };

  const decrement = () => {
    if (value > min) {
      setValue(value - 1);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = parseInt(e.target.value, 10);
    if (!isNaN(newValue) && newValue >= min && newValue <= max) {
      setValue(newValue);
    }
  };

  return (
    <div className="flex items-center">
      <Button
        kind="secondary"
        appearance="outlined"
        onClick={decrement}
        disabled={value <= min}
        label="-"
        className="rounded-r-none"
      />
      <Input
        type="number"
        value={value}
        onChange={handleChange}
        className="z-10 w-12 rounded-none border-l-0 border-r-0"
      />
      <Button
        kind="secondary"
        appearance="outlined"
        onClick={increment}
        disabled={value >= max}
        label="+"
        className="rounded-l-none"
      />
    </div>
  );
}
