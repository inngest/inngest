'use client';

import { TooltipProvider } from '@inngest/components/Tooltip';
import { IconApp } from '@inngest/components/icons/App';
import { IconFunction } from '@inngest/components/icons/Function';
import { cn } from '@inngest/components/utils/classNames';
import { Toaster } from 'sonner';
import colors from 'tailwindcss/colors';

import BG from '@/components/BG';
import Header from '@/components/Header';
import Navbar from '@/components/Navbar/Navbar';
import NavbarLink from '@/components/Navbar/NavbarLink';
import { IconFeed } from '@/icons';
import { useGetAppsQuery } from '@/store/generated';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { hasSyncingError } = useGetAppsQuery(undefined, {
    selectFromResult: (result) => ({
      hasSyncingError: result?.data?.apps?.some((app) => app.connected === false),
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
          <NavbarLink icon={<IconApp />} href="/apps" hasError={hasSyncingError} tabName="Apps" />
          <NavbarLink icon={<IconFunction />} href="/functions" tabName="Functions" />
        </Navbar>
      </Header>
      <TooltipProvider>{children}</TooltipProvider>
      <Toaster
        theme="dark"
        toastOptions={{
          style: { background: colors.slate['700'] },
        }}
      />
    </div>
  );
}
