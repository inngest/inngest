import Layout from '@/components/Layout/Layout'
import { TooltipProvider } from '@inngest/components/Tooltip'
import { createFileRoute, Outlet } from '@tanstack/react-router'
import * as React from 'react'
import { Toaster } from 'sonner'

export const Route = createFileRoute('/_dashboard')({
  component: DashboardComponent,
})

function DashboardComponent() {
  return (
    <TooltipProvider delayDuration={0}>
      <Layout>
        <React.Suspense>
          <Outlet />
        </React.Suspense>

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
  )
}
