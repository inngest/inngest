import React from "react";
import { ChevronRightIcon } from "@heroicons/react/20/solid";

type Props = {
  href: string;
  children: React.ReactNode;
  className?: string;
};

const PageBanner: React.FC<Props> = ({ href, children, className }) => (
  <a
    href={href}
    className={`page-banner flex items-center justify-center gap-2.5 w-full py-2 px-6 tracking-tight
                bg-[#A7F3D0] bg-gradient-to-r from-[#5EEAD4] via-[#A7F3D0] to-[#FDE68A]
                hover:from-[#5EEAD4] hover:via-[#C9EEB5] hover:to-[#FDE68A]
                text-base text-slate-900 font-bold transition-all ${className}`}
  >
    {children}
    <span className="inline-flex items-center ml-2 text-slate-900">
      <span className="hidden sm:inline">Learn more </span>
      <ChevronRightIcon className="h-5" />
    </span>
  </a>
);

export default PageBanner;
