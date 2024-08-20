'use client';

import type { Route } from 'next';
import Link from 'next/link';
import { NewButton } from '@inngest/components/Button';
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
      className={`${
        collapsed ? 'mx-auto' : 'mx-4'
      } mt-4 flex h-[28px] flex-row items-center justify-between`}
    >
      <div className={`flex flex-row items-center justify-start ${collapsed ? '' : 'mr-3'} `}>
        {collapsed ? (
          <div className="cursor-pointer group-hover:hidden">
            <InngestLogoSmallBW className="text-basis" />
          </div>
        ) : (
          <>
            <Link href={process.env.NEXT_PUBLIC_HOME_PATH as Route} scroll={false}>
              <InngestLogo className="text-basis mr-2" width={92} />
            </Link>
          </>
        )}
        <Search collapsed={collapsed} />
      </div>
      <NavToggle collapsed={collapsed} setCollapsed={setCollapsed} />
    </div>
  );
}
