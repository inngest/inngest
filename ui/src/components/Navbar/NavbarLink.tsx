import classNames from "../../utils/classnames";

interface NavbarLinkProps {
  icon: React.ReactNode;
  active?: boolean;
  badge?: number;
  onClick?: () => void;
  tabName: string;
}

export default function NavebarLink({
  icon,
  active = false,
  badge = 0,
  onClick,
  tabName,
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
        `border-t-2 flex items-center justify-center w-full p-3 transition-all duration-150 gap-2`
      )}
    >
      {icon}
      {tabName}
    </button>
  );
}
