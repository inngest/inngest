'use client';

import React, { useState } from 'react';
import { RiAddFill, RiSubtractFill } from '@remixicon/react';

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
    <div className="">
      <div className="relative flex  items-center">
        <Input
          type="number"
          value={value}
          onChange={handleChange}
          className="z-10 w-12 rounded-r-none border-r-0"
        />
        <Button
          kind="secondary"
          appearance="outlined"
          onClick={decrement}
          disabled={value <= min}
          icon={<RiSubtractFill className="h-4" />}
          className="disabled:border-muted disabled:bg-canvasBase rounded-none border-r-0"
        />
        <Button
          kind="secondary"
          appearance="outlined"
          onClick={increment}
          disabled={value >= max}
          icon={<RiAddFill className="h-4" />}
          className="disabled:border-muted disabled:bg-canvasBase rounded-l-none border-l-0"
        />
        {/* <p className="text-error absolute top-8 text-xs">Value not available</p> */}
      </div>
    </div>
  );
}
