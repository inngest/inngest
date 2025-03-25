import { RiShareCircleLine } from '@remixicon/react';

export function IconStatusSkipped({ className, title }: { className?: string; title?: string }) {
  return (
    <span title={title}>
      <RiShareCircleLine className={className ? className : 'h-6 w-6'} />
    </span>
  );
}
