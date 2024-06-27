import { RiErrorWarningLine } from '@remixicon/react';

export function IconStatusFailed({ className, title }: { className?: string; title?: string }) {
  return (
    <span title={title}>
      <RiErrorWarningLine className={className ? className : 'h-6 w-6'} />
    </span>
  );
}
