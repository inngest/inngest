import { forwardRef, useImperativeHandle, useMemo, useRef, useState } from 'react';
import { RiCloseFill, RiSearchLine } from '@remixicon/react';

import { cn } from '../utils/classNames';
import { Input, type InputProps } from './Input';

type SearchProps = Omit<InputProps, 'type'> & {
  // Use a custom handler instead of onChange to support clearing the value internally
  // when the parent element stores the value as state.
  onUpdate: (value: string) => void;
  value: string;
};

export const Search = forwardRef<HTMLInputElement, SearchProps>(
  ({ value, onUpdate, className, ...props }, ref) => {
    function clearInput() {
      onUpdate('');
    }
    return (
      <div className="relative">
        <RiSearchLine className="absolute bottom-0 left-2 top-0 my-auto h-4 w-4 text-[rgb(var(--color-border-muted))]" />
        <Input
          ref={ref}
          type="search"
          className={cn(className, 'px-7')}
          {...props}
          value={value}
          onChange={(e) => {
            // setInternalValue(e.target.value);
            onUpdate(e.target.value);
          }}
        />
        <button
          className={cn(
            'absolute bottom-0 right-2 top-0 my-auto text-[rgb(var(--color-border-muted))] hover:text-[rgb(var(--color-border-contrast))]',
            value.length ? 'block' : 'hidden'
          )}
          onClick={clearInput}
        >
          <RiCloseFill className="h-4 w-4" />
        </button>
      </div>
    );
  }
);
