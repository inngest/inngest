import type { Metadata } from 'next';

import StoreProvider from '@/app/StoreProvider';
import { interTight, robotoMono } from '@/app/fonts';
import './globals.css';

export const metadata: Metadata = {
  title: 'Inngest Development Server',
};

type RootLayoutProps = {
  children: React.ReactNode;
};

export default function RootLayout({ children }: RootLayoutProps) {
  return (
    <html lang="en" className={`${interTight.variable} ${robotoMono.variable}`}>
      <body className="bg-slate-940">
        <div id="app" />
        <div id="modals" />
        <StoreProvider>{children}</StoreProvider>
      </body>
    </html>
  );
}
