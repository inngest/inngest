import Head from 'next/head';

import { interTight, robotoMono } from './fonts';
import './globals.css';

export function AppRoot({
  children,
  mode,
  devServer = false,
}: {
  children: React.ReactNode;
  mode?: 'dark';
  devServer?: boolean;
}) {
  return (
    <html
      lang="en"
      className={`${devServer ? interTight.variable : ''} ${devServer ? robotoMono : ''} ${
        mode || ''
      } h-full`}
    >
      <body className="dark:bg-slate-940 h-full overflow-auto bg-white">
        <div id="app" />
        <div id="modals" />
        {children}
      </body>
    </html>
  );
}
