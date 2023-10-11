export const dynamic = 'force-dynamic';

type EventKeysLayoutProps = {
  children: React.ReactNode;
};
export default function EventKeysLayout({ children }: EventKeysLayoutProps) {
  return (
    <div className="flex min-h-0 flex-1">
      <div className="h-full min-w-0 flex-1 overflow-y-auto bg-white">{children}</div>
    </div>
  );
}
