'use client';

import { useState } from 'react';
import * as Popover from '@radix-ui/react-popover';
import * as chrono from 'chrono-node';
import { useDebounce } from 'react-use';

import Input from '@/components/Forms/Input';

type Props = {
  onChange: (newDateTime: Date) => void;
  required?: boolean;
};

export function TimeInput({ onChange, required }: Props) {
  // TODO: This component's state management can probably be improved by using useReducer()
  const [inputString, setInputString] = useState<string>('');
  const [parsedDateTime, setParsedDateTime] = useState<Date>();
  useDebounce(
    () => {
      if (status === 'selected') return;
      const parsedDateTime = chrono.parseDate(inputString);
      if (!parsedDateTime) {
        setStatus('invalid');
        return;
      }
      setStatus('valid');
      setParsedDateTime(parsedDateTime);
    },
    350,
    [inputString]
  );
  const [status, setStatus] = useState<'typing' | 'invalid' | 'valid' | 'selected'>('invalid');

  function onInputChange(event: React.ChangeEvent<HTMLInputElement>) {
    setStatus('typing');
    setParsedDateTime(undefined);
    setInputString(event.target.value);
  }

  function applyDateTime(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.code === 'Enter' && parsedDateTime && inputString.length > 0) {
      event.preventDefault();
      onChange(parsedDateTime);
      setInputString(parsedDateTime.toLocaleString());
      setStatus('selected');
    }
  }

  const isPopoverOpen = status === 'valid';

  return (
    <Popover.Root open={isPopoverOpen}>
      <Popover.Anchor>
        <Input
          type="text"
          value={inputString}
          onChange={onInputChange}
          onKeyDown={applyDateTime}
          required={required}
        />
      </Popover.Anchor>
      <Popover.Portal>
        <Popover.Content
          className="shadow-floating z-[100] inline-flex items-center gap-2 space-y-4 rounded-md bg-white/95 p-2 text-sm text-slate-800 ring-1 ring-black/5 backdrop-blur-[3px]"
          sideOffset={5}
          onOpenAutoFocus={(event) => event.preventDefault()}
        >
          {parsedDateTime?.toLocaleString()}
          <kbd
            className="ml-auto flex h-6 w-6 items-center justify-center rounded bg-slate-100 p-2 font-sans text-xs"
            aria-label="Press Enter to apply the parsed date and time."
          >
            ↵
          </kbd>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
