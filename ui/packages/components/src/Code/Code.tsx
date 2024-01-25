'use client';

export function Code({ children }: { children: React.ReactNode }) {
  return (
    <code className="inline-flex rounded bg-slate-100 px-1 font-mono text-xs font-medium tracking-tight">
      {children}
    </code>
  );
}
