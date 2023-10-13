import { IconExclamationTriangle } from '@inngest/components/icons/ExclamationTriangle';

const kindStyles = {
  outlined: 'border-white/20 text-slate-300',
  error: 'bg-rose-600/40 border-none text-slate-300',
  solid: 'border-transparent',
};

export function Badge({
  children,
  className = '',
  kind = 'outlined',
}: {
  children: React.ReactNode;
  className?: string;
  kind?: 'outlined' | 'error' | 'solid';
}) {
  const classNames = `text-xs leading-3 border rounded-md  box-border py-1.5 px-2 flex items-center gap-1 w-fit ${kindStyles[kind]} ${className}`;

  return (
    <span className={classNames}>
      {kind === 'error' && <IconExclamationTriangle className="mt-0.5 h-3 w-3 text-rose-400" />}
      {children}
    </span>
  );
}
