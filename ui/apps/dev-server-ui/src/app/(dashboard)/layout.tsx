'use client';

import { TooltipProvider } from '@inngest/components/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import { Toaster } from 'sonner';
import colors from 'tailwindcss/colors';

import BG from '@/components/BG';
import Header from '@/components/Header';
import Navbar from '@/components/Navbar/Navbar';
import NavbarLink from '@/components/Navbar/NavbarLink';
import { IconFeed, IconFunction, IconWindow } from '@/icons';
import { useGetAppsQuery } from '@/store/generated';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { hasConnectedError } = useGetAppsQuery(undefined, {
    selectFromResult: (result) => ({
      hasConnectedError: result?.data?.apps?.some((app) => app.connected === false),
    }),
    pollingInterval: 1500,
  });

  return (
    <div
      className={cn(
        'relative grid h-screen w-screen overflow-hidden text-sm text-slate-400',
        'grid-cols-1 grid-rows-[50px_1fr]'
      )}
    >
      <BG />
      <Header>
        <Navbar>
          <NavbarLink icon={<IconFeed />} href="/stream" tabName="Stream" />
          <NavbarLink
            icon={<IconWindow />}
            href="/apps"
            hasError={hasConnectedError}
            tabName="Apps"
          />
          <NavbarLink icon={<IconFunction />} href="/functions" tabName="Functions" />
        </Navbar>
      </Header>
      <TooltipProvider>{children}</TooltipProvider>
      <Toaster
        theme="dark"
        toastOptions={{
          style: { background: colors.slate['700'] },
        }}
        position="top-right"
      />
    </div>
  );
}
