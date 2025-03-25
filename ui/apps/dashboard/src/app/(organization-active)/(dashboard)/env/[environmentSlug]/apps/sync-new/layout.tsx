import { Header } from '@inngest/components/Header/Header';

type SyncNewLayoutProps = {
  params: {
    environmentSlug: string;
  };
  children: React.ReactNode;
};

export default async function Layout({
  children,
  params: { environmentSlug },
}: SyncNewLayoutProps) {
  return (
    <>
      <Header
        breadcrumb={[{ text: 'Apps', href: `/env/${environmentSlug}/apps` }, { text: 'Sync new' }]}
      />
      {children}
    </>
  );
}
