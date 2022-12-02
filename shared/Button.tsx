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
      className="group inline-flex items-center gap-0.5 rounded-full text-sm font-medium pl-4 pr-4 py-1  bg-indigo-500 tracking-tight hover:bg-indigo-400 transition-all text-white"
    >
      {children}
      {arrow ? (
        <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
      ) : null}
    </a>
  );
}
