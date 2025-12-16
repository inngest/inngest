import { Button } from '@inngest/components/Button';
import { InngestLogo } from '@inngest/components/icons/logos/InngestLogo';
import { InngestLogoSmall } from '@inngest/components/icons/logos/InngestLogoSmall';
import { RiContractLeftLine, RiContractRightLine } from '@remixicon/react';

import { useInfoQuery } from '@/store/devApi';
import { Link } from '@tanstack/react-router';

type LogoProps = {
  collapsed: boolean;
  setCollapsed: (arg: boolean) => void;
};

const NavToggle = ({ collapsed, setCollapsed }: LogoProps) => {
  const toggle = async () => {
    const toggled = !collapsed;
    setCollapsed(toggled);
    typeof window !== 'undefined' &&
      window.localStorage.setItem('navCollapsed', toggled ? 'true' : 'false');
  };

  return (
    <Button
      kind="primary"
      appearance="ghost"
      onClick={toggle}
      className={'hidden group-hover:block'}
      icon={
        collapsed ? (
          <RiContractRightLine className="text-muted h-5 w-5" />
        ) : (
          <RiContractLeftLine className="text-muted h-5 w-5" />
        )
      }
    />
  );
};

export default function Logo({ collapsed, setCollapsed }: LogoProps) {
  const { data: info, isLoading, error } = useInfoQuery();
  const isDevServer = error ? false : !info?.isSingleNodeService;

  return (
    <div
      className={`my-4 flex h-[28px] w-full flex-row items-center ${
        collapsed ? 'justify-center' : 'mx-4 justify-start'
      }`}
    >
      <div
        className={`flex flex-row items-center justify-start ${
          collapsed ? '' : 'mr-1.5'
        } `}
      >
        {collapsed ? (
          <div className="cursor-pointer group-hover:hidden">
            <InngestLogoSmall className="text-basis" />
          </div>
        ) : (
          <div className="flex flex-row items-center justify-start">
            <Link to="/">
              <InngestLogo className="text-basis mr-1.5" width={96} />
            </Link>
            <span className="text-primary-intense text-[11px] font-medium leading-none">
              {isDevServer ? 'DEV SERVER' : 'SERVER'}
            </span>
          </div>
        )}
      </div>
      <NavToggle collapsed={collapsed} setCollapsed={setCollapsed} />
    </div>
  );
}
