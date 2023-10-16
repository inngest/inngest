import { interTight, robotoMono } from './fonts';
import './globals.css';

export function AppRoot({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${interTight.variable} ${robotoMono.variable}`}>
      <body className="bg-slate-940">
        <div id="app" />
        <div id="modals" />
        {children}
      </body>
    </html>
  );
}
