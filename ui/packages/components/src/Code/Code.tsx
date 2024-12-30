'use client';

export function Code({ children }: { children: React.ReactNode }) {
  return (
    <code className="bg-canvasSubtle inline-flex rounded px-1 font-mono text-xs font-medium tracking-tight">
      {children}
    </code>
  );
}
