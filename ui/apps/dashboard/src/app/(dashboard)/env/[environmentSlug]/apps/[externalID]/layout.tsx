'use client';

import { Squares2X2Icon } from '@heroicons/react/20/solid';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import Header, { type HeaderLink } from '@/components/Header/Header';
import { useAppName } from './useAppName';

type Props = React.PropsWithChildren<{
  params: {
    externalID: string;
  };
}>;

export default function Layout({ children, params: { externalID } }: Props) {
  externalID = decodeURIComponent(externalID);
  const env = useEnvironment();

  const res = useAppName({
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
  if (res.isLoading) {
    return null;
  }

  const navLinks: HeaderLink[] = [
    {
      active: 'exact',
      href: `/env/${env.slug}/apps/${encodeURIComponent(externalID)}`,
      text: 'Info',
    },
    // TODO: Uncomment when the syncs page is added
    // {
    //   active: 'exact',
    //   href: `/env/${environmentSlug}/apps/${encodeURIComponent(externalID)}/syncs`,
    //   text: 'Syncs',
    // },
  ];

  return (
    <>
      <Header
        icon={<Squares2X2Icon className="h-5 w-5 text-white" />}
        links={navLinks}
        title={res.data}
      />
      <div className="overflow-y-auto">{children}</div>
    </>
  );
}
