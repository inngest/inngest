import './globals.css';
import type { Metadata } from 'next';

import StoreProvider from '@/app/StoreProvider';
import BG from '@/components/BG';
import Header from '@/components/Header';
import Sidebar from '@/components/Sidebar/Sidebar';
import SidebarLink from '@/components/Sidebar/SidebarLink';
import { IconBook, IconFeed, IconFunction } from '@/icons';

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
          <div className="relative w-screen h-screen text-slate-400 text-sm overflow-hidden">
            <BG />
            <Header />
            <div className="flex w-full h-full">
              <Sidebar>
                <SidebarLink href="/feed/events" icon={<IconFeed />} badge={20} />
                <SidebarLink href="/functions" icon={<IconFunction />} />
                <SidebarLink href="/docs" icon={<IconBook />} />
              </Sidebar>
              <div className="flex-1">{children}</div>
            </div>
          </div>
        </StoreProvider>
      </body>
    </html>
  );
}
