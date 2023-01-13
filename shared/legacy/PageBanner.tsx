import React from "react";

type Props = {
  href: string;
  children: React.ReactNode;
};

const PageBanner: React.FC<Props> = ({ href, children }) => (
  <a
    href={href}
    className="page-banner bg-indigo-500 text-sm block text-center w-full py-2 text-white font-medium hover:bg-indigo-600 transition-all"
  >
    {children}
    <span className="text-white"> &rsaquo;</span>
  </a>
);

export default PageBanner;
