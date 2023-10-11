export const dynamic = 'force-dynamic';

type Props = {
  children: React.ReactNode;
};
export default function Layout({ children }: Props) {
  return (
    <div className="flex min-h-0 flex-1">
      <div className="mt-4 h-full min-w-0 flex-1 overflow-y-auto bg-white">{children}</div>
    </div>
  );
}
