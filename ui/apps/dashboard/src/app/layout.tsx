import React from 'react';
import { type Metadata } from 'next';
import { AppRoot } from '@inngest/components/AppRoot';
import { TooltipProvider } from '@inngest/components/Tooltip';

import { ClientFeatureFlagProvider } from '@/components/FeatureFlags/ClientFeatureFlagProvider';
import PageViewTracker from '@/components/PageViewTracker';
import ClerkProvider from './Provider';
import SentryUserIdentification from './SentryUserIdentification';

export const metadata: Metadata = {
  title: 'Inngest Cloud',
  description: 'The Inngest Cloud dashboard',
  icons: {
    icon: [
      {
        url: process.env.NEXT_PUBLIC_FAVICON ?? '/favicon-june-2024-light.png',
        media: '(prefers-color-scheme: light)',
      },
      {
        url: process.env.NEXT_PUBLIC_FAVICON ?? '/favicon-june-2024-dark.png',
        media: '(prefers-color-scheme: dark)',
      },
    ],
  },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <AppRoot>
      <ClerkProvider>
        <SentryUserIdentification />
        <ClientFeatureFlagProvider>
          <TooltipProvider delayDuration={0}>{children}</TooltipProvider>
          <PageViewTracker />
        </ClientFeatureFlagProvider>
      </ClerkProvider>
    </AppRoot>
  );
}
