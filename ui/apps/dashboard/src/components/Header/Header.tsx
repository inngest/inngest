import { type Route } from 'next';

import { BreadCrumb } from './BreadCrumb';
import { HeaderTab } from './HeaderTab';

export type BreadCrumbType = {
  href?: string;
  text: string;
};

export type HeaderType = {
  tabs?: HeaderTab[];
  breadcrumb: BreadCrumbType[];
  icon?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
};

export const Header = ({ tabs, breadcrumb, icon, action, className = '' }: HeaderType) => {
  return (
    <div className="flex flex-col justify-start border-b">
      <div
        className={`bg-canvasBase border-subtle flex h-[52px] flex-row items-center justify-between p-4 ${className}`}
      >
        <div className="flex flex-row items-center justify-start align-baseline">
          <BreadCrumb path={breadcrumb} />
          {icon}
        </div>
        <div>{action}</div>
      </div>
      {tabs && (
        <div className="flex h-[30px] flex-row items-center justify-start space-x-3 px-4">
          {tabs.map(({ href, text, exactRouteMatch }) => (
            <HeaderTab
              key={href}
              href={href as Route}
              text={text}
              exactRouteMatch={exactRouteMatch}
            />
          ))}
        </div>
      )}
    </div>
  );
};
