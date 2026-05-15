import { RiLoader2Line } from '@remixicon/react';

export const LoadingIndicator = ({ text = 'Thinking…' }: { text?: string }) => {
  return (
    <div className="text-subtle flex items-center justify-start py-4">
      <RiLoader2Line className="text-light h-4 w-4 animate-spin duration-1000" />
      <span className="relative ml-2 inline-block text-sm">
        {text}
        <span
          aria-hidden
          className="pointer-events-none absolute inset-0 animate-[shimmer-text_1.25s_linear_infinite] bg-gradient-to-r from-transparent via-[rgb(var(--color-foreground-light)/0.8)] to-transparent bg-clip-text font-normal text-transparent [background-size:300%_100%]"
        >
          {text}
        </span>
      </span>
    </div>
  );
};
