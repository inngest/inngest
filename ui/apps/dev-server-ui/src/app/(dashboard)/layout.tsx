'use client';

import React from 'react';
import { TooltipProvider } from '@inngest/components/Tooltip';
import { Toaster } from 'sonner';

import Layout from '@/components/Layout/Layout';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <TooltipProvider delayDuration={0}>
      <Layout>
        <React.Suspense>{children}</React.Suspense>

        <Toaster
          toastOptions={{
            className: 'drop-shadow-lg',
            style: {
              background: `rgb(var(--color-background-canvas-base))`,
              borderRadius: 0,
              borderWidth: '0px 0px 2px',
              color: `rgb(var(--color-foreground-base))`,
            },
          }}
        />
      </Layout>
    </TooltipProvider>
  );
}
