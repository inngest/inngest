'use client';

interface QueryHelperPanelStaticMessageProps {
  children: React.ReactNode;
}

export function QueryHelperPanelStaticMessage({ children }: QueryHelperPanelStaticMessageProps) {
  return (
    <div className="text-subtle w-full cursor-default overflow-x-hidden truncate text-ellipsis whitespace-nowrap rounded px-2 py-1.5 text-left text-sm font-medium opacity-60">
      {children}
    </div>
  );
}
