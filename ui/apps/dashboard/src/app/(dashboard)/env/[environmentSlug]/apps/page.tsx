'use client';

import { Squares2X2Icon } from '@heroicons/react/20/solid';
import { Link } from '@inngest/components/Link';
import { useLocalStorage } from 'react-use';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { Banner } from '@/components/Banner';
import Header, { type HeaderLink } from '@/components/Header/Header';
import { useBooleanSearchParam } from '@/utils/useSearchParam';
import { Apps } from './Apps';

export default function Page() {
  const env = useEnvironment();
  const [isAppsBannerVisible, setIsAppsBannerVisible] = useLocalStorage(
    'AppsLaunchBannerVisible',
    true
  );

  const [isArchived] = useBooleanSearchParam('archived');

  const navLinks: HeaderLink[] = [
    {
      active: isArchived !== true,
      href: `/env/${env.slug}/apps`,
      text: 'Active',
    },
    {
      active: isArchived,
      href: `/env/${env.slug}/apps?archived=true`,
      text: 'Archived',
    },
  ];

  return (
    <>
      <Header
        icon={<Squares2X2Icon className="h-5 w-5 text-white" />}
        links={navLinks}
        title="Apps"
      />
      <div className="relative h-full overflow-y-auto bg-slate-100">
        {isAppsBannerVisible && (
          <Banner
            kind="info"
            onDismiss={() => {
              setIsAppsBannerVisible(false);
            }}
            className="absolute"
          >
            <p className="pr-2">
              Inngest deploys have been renamed to “<b>Syncs</b>”. All of your syncs can be found
              within your apps.{' '}
            </p>
            {/* To do: wire this to the docs */}
            {/* <Link internalNavigation={false} href="">
            Learn More
          </Link> */}
          </Banner>
        )}

        <Apps isArchived={isArchived} />
      </div>
    </>
  );
}
