import type { Metadata } from 'next';
import { AppRoot } from '@inngest/components/AppRoot';

import StoreProvider from '@/app/StoreProvider';

export const metadata: Metadata = {
  title: 'Inngest Development Server',
};

type RootLayoutProps = {
  children: React.ReactNode;
};

export default function RootLayout({ children }: RootLayoutProps) {
  return (
    <AppRoot>
      <StoreProvider>{children}</StoreProvider>
    </AppRoot>
  );
}
