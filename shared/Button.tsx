import { AnchorHTMLAttributes } from "react";
import classNames from "src/utils/classNames";
import ArrowRight from "./Icons/ArrowRight";

type ButtonProps = {
  variant?: "primary" | "secondary" | "tertiary";
  size?: "sm" | "md" | "lg";
  className?: string;
  arrow?: "left" | "right";
  full?: boolean;
  children?: React.ReactNode;
} & AnchorHTMLAttributes<HTMLAnchorElement>;

export function Button({
  children,
  variant = "primary",
  size = "md",
  arrow,
  full = false,
  target = "",
  ...props
}: ButtonProps) {
  const sizes = {
    sm: "text-sm px-4 py-1.5",
    md: "text-sm px-6 py-2.5",
    lg: "text-lg px-8 py-4",
  };

  const variants = {
    primary: "text-white bg-indigo-500 hover:bg-indigo-400",
    secondary: "bg-slate-800/80 hover:bg-slate-700/80 text-white",
    tertiary:
      "bg-slate-100 hover:bg-slate-300 text-slate-800 dark:bg-slate-800  dark:hover:bg-slate-700 dark:text-slate-100",
  };

  const width = full ? "w-full" : "";

  return (
    <a
      target={target}
      href={props.href}
      className={`button group inline-flex items-center justify-center gap-0.5 rounded-lg font-medium tracking-tight transition-all ${variants[variant]} ${sizes[size]} ${props.className} ${width}`}
    >
      {arrow && arrow === "left" ? (
        <ArrowRight className="group-hover:-translate-x-1 transition-transform rotate-180 duration-150 -ml-1.5" />
      ) : null}
      {children}
      {arrow && arrow === "right" ? (
        <ArrowRight className="group-hover:translate-x-1 transition-transform duration-150  -mr-1.5" />
      ) : null}
    </a>
  );
}
