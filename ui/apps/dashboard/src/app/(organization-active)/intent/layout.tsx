import InngestLogo from '@/icons/InngestLogo';

export default function SettingsLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="h-full overflow-y-scroll">
      <div className="mx-auto flex h-full max-w-screen-xl flex-col px-6">
        <header className="flex items-center justify-between py-6">
          <InngestLogo />
          <h1 className="hidden">Inngest</h1>
        </header>
        <div className="flex grow items-center">{children}</div>
      </div>
    </div>
  );
}
