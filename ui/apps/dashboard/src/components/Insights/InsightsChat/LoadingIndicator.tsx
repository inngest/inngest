'use client';

import { RiCodeSSlashLine } from '@remixicon/react';

export const LoadingIndicator = ({ text = 'Thinking…' }: { text?: string }) => {
  return (
    <div className="text-text-subtle flex items-center justify-start p-4">
      <RiCodeSSlashLine className="text-muted h-4 w-4" />
      <span className="relative ml-2 inline-block text-sm">
        {text}
        <span
          aria-hidden
          className="pointer-events-none absolute inset-0 animate-[shimmer-text_1.25s_linear_infinite] bg-gradient-to-r from-transparent via-white/70 to-transparent bg-clip-text text-transparent [background-size:300%_100%]"
        >
          {text}
        </span>
      </span>
    </div>
  );
};
