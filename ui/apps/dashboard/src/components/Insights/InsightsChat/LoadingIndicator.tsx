'use client';

export const LoadingIndicator = ({ text = 'Thinking…' }: { text?: string }) => {
  return (
    <div className="text-text-subtle flex items-center justify-center p-4">
      <div className="size-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
      <span className="ml-2 text-sm">{text}</span>
    </div>
  );
};
