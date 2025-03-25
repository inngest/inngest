import { RiCheckboxCircleLine } from '@remixicon/react';

export function IconStatusCompleted({ className, title }: { className?: string; title?: string }) {
  return (
    <span title={title}>
      <RiCheckboxCircleLine className={className ? className : 'h-6 w-6'} />
    </span>
  );
}
