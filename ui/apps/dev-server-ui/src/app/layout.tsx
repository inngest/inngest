import type { Metadata } from 'next';
import { AppRoot } from '@inngest/components/AppRoot';
import { SignalsProvider, type SignalHandlers } from '@inngest/components/Signals/SignalsContext';

import StoreProvider from '@/app/StoreProvider';
import { useInvokeFunctionMutation } from '@/store/generated';

export const metadata: Metadata = {
  title: 'Inngest Development Server',
  icons: {
    icon: [
      {
        url: '/favicon-june-2024-light.png',
        media: '(prefers-color-scheme: light)',
      },
      {
        url: '/favicon-june-2024-dark.png',
        media: '(prefers-color-scheme: dark)',
      },
    ],
  },
};

type RootLayoutProps = {
  children: React.ReactNode;
};

export default function RootLayout({ children }: RootLayoutProps) {
  return (
    <AppRoot mode="dark">
      <StoreProvider>{children}</StoreProvider>
    </AppRoot>
  );
}
