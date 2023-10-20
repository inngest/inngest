import URQLProvider from '@/queries/URQLProvider';

export default function LoggedOutLayout({ children }: { children: React.ReactNode }) {
  return <URQLProvider>{children}</URQLProvider>;
}
