'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { CopyButton } from '@inngest/components/CopyButton';
import { useCopyToClipboard } from '@inngest/components/hooks/useCopyToClipboard';
import { classNames } from '@inngest/components/utils/classNames';

type CodeKeyProps = {
  fullKey: string;
  maskedKey?: string;
  label?: string;
  className?: string;
};

export function CodeKey({ fullKey, maskedKey, label, className }: CodeKeyProps) {
  const [showKey, setShowKey] = useState(false);
  const { handleCopyClick, isCopying } = useCopyToClipboard();

  return (
    <div className="rounded-md bg-slate-50 dark:bg-slate-800">
      <div
        className={classNames(className, 'flex items-center justify-between rounded-t-md text-sm')}
      >
        <Button
          className="!block flex-1 truncate text-left font-mono"
          appearance="text"
          btnAction={() => {
            setShowKey(!showKey);
            handleCopyClick(fullKey);
          }}
          title={showKey ? 'Click to hide' : 'Click to reveal'}
          label={showKey ? fullKey : maskedKey + '-<click-to-reveal>'}
          kind="primary"
        />
        <CopyButton
          code={fullKey}
          iconOnly={true}
          isCopying={isCopying}
          handleCopyClick={handleCopyClick}
        />
      </div>
      {label && (
        <h3 className="rounded-b-md bg-slate-100 p-2.5 text-xs font-semibold text-slate-500 dark:bg-slate-900">
          {label}
        </h3>
      )}
    </div>
  );
}
