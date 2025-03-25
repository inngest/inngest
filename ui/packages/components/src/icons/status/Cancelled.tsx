import { RiCloseCircleLine } from '@remixicon/react';

export function IconStatusCancelled({ className, title }: { className?: string; title?: string }) {
  return (
    <span title={title}>
      <RiCloseCircleLine className={className ? className : 'h-6 w-6'} />
    </span>
  );
}
