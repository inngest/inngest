import classNames from "src/utils/classNames";
import ArrowRight from "./Icons/ArrowRight";
export default function Button({
  children,
  kind = "primary",
  arrow = false,
  ...props
}) {
  return (
    <a
      href={props.href}
      className={classNames(
        kind === "primary"
          ? `bg-indigo-500  hover:bg-indigo-400`
          : `bg-slate-800  hover:bg-slate-700`,
        `group inline-flex items-center gap-0.5 rounded-full text-sm font-medium pl-4 pr-4 py-2.5   tracking-tight transition-all text-white ${props.className}`
      )}
    >
      {children}
      {arrow ? (
        <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
      ) : null}
    </a>
  );
}
