import { useEffect, useRef } from "react";
import Link from "next/link";
import { useInView } from "framer-motion";

import { useSectionStore } from "./SectionProvider";
import { Tag } from "./Tag";
import { remToPx } from "../../utils/remToPx";

function AnchorIcon(props) {
  return (
    <svg
      viewBox="0 0 20 20"
      fill="none"
      strokeLinecap="round"
      aria-hidden="true"
      {...props}
    >
      <path d="m6.5 11.5-.964-.964a3.535 3.535 0 1 1 5-5l.964.964m2 2 .964.964a3.536 3.536 0 0 1-5 5L8.5 13.5m0-5 3 3" />
    </svg>
  );
}

function Eyebrow({ tag, label }) {
  if (!tag && !label) {
    return null;
  }

  return (
    <div className="flex items-center gap-x-3">
      {tag && <Tag>{tag}</Tag>}
      {tag && label && (
        <span className="h-0.5 w-0.5 rounded-full bg-slate-300 dark:bg-slate-600" />
      )}
      {label && (
        <span className="font-mono text-xs text-slate-400">{label}</span>
      )}
    </div>
  );
}

function Anchor({ id, inView, children, className = "" }) {
  return (
    <Link
      href={`#${id}`}
      className={`group text-inherit no-underline hover:text-inherit ${className}`}
    >
      {inView && (
        <div className="absolute mt-1 ml-[calc(-1*var(--width))] hidden w-[var(--width)] opacity-0 transition [--width:calc(2.625rem+0.5px+50%-min(50%,calc(theme(maxWidth.lg)+theme(spacing.8))))] group-hover:opacity-100 group-focus:opacity-100 md:block lg:z-50 2xl:[--width:theme(spacing.10)]">
          <div className="group/anchor block h-5 w-5 rounded-lg bg-slate-50 ring-1 ring-inset ring-slate-300 transition hover:ring-slate-500 dark:bg-slate-800 dark:ring-slate-700 dark:hover:bg-slate-700 dark:hover:ring-slate-600">
            <AnchorIcon className="h-5 w-5 stroke-slate-500 transition dark:stroke-slate-400 dark:group-hover/anchor:stroke-white" />
          </div>
        </div>
      )}
      {children}
    </Link>
  );
}

type HeadingProps = {
  level: 1 | 2 | 3 | 4 | 5;
  children: React.ReactNode;
  id: string;
  tag?: string;
  label?: string;
  anchor?: boolean;
};

export function Heading({
  level = 2,
  children,
  id,
  tag,
  label,
  anchor = true,
  ...props
}: HeadingProps) {
  let Component: "h1" | "h2" | "h3" | "h4" | "h5" = `h${level}`;
  let ref = useRef();
  let registerHeading = useSectionStore((s) => s.registerHeading);

  let inView = useInView(ref, {
    margin: `${remToPx(-3.5)}px 0px 0px 0px`,
    amount: "all",
  });

  useEffect(() => {
    if (level === 2) {
      registerHeading({ id, ref, offsetRem: tag || label ? 8 : 6 });
    }
  });

  const hasAnchor = anchor && id;
  const flexClasses = `flex gap-4 items-center`;

  return (
    <>
      <Eyebrow tag={tag} label={label} />
      <Component
        ref={ref}
        id={anchor ? id : undefined}
        className={`${flexClasses} ${
          tag || label ? "mt-2 scroll-mt-32" : "scroll-mt-24"
        }`}
        {...props}
      >
        {hasAnchor ? (
          <Anchor id={id} inView={inView} className={flexClasses}>
            {children}
          </Anchor>
        ) : (
          children
        )}
      </Component>
    </>
  );
}
