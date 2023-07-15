'use client';
import { usePathname } from 'next/navigation';
import BG from '@/components/BG';
import Header from '@/components/Header';
import Navbar from '@/components/Navbar/Navbar';
import NavbarLink from '@/components/Navbar/NavbarLink';
import classNames from '@/utils/classnames';
import { IconBook, IconFeed, IconFunction } from '@/icons';
import { useGetAppsQuery } from '@/store/generated';

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const { appsCount, hasConnectedError } = useGetAppsQuery(undefined, {
    selectFromResult: (result) => ({
      appsCount: result.data?.apps?.length || 0,
      hasConnectedError: result?.data?.apps?.some(
        (app) => app.connected === false
      ),
    }),
    pollingInterval: 1500,
  });

  return (
    <div
      className={classNames(
        "w-screen h-screen text-slate-400 text-sm grid overflow-hidden relative",
        pathname === "/stream"
          ? "grid-cols-app-sm xl:grid-cols-app 2xl:grid-cols-app-desktop grid-rows-app"
          : "grid-cols-docs grid-rows-docs"
      )}
    >
      <BG />
      <Header>
        <Navbar>
          <NavbarLink icon={<IconFeed />} href="stream" tabName="Stream" />
          <NavbarLink
            icon={<IconFunction />}
            href="apps"
            badge={appsCount}
            hasError={hasConnectedError}
            tabName="Apps"
          />
          <NavbarLink icon={<IconFunction />} href="functions" tabName="Functions" />
          <NavbarLink icon={<IconBook />} href="docs" tabName="Docs" />
        </Navbar>
      </Header>
      {children}
    </div>
  );
}
