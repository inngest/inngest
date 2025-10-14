import { Header } from '@inngest/components/Header/Header';

type SyncNewLayoutProps = {
  params: Promise<{
    environmentSlug: string;
  }>;
  children: React.ReactNode;
};

export default async function Layout(props: SyncNewLayoutProps) {
  const params = await props.params;

  const { environmentSlug } = params;

  const { children } = props;

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Apps', href: `/env/${environmentSlug}/apps` }, { text: 'Sync new' }]}
      />
      {children}
    </>
  );
}
