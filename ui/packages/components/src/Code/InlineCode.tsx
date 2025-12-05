'use client';

export function InlineCode({ children }: { children: React.ReactNode }) {
  return (
    <code className="bg-canvasMuted inline-flex items-center rounded-sm px-1 py-0.5 font-mono text-xs font-medium tracking-tight">
      {children}
    </code>
  );
}
