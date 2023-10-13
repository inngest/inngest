import { interTight, robotoMono } from '@/app/fonts';
import '@/app/globals.css';

// This is separated from RootLayout so that we can use it in Storybook.
export function BaseWrapper({ children }: { children: React.ReactNode }) {
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
