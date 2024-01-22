'use client';

import NextLink from 'next/link';
import { InformationCircleIcon, Squares2X2Icon } from '@heroicons/react/20/solid';
import { HoverCardContent, HoverCardRoot, HoverCardTrigger } from '@inngest/components/HoverCard';
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
        action={
          <HoverCardRoot>
            <HoverCardTrigger asChild>
              <NextLink
                className="flex cursor-pointer items-center gap-1 rounded-md border border-slate-200 px-2 py-1 text-slate-200 hover:border-white hover:text-white"
                href="https://www.inngest.com/docs/apps"
                target="_blank"
                rel="noreferrer noopener"
              >
                <InformationCircleIcon className="h-5 w-5" />
                What are Apps?
              </NextLink>
            </HoverCardTrigger>
            <HoverCardContent className="w-72 p-2.5 text-sm">
              <p>
                When you serve your functions using Inngest&apos;s serve API handler, you are
                hosting a new Inngest app.
              </p>
              <br />
              <p>
                Each time you deploy new code to your hosted platform, you must sync your app to
                Inngest. When using our Vercel or Netlify integrations, your app will be synced
                automatically.
              </p>
              <br />
              <p>Deploys have been renamed to “Syncs.” Syncs are found within Apps.</p>
              <br />
              <Link href="https://www.inngest.com/docs/apps/cloud">Read Docs</Link>
            </HoverCardContent>
          </HoverCardRoot>
        }
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
            <Link internalNavigation={false} href="https://www.inngest.com/docs/apps">
              Learn More
            </Link>
          </Banner>
        )}

        <Apps isArchived={isArchived} />
      </div>
    </>
  );
}
