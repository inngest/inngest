'use client';

import { Squares2X2Icon } from '@heroicons/react/20/solid';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import Header, { type HeaderLink } from '@/components/Header/Header';
import { Time } from '@/components/Time';
import { ResyncButton } from './ResyncButton';
import { useNavData } from './useNavData';

type Props = React.PropsWithChildren<{
  params: {
    externalID: string;
  };
}>;

export default function Layout({ children, params: { externalID } }: Props) {
  externalID = decodeURIComponent(externalID);
  const env = useEnvironment();

  const res = useNavData({
    envID: env.id,
    externalAppID: externalID,
  });
  if (res.error) {
    if (res.error.message.includes('no rows')) {
      // TODO: Make this look better.
      return <span className="m-auto">App not found: {externalID}</span>;
    }
    throw res.error;
  }
  if (res.isLoading && !res.data) {
    return null;
  }

  const navLinks: HeaderLink[] = [
    {
      active: 'exact',
      href: `/env/${env.slug}/apps/${encodeURIComponent(externalID)}`,
      text: 'Info',
    },
    {
      active: 'exact',
      href: `/env/${env.slug}/apps/${encodeURIComponent(externalID)}/syncs`,
      text: 'Syncs',
    },
  ];

  let action;
  if (res.data.latestSync?.url) {
    const canResync = res.data.latestSync.platform !== 'vercel';
    console.log(res.data.latestSync.platform);
    if (canResync) {
      action = <ResyncButton latestSyncUrl={res.data.latestSync.url} />;
    }
  }

  return (
    <>
      <Header
        action={action}
        icon={<Squares2X2Icon className="h-5 w-5 text-white" />}
        links={navLinks}
        title={res.data.name}
      />
      <div className="h-full overflow-hidden bg-slate-100">{children}</div>
    </>
  );
}
