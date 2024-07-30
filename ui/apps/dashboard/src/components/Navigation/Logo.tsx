'use client';

import type { Route } from 'next';
import Link from 'next/link';
import { InngestLogo } from '@inngest/components/icons/logos/InngestLogo';
import { InngestLogoSmallBW } from '@inngest/components/icons/logos/InngestLogoSmall';
import { RiContractLeftLine, RiContractRightLine } from '@remixicon/react';

import Search from './Search';

type LogoProps = {
  collapsed: boolean;
  setCollapsed: (arg: boolean) => void;
};

const NavToggle = ({ collapsed, setCollapsed }: LogoProps) => {
  const toggle = async () => {
    const toggled = !collapsed;
    setCollapsed(toggled);
    typeof window !== 'undefined' &&
      window.cookieStore.set('navCollapsed', toggled ? 'true' : 'false');
  };

  return collapsed ? (
    <RiContractRightLine
      className="bg-canvasBase text-subtle hidden h-5 w-5 cursor-pointer group-hover:block"
      onClick={toggle}
    />
  ) : (
    <RiContractLeftLine
      className="bg-canvasBase text-subtle hidden h-5 w-5 cursor-pointer group-hover:block"
      onClick={toggle}
    />
  );
};

export default function Logo({ collapsed, setCollapsed }: LogoProps) {
  return (
    <div
      className={`mt-5 flex h-10 w-full flex-row items-center ${
        collapsed ? 'justify-center' : 'ml-5 justify-start'
      }`}
    >
      <div className={`flex flex-row items-center justify-start ${collapsed ? '' : 'mr-4'} `}>
        {collapsed ? (
          <div className="cursor-pointer group-hover:hidden">
            <InngestLogoSmallBW />
          </div>
        ) : (
          <>
            <Link href={process.env.NEXT_PUBLIC_HOME_PATH as Route}>
              <InngestLogo className="text-basis mr-3" width={92} />
            </Link>
            <Search />
          </>
        )}
      </div>
      <NavToggle collapsed={collapsed} setCollapsed={setCollapsed} />
    </div>
  );
}
