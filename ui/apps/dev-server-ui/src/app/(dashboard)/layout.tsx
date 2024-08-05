'use client';

import React from 'react';
import { TooltipProvider } from '@inngest/components/Tooltip';
import { Toaster } from 'sonner';
import colors from 'tailwindcss/colors';

import Layout from '@/components/Layout/Layout';
import { useGetAppsQuery } from '@/store/generated';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { hasSyncingError } = useGetAppsQuery(undefined, {
    selectFromResult: (result) => ({
      hasSyncingError: result?.data?.apps?.some((app) => app.connected === false),
    }),
    pollingInterval: 1500,
  });

  const collapsed = localStorage.getItem('collapsed');

  return (
    <TooltipProvider delayDuration={0}>
      <Layout>
        <React.Suspense>{children}</React.Suspense>

        <Toaster
          theme="dark"
          toastOptions={{
            style: { background: colors.slate['700'] },
          }}
        />
      </Layout>
    </TooltipProvider>
  );
}
