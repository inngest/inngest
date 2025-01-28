import Keys from './Keys';

export const dynamic = 'force-dynamic';

type KeysLayoutProps = {
  children: React.ReactNode;
};
export default function KeysLayout({ children }: KeysLayoutProps) {
  return (
    <div className="flex min-h-0 flex-1">
      <div className="border-muted w-80 flex-shrink-0 border-r">
        <Keys />
      </div>
      <div className="text-basis h-full min-w-0 flex-1 overflow-y-auto">{children}</div>
    </div>
  );
}
