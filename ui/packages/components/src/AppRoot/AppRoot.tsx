import './globals.css';
import './fonts.css';
import { ThemeProvider } from 'next-themes';

export function AppRoot({ children, mode }: { children: React.ReactNode; mode?: 'dark' }) {
  return (
    <html lang="en" className="h-full" suppressHydrationWarning>
      <body className=" bg-canvasBase text-basis h-full overflow-auto">
        <div id="app" />
        <div id="modals" />
        {/* Once released to everybody, we can defaultTheme="system" */}
        <ThemeProvider attribute="class" defaultTheme={mode || 'light'}>
          {children}
        </ThemeProvider>
      </body>
    </html>
  );
}
