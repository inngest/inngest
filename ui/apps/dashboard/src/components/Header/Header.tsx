import { type Route } from 'next';

import cn from '@/utils/cn';
import NavItem, { type ActiveMatching } from '../Navigation/NavItem';
import Navigation from '../Navigation/Navigation';

export type HeaderLink = {
  href: string;
  text: string;
  icon?: React.ReactNode;
  active?: ActiveMatching;
};

type HeaderTypes = {
  children?: React.ReactNode;
  links?: HeaderLink[];
  title: string | React.ReactNode;
  icon?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
  tag?: React.ReactNode;
};

export default function Header({
  children,
  links,
  title,
  icon,
  action,
  tag,
  className = '',
}: HeaderTypes) {
  return (
    <div className={cn('dark left-0 right-0 top-0 z-10 bg-slate-900', className)}>
      <div className="flex items-center justify-between px-6">
        <div>
          <div className="flex items-center">
            {icon ? (
              <div className="mr-2 flex h-8 w-8 items-center justify-center rounded-md border border-slate-800">
                {icon}
              </div>
            ) : (
              ''
            )}
            <h1 className="py-3 text-lg font-medium tracking-wide text-white">{title}</h1>
            <span className="pl-4">{tag}</span>
          </div>
          {links && (
            <Navigation className="-ml-2 -mt-2">
              {links.map(({ href, text, icon, ...props }) => (
                <NavItem key={href} href={href as Route} text={text} icon={icon} {...props} />
              ))}
            </Navigation>
          )}
        </div>
        <div>{action}</div>
      </div>
      <div>{children}</div>
    </div>
  );
}
