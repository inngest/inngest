import { IconExclamationTriangleSolid } from '@/icons';

const kindStyles = {
  outlined: 'border-white/20 text-slate-300',
  error: 'bg-rose-600/40 border-none text-slate-300',
};

export default function Badge({
  children,
  className = '',
  kind = 'outlined',
}: {
  children: React.ReactNode;
  className?: string;
  kind?: 'outlined' | 'error';
}) {
  const classNames = `text-xs leading-3 border rounded-md  box-border py-1.5 px-2 flex items-center gap-1 ${kindStyles[kind]} ${className}`;

  return (
    <span className={classNames}>
      {kind === 'error' && (
        <IconExclamationTriangleSolid className="text-rose-400 w-3 h-3" />
      )}
      {children}
    </span>
  );
}
