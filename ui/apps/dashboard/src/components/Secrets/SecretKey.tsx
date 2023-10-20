'use client';

import { useState } from 'react';
import { ClipboardIcon } from '@heroicons/react/20/solid';
import { CheckIcon } from '@heroicons/react/24/outline';
import { useCopyToClipboard } from 'react-use';
import { toast } from 'sonner';

import cn from '@/utils/cn';

type SecretKeyProps = {
  value: string;
  masked: string;
  className?: string;
  context?: 'dark' | 'light';
};

const contextStyles = {
  light: 'text-slate-700 border-slate-300 bg-slate-50 hover:bg-slate-100 font-medium',
  dark: 'text-white border-slate-700 bg-slate-800 text-white font-medium hover:bg-slate-600/50',
};

const buttonContextStyles = {
  light: 'bg-white border-l border-slate-200 text-slate-700 hover:text-indigo-500',
  dark: 'text-white bg-slate-700 hover:text-indigo-400',
};

export default function SecretKey({ value, masked, className, context = 'light' }: SecretKeyProps) {
  const [showKey, setShowKey] = useState<boolean>(false);
  const [clipboardState, copy] = useCopyToClipboard();

  function onCopy() {
    copy(value);
    toast.message(
      <>
        <ClipboardIcon className="h-3" /> Copied to clipboard!
      </>
    );
  }

  return (
    <div
      className={cn(
        `flex cursor-pointer overflow-hidden rounded-lg border font-mono text-xs transition-all `,
        contextStyles[context],
        className
      )}
    >
      <div className="grow truncate py-2 pl-3 pr-1.5" onClick={() => setShowKey(!showKey)}>
        {showKey ? value : masked}
      </div>
      <button className={cn(`px-2 transition-all`, buttonContextStyles[context])} onClick={onCopy}>
        {clipboardState.value ? <CheckIcon className="w-4" /> : <ClipboardIcon className="w-4" />}
      </button>
    </div>
  );
}
