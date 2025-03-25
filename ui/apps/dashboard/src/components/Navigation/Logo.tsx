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
  envName: string;
  setCollapsed: (arg: boolean) => void;
};

const NavToggle = ({
  collapsed,
  setCollapsed,
}: {
  collapsed: boolean;
  setCollapsed: (arg: boolean) => void;
}) => {
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
      size="small"
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

export default function Logo({
  collapsed,
  enableQuickSearchV2,
  envSlug,
  envName,
  setCollapsed,
}: LogoProps) {
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
              <InngestLogo className="text-basis mr-2 mt-1" width={82} />
            </NextLink>
          </>
        )}
        {enableQuickSearchV2 && (
          <QuickSearch collapsed={collapsed} envSlug={envSlug} envName={envName} />
        )}
        {!enableQuickSearchV2 && <Search collapsed={collapsed} />}
      </div>
      <NavToggle collapsed={collapsed} setCollapsed={setCollapsed} />
    </div>
  );
}
