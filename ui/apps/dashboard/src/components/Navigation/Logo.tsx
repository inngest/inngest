'use client';

import type { Route } from 'next';
import NextLink from 'next/link';
import { Button } from '@inngest/components/Button';
import { InngestLogo } from '@inngest/components/icons/logos/InngestLogo';
import { InngestLogoSmallBW } from '@inngest/components/icons/logos/InngestLogoSmall';
import { RiContractLeftLine, RiContractRightLine } from '@remixicon/react';

import { QuickSearch } from './QuickSearch/QuickSearch';
import Search from './Search';

type LogoProps = {
  collapsed: boolean;
  enableQuickSearchV2: boolean;
  envSlug: string;
  setCollapsed: (arg: boolean) => void;
};

const NavToggle = ({ collapsed, setCollapsed }: LogoProps) => {
  const toggle = async () => {
    const toggled = !collapsed;
    setCollapsed(toggled);

    if (typeof window !== 'undefined') {
      window.cookieStore.set('navCollapsed', toggled ? 'true' : 'false');
      //
      // some downstream things, like charts, may need to redraw themselves
      setTimeout(() => window.dispatchEvent(new Event('navToggle')), 200);
    }
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

export default function Logo({ collapsed, enableQuickSearchV2, envSlug, setCollapsed }: LogoProps) {
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
            <NextLink href={process.env.NEXT_PUBLIC_HOME_PATH as Route} scroll={false}>
              <InngestLogo className="text-basis mr-2" width={92} />
            </NextLink>
          </>
        )}
        {enableQuickSearchV2 && <QuickSearch collapsed={collapsed} envSlug={envSlug} />}
        {!enableQuickSearchV2 && <Search collapsed={collapsed} />}
      </div>
      <NavToggle
        collapsed={collapsed}
        enableQuickSearchV2={enableQuickSearchV2}
        envSlug={envSlug}
        setCollapsed={setCollapsed}
      />
    </div>
  );
}
