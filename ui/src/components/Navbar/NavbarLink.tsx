import classNames from '@/utils/classnames';
import Badge from '@/components/Badge';

interface NavbarLinkProps {
  icon: React.ReactNode;
  active?: boolean;
  badge?: number;
  hasError?: boolean;
  onClick?: () => void;
  tabName: string;
}

export default function NavebarLink({
  icon,
  active = false,
  badge,
  onClick,
  tabName,
  hasError,
}: NavbarLinkProps) {
  return (
    <button
      onClick={
        onClick
          ? (e) => {
              e.preventDefault();
              onClick();
            }
          : undefined
      }
      className={classNames(
        active
          ? `border-indigo-400 text-white`
          : `border-transparent text-slate-400 hover:text-white`,
        `border-t-2 flex items-center justify-center w-full px-3 leading-[2.75rem] transition-all duration-150 gap-2`
      )}
    >
      {icon}
      {tabName}
      {typeof badge === 'number' && <Badge kind={hasError ? 'error' : 'outlined'}>{badge.toString()}</Badge>}
    </button>
  );
}
