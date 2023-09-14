import type { Metadata } from 'next';

import StoreProvider from '@/app/StoreProvider';

import './globals.css';

export const metadata: Metadata = {
  title: 'Inngest Development Server',
};

type RootLayoutProps = {
  children: React.ReactNode;
};

export default function RootLayout({ children }: RootLayoutProps) {
  return (
    <html lang="en">
      <body className="bg-slate-1000">
        <div id="app" />
        <div id="modals" />
        <StoreProvider>
          {children}
        </StoreProvider>
      </body>
    </html>
  );
}
