'use client';

import { Squares2X2Icon } from '@heroicons/react/20/solid';

import Header, { type HeaderLink } from '@/components/Header/Header';
import { useEnvironment } from '@/queries';
import { useAppName } from './useAppName';

type Props = React.PropsWithChildren<{
  params: {
    environmentSlug: string;
    externalID: string;
  };
}>;

export default function Layout({ children, params: { environmentSlug, externalID } }: Props) {
  const [envRes] = useEnvironment({ environmentSlug });
  if (envRes.error) {
    throw envRes.error;
  }

  const res = useAppName({
    envID: envRes.data?.id ?? '',
    externalAppID: externalID,
    skip: !envRes.data,
  });
  if (res.error) {
    throw res.error;
  }

  if (envRes.fetching || res.isLoading || res.isSkipped) {
    return null;
  }

  const navLinks: HeaderLink[] = [
    {
      active: 'exact',
      href: `/env/${environmentSlug}/apps/${encodeURIComponent(externalID)}`,
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
