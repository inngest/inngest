'use client';

import { useCallback, useEffect, useState } from 'react';
import { CheckIcon, ClipboardIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import { useCopyToClipboard } from 'react-use';
import { toast } from 'sonner';

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
    <div className="inline-flex w-full flex-col rounded-md bg-slate-50">
      <div className="flex flex-row overflow-hidden rounded-t-md">
        <Button
          aria-label="Copy key to clipboard"
          iconSide="right"
          btnAction={() => {
            setShowKey(!showKey);
            handleCopy();
          }}
          appearance="text"
          title="Click to reveal"
          className="min-w-[600px]	flex-1 !justify-start gap-4 rounded-none px-4 font-mono font-medium !text-indigo-600 hover:bg-indigo-100 hover:no-underline"
          label={showKey ? value : maskedValue}
        />
        <Button
          aria-label="Copy key to clipboard"
          iconSide="right"
          btnAction={handleCopy}
          appearance="text"
          kind="primary"
          icon={isCopied ? <CheckIcon /> : <ClipboardIcon />}
          title="Click to copy"
        />
      </div>
      <h3 className="rounded-b-md bg-slate-100 px-4 py-2 text-xs font-semibold text-slate-500">
        {label}
      </h3>
    </div>
  );
}
