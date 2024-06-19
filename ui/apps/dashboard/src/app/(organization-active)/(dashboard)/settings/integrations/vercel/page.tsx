import type { Route } from 'next';
import { NewButton } from '@inngest/components/Button/index';
import { Link } from '@inngest/components/Link/Link';
import { IconVercel } from '@inngest/components/icons/platforms/Vercel';
import { RiExternalLinkLine } from '@remixicon/react';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import OldVercelIntegrationPage from './oldPage';
import VercelProjects from './projects';

export default async function VercelIntegrationPage() {
  const newIntegrations = await getBooleanFlag('new-integrations');

  return !newIntegrations ? (
    <OldVercelIntegrationPage />
  ) : (
    <div className="mx-auto mt-8 flex w-[800px] flex-col p-8">
      <div className="flex flex-row items-center justify-between">
        <div className="flex flex-row items-center justify-start">
          <div className="mb-7 mr-4 flex h-[52px] w-[52px] items-center justify-center rounded bg-black">
            <IconVercel className="h-6 w-6 text-white" />
          </div>
          <div className="flex flex-col">
            <div className="mb-2 text-2xl font-medium text-gray-900">Vercel</div>
            <div className="mb-7 text-slate-500">
              You can manage all your projects on this page.{' '}
              <Link showIcon={false} href={'https://www.inngest.com/docs/deploy/vercel' as Route}>
                Learn more
              </Link>
            </div>
          </div>
        </div>

        <div className="place-self-start">
          <NewButton
            appearance="outlined"
            kind="secondary"
            href={'https://vercel.com/integrations/inngest' as Route}
            icon={<RiExternalLinkLine className="mr-1" />}
            iconSide="left"
            label="Go to Vercel"
          />
        </div>
      </div>
      <VercelProjects />
    </div>
  );
}
