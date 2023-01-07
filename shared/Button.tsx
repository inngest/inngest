import classNames from "src/utils/classNames";
import ArrowRight from "./Icons/ArrowRight";
export function Button({
  children,
  variant = "primary",
  size = "md",
  arrow = false,
  ...props
}) {
  const sizes = {
    sm: "text-sm px-4 py-1.5",
    md: "text-sm px-6 py-2.5",
    lg: "text-lg px-8 py-4",
  };

  const variants = {
    primary: "text-white bg-indigo-500 hover:bg-indigo-400",
    secondary:
      "bg-slate-100 hover:bg-slate-300 text-slate-800 dark:bg-slate-800  dark:hover:bg-slate-700 dark:text-slate-100",
  };

  return (
    <a
      href={props.href}
      className={`group inline-flex items-center gap-0.5 rounded-full font-medium tracking-tight transition-all ${variants[variant]} ${sizes[size]} ${props.className}`}
    >
      {children}
      {arrow ? (
        <ArrowRight className="group-hover:translate-x-1 relative top-px transition-transform duration-150  -mr-1.5" />
      ) : null}
    </a>
  );
}
