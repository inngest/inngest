import { type Route } from 'next';

import { Back } from './Back';
import { BreadCrumb } from './BreadCrumb';
import { HeaderTab } from './HeaderTab';

export type BreadCrumbType = {
  href?: string;
  text: string;
};

export type HeaderType = {
  tabs?: HeaderTab[];
  breadcrumb: BreadCrumbType[];
  infoIcon?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
  backNav?: boolean;
  loading?: boolean;
};

export const Header = ({
  tabs,
  breadcrumb,
  infoIcon: icon,
  action,
  className = '',
  backNav = false,
  loading,
}: HeaderType) => {
  return (
    <div className="border-subtle sticky top-0 z-50 flex flex-col justify-start border-b">
      <div
        className={`bg-canvasBase flex h-[52px] flex-row items-center justify-between px-3 ${className}`}
      >
        <div className="flex flex-row items-center justify-start align-baseline">
          {backNav && <Back className="mr-2" />}
          <BreadCrumb path={breadcrumb} />
          {icon}
        </div>
        <div>{action}</div>
      </div>
      {loading && (
        <span className="bg-secondary-xSubtle animate-underline absolute bottom-0 left-0 h-px w-0" />
      )}
      {tabs && (
        <div className="bg-canvasBase flex flex-row items-center justify-start space-x-3 px-4">
          {tabs.map(({ href, children, exactRouteMatch }) => (
            <HeaderTab key={href} href={href as Route} exactRouteMatch={exactRouteMatch}>
              {children}
            </HeaderTab>
          ))}
        </div>
      )}
    </div>
  );
};
