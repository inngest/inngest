import React from 'react';
import { type Metadata } from 'next';
import { ClerkProvider } from '@clerk/nextjs';
import { AppRoot } from '@inngest/components/AppRoot';
import { getButtonColors } from '@inngest/components/Button/buttonStyles';
import { TooltipProvider } from '@inngest/components/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';
import tailwindConfig from 'tailwind.config';
import resolveConfig from 'tailwindcss/resolveConfig';

import { ClientFeatureFlagProvider } from '@/components/FeatureFlags/ClientFeatureFlagProvider';
import PageViewTracker from '@/components/PageViewTracker';
import SentryUserIdentification from './SentryUserIdentification';

const {
  theme: { textColor, colors, backgroundColor, borderColor },
} = resolveConfig(tailwindConfig);

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
  const dark = isDark();

  const primarySolidButton =
    'data-[color=primary]:data-[variant=solid]:bg-btnPrimary data-[color=primary]:data-[variant=solid]:focus:bg-btnPrimaryPressed data-[color=primary]:data-[variant=solid]:hover:bg-btnPrimaryHover data-[color=primary]:data-[variant=solid]:active:bg-btnPrimaryPressed data-[color=primary]:data-[variant=solid]:disabled:bg-btnPrimaryDisabled data-[color=primary]:data-[variant=solid]:text-alwaysWhite';
  // primary outline is using secondary outlined
  const primaryGhostButton =
    'data-[color=primary]:data-[variant=ghost]:text-btnPrimary data-[color=primary]:data-[variant=ghost]:focus:bg-canvasSubtle data-[color=primary]:data-[variant=ghost]:hover:bg-canvasSubtle data-[color=primary]:data-[variant=ghost]:active:bg-canvasMuted data-[color=primary]:data-[variant=ghost]:disabled:bg-disabled data-[color=primary]:data-[variant=ghost]:disabled:text-btnPrimaryDisabled data-[color=primary]:data-[variant=ghost]:text-btnPrimary data-[color=primary]:data-[variant=ghost]:focus:bg-canvasSubtle data-[color=primary]:data-[variant=ghost]:hover:bg-canvasSubtle  data-[color=primary]:data-[variant=ghost]:active:bg-canvasMuted data-[color=primary]:data-[variant=ghost]:disabled:bg-disabled data-[color=primary]:data-[variant=ghost]:disabled:text-btnPrimaryDisabled';
  const primaryOutlineButton =
    'data-[color=primary]:data-[variant=outline]:border data-[color=primary]:data-[variant=outline]:border-muted data-[color=primary]:data-[variant=outline]:text-basis data-[color=primary]:data-[variant=outline]:bg-canvasBase data-[color=primary]:data-[variant=outline]:focus:bg-canvasSubtle data-[color=primary]:data-[variant=outline]:hover:bg-canvasSubtle data-[color=primary]:data-[variant=outline]:active:bg-canvasMuted data-[color=primary]:data-[variant=outline]:disabled:border-disabled data-[color=primary]:data-[variant=outline]:disabled:bg-disabled data-[color=primary]:data-[variant=outline]:disabled:text-disabled';
  const neutralGhostButton =
    'data-[color="neutral"]:data-[variant="ghost"]:disabled:text-disabled data-[color="neutral"]:data-[variant="ghost"]:disabled:bg-disabled data-[color="neutral"]:data-[variant="ghost"]:active:bg-canvasMuted data-[color="neutral"]:data-[variant="ghost"]:hover:bg-canvasSubtle data-[color="neutral"]:data-[variant="ghost"]:text-basis data-[color="neutral"]:data-[variant="ghost"]:focus:bg-canvasSubtle ';
  const dangerGhostButton =
    'data-[color=danger]:data-[variant=ghost]:text-btnDanger data-[color=danger]:data-[variant=ghost]:focus:bg-canvasSubtle data-[color=danger]:data-[variant=ghost]:hover:bg-canvasSubtle data-[color=danger]:data-[variant=ghost]:active:bg-canvasMuted data-[color=danger]:data-[variant=ghost]:disabled:bg-disabled data-[color=danger]:data-[variant=ghost]:disabled:text-btnDangerDisabled';
  return (
    <AppRoot>
      <ClerkProvider
        appearance={{
          layout: {
            logoImageUrl: '/images/logos/inngest.svg',
            logoPlacement: 'outside' as const,
          },
          variables: {
            colorPrimary: resolveColor(colors.primary.moderate, dark, '#2c9b63'), //matcha 500
            colorDanger: resolveColor(textColor.error, dark, '#cb2a1d'), //ruby-600
            colorSuccess: resolveColor(textColor.success, dark, '#027a48'), //matcha 600
            colorWarning: resolveColor(textColor.warning, dark, '#ba4210'), //honey 600
            colorText: resolveColor(textColor.basis, dark, '#242424'), //carbon 900
            colorTextSecondary: resolveColor(textColor.subtle, dark, '#4b4b4b'), // carbon 700
            colorBackground: resolveColor(backgroundColor.canvasBase, dark, '#fefefe'),
            colorInputBackground: resolveColor(backgroundColor.canvasBase, dark, '#fefefe'), // carbon 0
            colorInputText: resolveColor(textColor.basis, dark, '#242424'),
            colorNeutral: resolveColor(borderColor.subtle, dark, '#E2E2E2'), //carbon 100
            colorShimmer: resolveColor(colors.primary.moderate, dark, '#2c9b63'),
          },
          elements: {
            button: cn(
              '!shadow-none disabled:cursor-not-allowed font-normal',
              primarySolidButton,
              primaryGhostButton,
              primaryOutlineButton,
              neutralGhostButton,
              dangerGhostButton
            ),
            input:
              '!border !ring-0 focus:ring-0 bg-canvasBase border-muted hover:border-muted focus:border-muted placeholder-disabled text-basis focus:outline-primary-moderate w-full border text-sm leading-none outline-2 transition-all focus:outline rounded-lg',
            main: 'text-basis',
            rootBox: 'px-6 mx-auto max-w-[1200px]',
            card: 'shadow-none border-0 bg-canvasBase',
            actionCard: 'bg-canvasSubtle text-basis',
            cardBox: 'shadow-none h-fit block',
            scrollBox: 'w-fit md:min-w-[800px]',
            logoBox: 'flex m-0 h-fit justify-center',
            header: 'my-9 group-[.cl-tabPanel]:m-0 group-[.cl-formContainer]:m-0',
            headerTitle:
              'text-2xl font-normal group-[.cl-tabPanel]:text-sm group-[.cl-tabPanel]:font-medium group-[.cl-formContainer]:text-lg group-[.cl-formContainer]:font-medium',
            tabPanel: 'group',
            formContainer: 'group',
            tabListContainer: 'border-b-subtle group',
            tabButton:
              'hover:bg-canvasSubtle text-sm px-2 aria-selected:border-contrast aria-selected:border-b-2 aria-selected:!text-basis !text-muted',
            notificationBadge: 'bg-canvasMuted text-basis', // Pill component solid default styles
            badge:
              '!shadow-none border border-muted bg-canvasBase text-basis shadow-none data-[color=warning]:bg-warning data-[color=warning]:border-warning data-[color=warning]:text-warning data-[color=success]:bg-success data-[color=success]:border-success data-[color=success]:text-success data-[color=danger]:bg-error data-[color=danger]:border-error data-[color=danger]:text-error', // Pill component outlined default styles
            tagPillContainer: 'bg-canvasMuted text-basis shadow-none hover:bg-surfaceMuted', // Pill component solid default styles
            table: 'border border-subtle rounded-lg shadow-none bg-canvasBase',
            tableHead:
              'border-b border-subtle pl-4 pr-2 py-3 whitespace-nowrap text-muted text-sm font-semibold',
            formattedDate__tableCell: 'text-sm',
            formInputGroup: 'shadow-none',
            socialButtons: 'flex flex-col gap-4',
            profileSection: 'flex-col-reverse gap-2 border border-subtle rounded-md p-6 pt-0 mb-8',
            profileSectionTitleText: 'text-muted text-lg',
            profileSection__organizationProfile: 'border-0',
            profileSection__organizationDanger: 'border-0 !flex-row-reverse !justify-between',
            profileSection__profile: 'border-0',
            profileSection__danger:
              'border-0 !flex-row-reverse !justify-between items-baseline !my-0',
            profileSectionItem__danger: 'p-0',
            profileSectionTitle: 'pt-6',
            profileSectionHeader: 'text-sm text-muted font-medium mt-0 pt-0',
            profileSectionContent__organizationDanger: 'w-fit',
            profileSectionContent__danger: 'w-fit',
            profileSectionPrimaryButton__organizationProfile: getButtonColors({
              kind: 'primary',
              appearance: 'outlined',
              loading: false,
            }),
            profileSectionPrimaryButton__organizationDanger: getButtonColors({
              kind: 'danger',
              appearance: 'outlined',
              loading: false,
            }),
            profileSectionPrimaryButton__profile: getButtonColors({
              kind: 'primary',
              appearance: 'outlined',
              loading: false,
            }),
            profileSectionPrimaryButton__danger: getButtonColors({
              kind: 'danger',
              appearance: 'outlined',
              loading: false,
            }),
            socialButtonsBlockButton__github:
              'border border-muted text-basis focus:bg-canvasSubtle hover:bg-canvasSubtle active:bg-canvasMuted h-10 text-xs leading-[18px] px-3 py-1.5 flex items-center justify-center whitespace-nowrap rounded-md ',
            socialButtonsProviderIcon__github: 'dark:invert',
            socialButtonsBlockButton__google:
              'border border-muted text-basis focus:bg-canvasSubtle hover:bg-canvasSubtle active:bg-canvasMuted h-10 text-xs leading-[18px] px-3 py-1.5 flex items-center justify-center whitespace-nowrap rounded-md ',
            socialButtonsBlockButtonText: 'text-sm font-normal',
            form: 'text-left',
            formFieldLabel: 'text-basis text-sm font-medium',
            formFieldAction:
              'text-subtle hover:text-subtle hover:decoration-subtle decoration-transparent decoration-1 underline underline-offset-2 cursor-pointer transition-color duration-300',
            buttonArrowIcon: 'hidden',
            tagInputContainer: 'border-0 shadow-none bg-transparent *:px-3 *:p-1.5 *:text-sm',
            footerActionText: 'text-sm font-medium text-basis',
            footerActionLink:
              'text-link hover:text-link hover:decoration-link decoration-transparent decoration-1 underline underline-offset-2 cursor-pointer transition-color duration-300',
            footerPagesLink: 'text-sm font-medium text-basis',
            profileSection__activeDevices: 'max-h-80 overflow-scroll',
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
