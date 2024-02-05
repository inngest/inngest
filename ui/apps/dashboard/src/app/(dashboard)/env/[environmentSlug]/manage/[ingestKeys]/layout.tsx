import Keys from './Keys';

export const dynamic = 'force-dynamic';

type KeysLayoutProps = {
  children: React.ReactNode;
};
export default function KeysLayout({ children }: KeysLayoutProps) {
  return (
    <div className="flex min-h-0 flex-1">
      <div className="w-80 flex-shrink-0 border-r border-slate-300">
        <Keys />
      </div>
      <div className="h-full min-w-0 flex-1 overflow-y-auto bg-white">{children}</div>
    </div>
  );
}
