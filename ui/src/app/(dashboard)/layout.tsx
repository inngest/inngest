"use client";

import Header from '@/components/Header';
import Navbar from '@/components/Navbar/Navbar';
import NavbarLink from '@/components/Navbar/NavbarLink';
import { IconBook, IconFeed, IconFunction } from '@/icons';
import { useGetAppsQuery } from '@/store/generated';

export default function DashboardLayout({children}: {children: React.ReactNode}) {
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
    <>
      <Header>
        <Navbar>
          <NavbarLink
            icon={<IconFeed />}
            href="stream"
            tabName="Stream"
          />
          <NavbarLink
            icon={<IconFunction />}
            href="functions"
            badge={appsCount}
            hasError={hasConnectedError}
            tabName="Functions"
          />
          <NavbarLink
            icon={<IconFunction />}
            href="apps"
            tabName="Apps"
          />
          <NavbarLink
            icon={<IconBook />}
            href="docs"
            tabName="Docs"
          />
        </Navbar>
      </Header>
      {children}
    </>
  );
}
