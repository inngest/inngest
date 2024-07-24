import './globals.css';
import './fonts.css';

export function AppRoot({ children, mode }: { children: React.ReactNode; mode?: 'dark' }) {
  return (
    <html lang="en" className={`${mode || ''} h-full`}>
      <body className="dark:bg-slate-940 h-full overflow-auto bg-white">
        <div id="app" />
        <div id="modals" />
        {children}
      </body>
    </html>
  );
}
