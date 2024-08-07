import { Header } from '@inngest/components/Header/Header';
import { IconApp } from '@inngest/components/icons/App';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import OldHeader from '@/components/Header/old/Header';

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
  const newIANav = await getBooleanFlag('new-ia-nav');

  return (
    <>
      {newIANav ? (
        <Header
          breadcrumb={[
            { text: 'Apps', href: `/env/${environmentSlug}/apps` },
            { text: 'Sync new' },
          ]}
        />
      ) : (
        <OldHeader title="Apps" icon={<IconApp className="h-5 w-5 text-white" />} />
      )}
      {children}
    </>
  );
}
