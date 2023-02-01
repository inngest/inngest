import classNames from "../../utils/classnames";

interface SidebarLinkProps {
  icon: React.ReactNode;
  active?: boolean;
  badge?: number;
  onClick?: () => void;
}

export default function SidebarLink({
  icon,
  active = false,
  badge = 0,
  onClick,
}: SidebarLinkProps) {
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
          ? `border-indigo-400`
          : `border-transparent opacity-40 hover:opacity-100`,
        `border-l-2 flex items-center justify-center w-full py-3 transition-all duration-150`
      )}
    >
      {icon}
    </button>
  );
}
