import Layout from '@/components/Layout/Layout';
import { TooltipProvider } from '@inngest/components/Tooltip';
import { createFileRoute, Outlet } from '@tanstack/react-router';
import * as React from 'react';

export const Route = createFileRoute('/_dashboard')({
  component: DashboardComponent,
});

function DashboardComponent() {
  return (
    <TooltipProvider delayDuration={0}>
      <Layout>
        <React.Suspense>
          <Outlet />
        </React.Suspense>
      </Layout>
    </TooltipProvider>
  );
}
