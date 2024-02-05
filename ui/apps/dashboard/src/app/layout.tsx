import { ClientFeatureFlagProvider } from '@/components/FeatureFlags/ClientFeatureFlagProvider';
import PageViewTracker from '@/components/PageViewTracker';
import URQLProvider from '@/queries/URQLProvider';
import SentryUserIdentification from './SentryUserIdentification';
import './globals.css';
import React from 'react';
import { type Metadata } from 'next';
import { ClerkProvider } from '@clerk/nextjs';
import colors from 'tailwindcss/colors';

import { BaseWrapper } from './baseWrapper';

export const metadata: Metadata = {
  title: 'Inngest Cloud',
  description: 'The Inngest Cloud dashboard',
  icons: process.env.NEXT_PUBLIC_FAVICON,
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <BaseWrapper>
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
            colorAlphaShade: colors.slate['900'],
            colorText: colors.slate['800'],
            colorTextSecondary: colors.slate['600'],
          },
          elements: {
            card: 'shadow-none border-0',
            logoBox: 'flex m-0 h-fit justify-center',
            headerTitle: 'text-lg',
            socialButtons: 'gap-4',
            socialButtonsBlockButton__github:
              'inline-flex flex-shrink-0 items-center gap-3 overflow-hidden rounded-[6px] transition-all shadow-outline-secondary-light font-medium px-6 py-2.5 bg-gray-900 text-white shadow-sm hover:bg-gray-700 hover:text-white',
            socialButtonsProviderIcon__github: 'invert',
            socialButtonsBlockButton__google:
              'inline-flex flex-shrink-0 items-center gap-3 overflow-hidden rounded-[6px] transition-all shadow-outline-secondary-light font-medium px-6 py-2.5 bg-slate-700 text-white shadow-sm hover:bg-slate-500 hover:text-white',
            socialButtonsBlockButtonText: 'text-sm font-regular',
            formFieldLabel: 'text-slate-600',
            formFieldInput:
              'border border-slate-300 placeholder-slate-500 shadow transition-all text-sm px-3.5 py-3 rounded-lg',
            formButtonPrimary:
              'inline-flex flex-shrink-0 items-center gap-1 justify-center overflow-hidden text-sm font-regular rounded-[6px] transition-all bg-gradient-to-b from-[#6d7bfe] to-[#6366f1] hover:from-[#7986fd] hover:to-[#7679f9] text-shadow text-white font-medium px-6 py-2.5 capitalize',
            footerActionText: 'text-sm font-medium text-slate-700',
            footerActionLink:
              'transition-color text-sm font-medium text-indigo-500 underline hover:text-indigo-800',
            footerPagesLink: 'text-sm font-medium text-slate-700',
          },
        }}
      >
        <SentryUserIdentification />
        <URQLProvider>
          <ClientFeatureFlagProvider>
            {children}
            <PageViewTracker />
          </ClientFeatureFlagProvider>
        </URQLProvider>
      </ClerkProvider>
    </BaseWrapper>
  );
}
