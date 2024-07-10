import { RiPauseCircleLine } from '@remixicon/react';

export function IconStatusPaused({ className, title }: { className?: string; title?: string }) {
  return (
    <span title={title}>
      <RiPauseCircleLine className={className ? className : 'h-6 w-6'} />
    </span>
  );
}
