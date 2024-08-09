'use client';

import NextLink from 'next/link';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { HoverCardContent, HoverCardRoot, HoverCardTrigger } from '@inngest/components/HoverCard';
import { Link } from '@inngest/components/Link';
import { useBooleanSearchParam } from '@inngest/components/hooks/useSearchParam';
import { IconApp } from '@inngest/components/icons/App';
import { RiAddLine, RiInformationLine } from '@remixicon/react';

import { useEnvironment } from '@/components/Environments/environment-context';
import Header, { type HeaderLink } from '@/components/Header/old/Header';
import { pathCreator } from '@/utils/urls';
import { Apps } from './Apps';

export default function Page() {
  const env = useEnvironment();
  const router = useRouter();

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
        icon={<IconApp className="h-5 w-5 text-white" />}
        links={navLinks}
        title="Apps"
        action={
          <div className="flex items-center gap-2">
            {!isArchived && (
              <Button
                kind="primary"
                label="Sync App"
                btnAction={() => router.push(pathCreator.createApp({ envSlug: env.slug }))}
                icon={<RiAddLine />}
              />
            )}
            <HoverCardRoot>
              <HoverCardTrigger asChild>
                <NextLink
                  className="flex cursor-pointer items-center gap-1 rounded-md border border-slate-200 px-2 py-1 text-slate-200 hover:border-white hover:text-white"
                  href="https://www.inngest.com/docs/apps"
                  target="_blank"
                  rel="noreferrer noopener"
                >
                  <RiInformationLine className="h-5 w-5" />
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
                <Link href="https://www.inngest.com/docs/apps/cloud">Read Docs</Link>
              </HoverCardContent>
            </HoverCardRoot>
          </div>
        }
      />
      <div className="bg-canvasBase relative my-16 h-full overflow-y-auto px-6">
        <Apps isArchived={isArchived} />
      </div>
    </>
  );
}
