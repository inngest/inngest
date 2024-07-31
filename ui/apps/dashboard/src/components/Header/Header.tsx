import { type Route } from 'next';

import NavItem, { type ActiveMatching } from '@/components/Navigation/old/NavItem';
import Navigation from '@/components/Navigation/old/Navigation';
import { BreadCrumb } from './BreadCrumb';

export type HeaderLink = {
  href: string;
  text: string;
  icon?: React.ReactNode;
  active?: ActiveMatching | boolean;
  badge?: React.ReactNode;
};

export type BreadCrumbType = {
  href?: string;
  text: string;
};

export type HeaderType = {
  links?: HeaderLink[];
  breadcrumb: BreadCrumbType[];
  icon?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
};

export const Header = ({ links, breadcrumb, icon, action, className = '' }: HeaderType) => {
  return (
    <>
      <div
        className={`bg-canvasBase border-subtle flex h-[52px] flex-row items-center justify-between border-b p-4 ${className}`}
      >
        <div className="flex flex-row items-center justify-start align-baseline">
          <BreadCrumb path={breadcrumb} />
          {icon}
        </div>
        <div>{action}</div>
      </div>
      {links && (
        <Navigation className="-ml-2 -mt-2">
          {links.map(({ href, text, ...props }) => (
            <NavItem key={href} href={href as Route} text={text} {...props} />
          ))}
        </Navigation>
      )}
    </>
  );
};
