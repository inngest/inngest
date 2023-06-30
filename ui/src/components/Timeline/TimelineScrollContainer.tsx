'use client';

export function TimelineScrollContainer({ children }) {
  return (
    <ul className="bg-slate-950/50 border-r border-slate-800/40 overflow-y-scroll relative py-4 pr-2.5 w-96 h-full">
      {children}
    </ul>
  );
}
