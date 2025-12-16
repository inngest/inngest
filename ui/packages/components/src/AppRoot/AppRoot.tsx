import './globals.css';
import './fonts.css';
import { ThemeProvider } from 'next-themes';

export function AppRoot({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="h-full" suppressHydrationWarning>
      <body className=" bg-canvasBase text-basis h-full overflow-auto overscroll-none">
        <div id="app" />
        <div id="modals" />
        <ThemeProvider attribute="class" defaultTheme="system">
          {children}
        </ThemeProvider>
      </body>
    </html>
  );
}
