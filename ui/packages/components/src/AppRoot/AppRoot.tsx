import './globals.css';
import './fonts.css';

export function AppRoot({ children, mode }: { children: React.ReactNode; mode?: 'dark' }) {
  return (
    <html lang="en" className={`${mode || ''} h-full`}>
      <body className=" bg-canvasBase text-basis h-full overflow-auto">
        <div id="app" />
        <div id="modals" />
        {children}
      </body>
    </html>
  );
}
