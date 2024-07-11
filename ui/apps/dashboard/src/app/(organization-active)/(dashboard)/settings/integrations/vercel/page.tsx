import type { Route } from 'next';
import NextLink from 'next/link';
import { NewButton } from '@inngest/components/Button/index';
import { Link } from '@inngest/components/Link/Link';
import { IconVercel } from '@inngest/components/icons/platforms/Vercel';
import { RiArrowRightSLine, RiLinksLine } from '@remixicon/react';

import { vercelIntegration } from '../page';
import VercelProjects from './projects';

export default async function VercelIntegrationPage() {
  const integrations = await vercelIntegration();

  return (
    <div className="mx-auto mt-6 flex w-[800px] flex-col p-8">
      <div className="flex flex-row items-center justify-start">
        <NextLink href="/settings/integrations">
          <div className="text-subtle text-base">All integrations</div>
        </NextLink>
        <RiArrowRightSLine className="text-disabled h-4" />
        <div className="text-basis text-base">Vercel</div>
      </div>
      <div className="mt-6 flex flex-row items-center justify-between">
        <div className="flex flex-row items-center justify-start">
          <div className="bg-contrast mb-7 mr-4 flex h-[52px] w-[52px] items-center justify-center rounded">
            <IconVercel className="text-onContrast h-6 w-6" />
          </div>
          <div className="flex flex-col">
            <div className="text-basis mb-2 text-xl font-medium leading-7">Vercel</div>
            <div className="text-subtle mb-7 text-base">
              You can manage all your projects on this page.{' '}
              <Link showIcon={false} href={'https://www.inngest.com/docs/deploy/vercel' as Route}>
                Learn more
              </Link>
            </div>
          </div>
        </div>

        <div className="place-self-start">
          <NewButton
            kind="primary"
            href={'https://vercel.com/integrations/inngest/new' as Route}
            icon={<RiLinksLine className="mr-1" />}
            iconSide="left"
            label="Reconnect"
          />
        </div>
      </div>
      <VercelProjects integration={integrations} />
    </div>
  );
}
