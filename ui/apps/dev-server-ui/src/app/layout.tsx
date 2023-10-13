import type { Metadata } from 'next';

import StoreProvider from '@/app/StoreProvider';
import { BaseWrapper } from './baseWrapper';

export const metadata: Metadata = {
  title: 'Inngest Development Server',
};

type RootLayoutProps = {
  children: React.ReactNode;
};

export default function RootLayout({ children }: RootLayoutProps) {
  return (
    <BaseWrapper>
      <StoreProvider>{children}</StoreProvider>
    </BaseWrapper>
  );
}
