'use client';

import Link from 'next/link';
import { NewButton } from '@inngest/components/Button';
import { InngestLogo } from '@inngest/components/icons/logos/InngestLogo';
import { InngestLogoSmallBW } from '@inngest/components/icons/logos/InngestLogoSmall';
import { RiContractLeftLine, RiContractRightLine } from '@remixicon/react';

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
    <NewButton
      kind="primary"
      appearance="ghost"
      onClick={toggle}
      className={'hidden group-hover:block'}
      icon={
        collapsed ? (
          <RiContractRightLine className="text-subtle h-5 w-5" />
        ) : (
          <RiContractLeftLine className="text-subtle h-5 w-5" />
        )
      }
    />
  );
};

export default function Logo({ collapsed, setCollapsed }: LogoProps) {
  return (
    <div
      className={`my-5 flex h-10 w-full flex-row items-center ${
        collapsed ? 'justify-center' : 'mx-4 justify-start'
      }`}
    >
      <div className={`flex flex-row items-center justify-start ${collapsed ? '' : 'mr-1.5'} `}>
        {collapsed ? (
          <div className="cursor-pointer group-hover:hidden">
            <InngestLogoSmallBW className="text-basis" />
          </div>
        ) : (
          <div className="flex flex-row items-center justify-start">
            <Link href="/">
              <InngestLogo className="text-basis mr-1.5" width={92} />
            </Link>
            <span className="text-primary-intense text-[11px] leading-none">DEV SERVER</span>
          </div>
        )}
      </div>
      <NavToggle collapsed={collapsed} setCollapsed={setCollapsed} />
    </div>
  );
}