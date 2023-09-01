'use client';

import { Toaster } from 'sonner';

import BG from '@/components/BG';
import Header from '@/components/Header';
import Navbar from '@/components/Navbar/Navbar';
import NavbarLink from '@/components/Navbar/NavbarLink';
import { IconFeed, IconFunction, IconWindow } from '@/icons';
import { useGetAppsQuery } from '@/store/generated';
import classNames from '@/utils/classnames';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { appsCount, hasConnectedError } = useGetAppsQuery(undefined, {
    selectFromResult: (result) => ({
      appsCount: result.data?.apps?.length || 0,
      hasConnectedError: result?.data?.apps?.some((app) => app.connected === false),
    }),
    pollingInterval: 1500,
  });

  return (
    <div
      className={classNames(
        'w-screen h-screen text-slate-400 text-sm grid overflow-hidden relative',
        'grid-cols-app grid-rows-app',
      )}
    >
      <BG />
      <Header>
        <Navbar>
          <NavbarLink icon={<IconFeed />} href="stream" tabName="Stream" />
          <NavbarLink
            icon={<IconWindow />}
            href="apps"
            badge={appsCount}
            hasError={hasConnectedError}
            tabName="Apps"
          />
          <NavbarLink icon={<IconFunction />} href="functions" tabName="Functions" />
        </Navbar>
      </Header>
      {children}
      <Toaster theme="dark" toastOptions={{ style: { background: '#334155' } }} />
    </div>
  );
}
