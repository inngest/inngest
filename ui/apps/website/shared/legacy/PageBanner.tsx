import React from 'react';
import { ChevronRightIcon } from '@heroicons/react/20/solid';

type Props = {
  href: string;
  children: React.ReactNode;
  className?: string;
};

const PageBanner: React.FC<Props> = ({ href, children, className }) => (
  <a
    href={href}
    className={`page-banner flex w-full items-center justify-center gap-2.5 bg-[#A7F3D0] bg-gradient-to-r from-[#5EEAD4]
                via-[#A7F3D0] to-[#FDE68A] px-6 py-2 text-base
                font-bold tracking-tight text-slate-900
                transition-all hover:from-[#5EEAD4] hover:via-[#C9EEB5] hover:to-[#FDE68A] ${className}`}
  >
    {children}
    <span className="ml-2 inline-flex items-center text-slate-900">
      <span className="hidden sm:inline">Learn more </span>
      <ChevronRightIcon className="h-5" />
    </span>
  </a>
);

export default PageBanner;
