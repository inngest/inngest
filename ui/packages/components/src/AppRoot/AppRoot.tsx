import { interTight, robotoMono } from './fonts';
import './globals.css';

export function AppRoot({ children, mode }: { children: React.ReactNode; mode?: 'dark' }) {
  return (
    <html lang="en" className={`${interTight.variable} ${robotoMono.variable} ${mode || ''}`}>
      <body className="dark:bg-slate-940 bg-white">
        <div id="app" />
        <div id="modals" />
        {children}
      </body>
    </html>
  );
}
