import { IconExclamationTriangle } from '@inngest/components/icons/ExclamationTriangle';

import { cn } from '../utils/classNames';

const kindStyles = {
  outlined: 'dark:border-white/20 text-slate-600 dark:text-slate-300',
  error: 'bg-rose-600/40 border-none text-slate-300',
  solid: 'border-transparent',
};

type Props = {
  children?: React.ReactNode;
  className?: string;
  kind?: 'outlined' | 'error' | 'solid';

  /**
   * Use this when you want one of the sides to be flat. The other sides will be
   * rounded.
   */
  flatSide?: 'left' | 'right';
};

export function Badge({ children, className = '', kind = 'outlined', flatSide }: Props) {
  let roundedClasses = 'rounded-full';
  if (flatSide === 'left') {
    roundedClasses = 'rounded-r-full';
  } else if (flatSide === 'right') {
    roundedClasses = 'rounded-l-full';
  }

  return (
    <span
      className={cn(
        'box-border flex w-fit items-center gap-1 border px-3 py-1.5 text-xs leading-3',
        kindStyles[kind],
        roundedClasses,
        className
      )}
    >
      {kind === 'error' && <IconExclamationTriangle className="mt-0.5 h-3 w-3 text-rose-400" />}
      {children}
    </span>
  );
}
