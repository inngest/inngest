import React from 'react';
import { type Metadata } from 'next';
import { ClerkProvider } from '@clerk/nextjs';
import { AppRoot } from '@inngest/components/AppRoot';
import { TooltipProvider } from '@inngest/components/Tooltip';
import colors from 'tailwindcss/colors';

import { ClientFeatureFlagProvider } from '@/components/FeatureFlags/ClientFeatureFlagProvider';
import PageViewTracker from '@/components/PageViewTracker';
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
      <ClerkProvider
        appearance={{
          layout: {
            logoImageUrl: '/images/logos/inngest.svg',
            logoPlacement: 'outside',
          },
          variables: {
            colorPrimary: colors.indigo['500'],
            colorDanger: colors.red['500'],
            colorSuccess: colors.teal['500'],
            colorWarning: colors.amber['500'],
            colorText: colors.slate['800'],
            colorTextSecondary: colors.slate['600'],
          },
          elements: {
            card: 'shadow-none border-0',
            cardBox: 'shadow-none h-fit',
            scrollBox: 'w-fit md:min-w-[800px]',
            logoBox: 'flex m-0 h-fit justify-center',
            headerTitle: 'text-lg',
            socialButtons: 'flex flex-col gap-4',
            socialButtonsBlockButton__github:
              'border border-muted text-basis focus:bg-canvasSubtle hover:bg-canvasSubtle active:bg-canvasMuted h-10 text-xs leading-[18px] px-3 py-1.5 flex items-center justify-center whitespace-nowrap rounded-md ',
            socialButtonsBlockButton__google:
              'border border-muted text-basis focus:bg-canvasSubtle hover:bg-canvasSubtle active:bg-canvasMuted h-10 text-xs leading-[18px] px-3 py-1.5 flex items-center justify-center whitespace-nowrap rounded-md ',
            socialButtonsBlockButtonText: 'text-sm font-regular',
            form: 'text-left',
            formFieldLabel: 'text-basis text-sm font-medium',
            formFieldInput:
              'border border-muted placeholder-slate-500 shadow transition-all text-sm px-3.5 py-3 rounded-lg',
            formButtonPrimary:
              'bg-btnPrimary focus:bg-btnPrimaryPressed hover:bg-btnPrimaryHover active:bg-btnPrimaryPressed disabled:bg-btnPrimaryDisabled text-alwaysWhite h-8 text-xs leading-[18px] px-3 py-1.5 flex items-center justify-center whitespace-nowrap rounded-md disabled:cursor-not-allowed"',
            buttonArrowIcon: 'hidden',
            tagInputContainer:
              'border border-muted placeholder-slate-500 shadow transition-all rounded-lg focus-within:*:ring-0 *:px-3 *:p-1.5 *:text-sm',
            footerActionText: 'text-sm font-medium text-basis',
            footerActionLink:
              'text-link hover:decoration-link decoration-transparent decoration-1 underline underline-offset-2 cursor-pointer transition-color duration-300',
            footerPagesLink: 'text-sm font-medium text-basis',
          },
        }}
      >
        <SentryUserIdentification />
        <ClientFeatureFlagProvider>
          <TooltipProvider delayDuration={0}>{children}</TooltipProvider>
          <PageViewTracker />
        </ClientFeatureFlagProvider>
      </ClerkProvider>
    </AppRoot>
  );
}
