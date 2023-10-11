import cn from '@/utils/cn';

export type StatusTypeKind = 'success' | 'error' | 'warning' | 'info';

type StatusTypes = {
  kind: StatusTypeKind;
  children: React.ReactNode;
  className?: string;
};

const kindStyles = {
  success: 'bg-teal-500',
  error: 'bg-red-500',
  warning: 'bg-yellow-500',
  info: 'bg-indigo-500',
};

export default function Status({ kind, children, className }: StatusTypes) {
  return (
    <span
      className={cn('flex items-center gap-1.5 text-xs font-medium text-slate-600 ', className)}
    >
      <span className={cn('h-2 w-2 rounded-full', kindStyles[kind])} />
      {children}
    </span>
  );
}
