import classNames from "../utils/classnames";

interface ButtonProps {
  kind?: "primary" | "secondary" | "text";
  label?: React.ReactNode;
  icon?: React.ReactNode;
  btnAction?: () => void;
}

export default function Button({
  label,
  icon,
  btnAction,
  kind = "primary",
}: ButtonProps) {
  return (
    <button
      className={classNames(
        "flex gap-1.5 items-center border text-xs rounded-sm px-2.5 py-1 text-slate-100 transition-all duration-150",
        kind === "primary"
          ? "bg-slate-700/50 border-slate-700/50 hover:bg-slate-700/80"
          : kind === "text"
          ? "text-slate-400 border-transparent hover:text-white"
          : "bg-slate-800/20 border-slate-700/50 hover:bg-slate-800/40"
      )}
      onClick={btnAction}
    >
      {label && label}
      {icon && icon}
    </button>
  );
}
