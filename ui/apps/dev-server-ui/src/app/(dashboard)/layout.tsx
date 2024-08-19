'use client';

import React from 'react';
import { TooltipProvider } from '@inngest/components/Tooltip';
import { Toaster } from 'sonner';
import colors from 'tailwindcss/colors';

import Layout from '@/components/Layout/Layout';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
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
