'use client';

import { useCallback, useEffect, useState } from 'react';
import { CheckIcon, ClipboardIcon } from '@heroicons/react/20/solid';
import { useCopyToClipboard } from 'react-use';
import { toast } from 'sonner';

import Button from '@/components/Button';

type Props = {
  value: string;
  maskedValue?: string;
  label: string;
};

export function KeyBox({ value, maskedValue, label }: Props) {
  const [isCopied, setCopied] = useState(false);
  const [showKey, setShowKey] = useState(false);
  const [, copy] = useCopyToClipboard();

  const handleCopy = useCallback(() => {
    copy(value || '');
    setCopied(true);
    toast.success(`${label} copied`);
  }, [value, copy, label]);

  useEffect(() => {
    if (isCopied) {
      const timeout = setTimeout(() => setCopied(false), 3000);
      return () => clearTimeout(timeout);
    }
  }, [isCopied, handleCopy]);

  return (
    <div className="inline-flex flex-col rounded-md bg-slate-50">
      <div className="flex flex-row overflow-hidden rounded-t-md">
        <Button
          aria-label="Copy key to clipboard"
          iconSide="right"
          onClick={() => {
            setShowKey(!showKey);
            handleCopy();
          }}
          variant="text"
          title="Click to reveal"
          className="h-10 min-w-[690px] justify-start gap-4 rounded-none bg-slate-100 px-4 font-mono font-medium text-indigo-600 hover:bg-indigo-100 hover:no-underline"
        >
          {showKey ? value : maskedValue}
        </Button>
        <Button
          aria-label="Copy key to clipboard"
          iconSide="right"
          onClick={handleCopy}
          variant="text"
          title="Click to copy"
          className="h-10 rounded-none bg-slate-100 px-4 font-medium text-indigo-600 hover:bg-indigo-100 hover:no-underline"
        >
          {isCopied ? <CheckIcon className="w-4" /> : <ClipboardIcon className="w-4" />}
        </Button>
      </div>
      <h3 className="rounded-b-md bg-slate-50 px-4 py-2 text-xs font-semibold text-slate-500">
        {label}
      </h3>
    </div>
  );
}
