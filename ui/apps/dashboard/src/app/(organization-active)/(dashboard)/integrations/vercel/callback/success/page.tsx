import { NewButton } from '@inngest/components/Button/index';
import { Card } from '@inngest/components/Card/Card';
import { IconVercel } from '@inngest/components/icons/platforms/Vercel';
import { RiCheckLine, RiInformationLine } from '@remixicon/react';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import VercelIntegrationCallbackSuccessPage from './oldPage';

type SuccessProps = {
  searchParams: {
    onSuccessRedirectURL: string;
    source?: string;
  };
};

export default async function SuccessPage({ searchParams }: SuccessProps) {
  const newIntegrations = await getBooleanFlag('new-integrations');

  return !newIntegrations ? (
    <VercelIntegrationCallbackSuccessPage searchParams={searchParams} />
  ) : (
    <div className="mx-auto mt-8 flex w-[800px] flex-col p-8">
      <div className="bg-contrast mb-7 flex h-12 w-12 items-center justify-center rounded">
        <IconVercel className="text-onContrast h-6 w-6" />
      </div>
      <div className="text-basis mb-2 text-2xl leading-loose">
        Inngest sucessfully installed on Vercel!
      </div>
      <div className="text-subtle mb-7 text-base">
        The Inngest integration has successfully been installed on your Vercel account.
      </div>
      <>
        <Card className="w-full">
          <Card.Content className="rounded-0 p-0">
            <div className="border-subtle flex h-[72px] flex-row items-start justify-start border-b p-4">
              <div className="bg-primary-moderate mr-3 mt-1 flex h-4 w-4 shrink-0 items-center justify-center rounded-[50%] ">
                <RiCheckLine size={12} className="text-onContrast" />
              </div>
              <div className="text-subtle text-base">
                Each Vercel project will have{' '}
                <span className="font-semibold">INNGEST_SIGNING_KEY</span> and{' '}
                <span className="font-semibold">INNGEST_EVENT_KEY</span> environment variables set.
              </div>
            </div>
            <div className="flex h-[72px] flex-row items-start justify-start p-4">
              <div className="bg-primary-moderate mr-3 mt-1 flex h-4 w-4 shrink-0 items-center justify-center rounded-[50%]">
                <RiCheckLine size={12} className="text-white" />
              </div>
              <div className="text-subtle text-base">
                The next time you deploy your project to Vercel your functions will automatically
                appear in the Inngest dashboard.
              </div>
            </div>
          </Card.Content>
        </Card>
        <div className="flex flex-row items-center justify-start rounded py-6">
          <RiInformationLine size={20} className="text-disabled mr-1" />
          <div className="text-subtle text-sm font-normal leading-tight">
            Advanced configuration options are available on the Inngest dashboard.
          </div>
        </div>
        <div>
          <NewButton
            kind="primary"
            appearance="solid"
            size="medium"
            label="Continue to Inngest Vercel Dashbaord"
            href={
              searchParams.source === 'marketplace'
                ? searchParams.onSuccessRedirectURL
                : '/settings/integrations/vercel'
            }
          />
        </div>
      </>
    </div>
  );
}
