import classNames from '../utils/classnames';

interface ButtonProps {
  kind?: 'primary' | 'secondary' | 'text';
  label?: React.ReactNode;
  icon?: React.ReactNode;
  disabled?: boolean;
  type?: 'submit' | 'button';
  btnAction?: (e?: React.MouseEvent) => void;
}

export default function Button({
  label,
  icon,
  disabled,
  btnAction,
  kind = 'primary',
  type,
}: ButtonProps) {
  return (
    <button
      className={classNames(
        'flex gap-1.5 items-center border text-xs rounded-sm px-2.5 py-1 text-slate-100 transition-all duration-150 disabled:text-slate-500',
        kind === 'primary'
          ? 'bg-slate-700/50 border-slate-700/50 hover:bg-slate-700/80 disabled:hover:bg-slate-700/50'
          : kind === 'text'
          ? 'text-slate-400 border-transparent hover:text-white disabled:hover:text-slate-40'
          : 'bg-slate-800/20 border-slate-700/50 hover:bg-slate-800/40 disabled:hover:bg-slate-800/20',
      )}
      type={type}
      onClick={btnAction}
      disabled={disabled}
    >
      {label && label}
      {icon && <span className="icon-xs">{icon}</span>}
    </button>
  );
}
