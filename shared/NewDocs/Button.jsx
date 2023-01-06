import Link from "next/link";
import clsx from "clsx";

function ArrowIcon(props) {
  return (
    <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" {...props}>
      <path
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        d="m11.5 6.5 3 3.5m0 0-3 3.5m3-3.5h-9"
      />
    </svg>
  );
}

const variantStyles = {
  primary:
    "rounded-full bg-slate-900 py-1 px-3 text-white hover:bg-slate-700 dark:bg-indigo-500 dark:text-indigo-000 dark:ring-1 dark:ring-inset dark:ring-indigo-400/20 dark:hover:bg-indigo-400/10 dark:hover:text-indigo-300 dark:hover:ring-indigo-300",
  secondary:
    "rounded-full bg-slate-100 py-1 px-3 text-slate-900 hover:bg-slate-200 dark:bg-slate-800/40 dark:text-slate-400 dark:ring-1 dark:ring-inset dark:ring-slate-800 dark:hover:bg-slate-800 dark:hover:text-slate-300",
  filled:
    "rounded-full bg-slate-900 py-1 px-3 text-white hover:bg-slate-700 dark:bg-indigo-500 dark:text-white dark:hover:bg-indigo-400",
  outline:
    "rounded-full py-1 px-3 text-slate-700 ring-1 ring-inset ring-slate-900/10 hover:bg-slate-900/2.5 hover:text-slate-900 dark:text-slate-400 dark:ring-white/10 dark:hover:bg-white/5 dark:hover:text-white",
  text: "text-indigo-500 hover:text-indigo-600 dark:text-indigo-400 dark:hover:text-indigo-500",
};

export function Button({
  variant = "primary",
  className,
  children,
  arrow,
  ...props
}) {
  let Component = props.href ? Link : "button";

  className = clsx(
    "inline-flex items-center gap-0.5 justify-center overflow-hidden text-sm font-medium transition",
    variantStyles[variant],
    className
  );

  let arrowIcon = (
    <ArrowIcon
      className={clsx(
        "mt-0.5 h-5 w-5",
        variant === "text" && "relative top-px",
        arrow === "left" && "-ml-1 rotate-180",
        arrow === "right" && "-mr-1"
      )}
    />
  );

  return (
    <Component className={className} {...props}>
      {arrow === "left" && arrowIcon}
      {children}
      {arrow === "right" && arrowIcon}
    </Component>
  );
}
