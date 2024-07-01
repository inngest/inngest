'use client';

import type { ReactNode } from 'react';
import { NewButton } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card';
import { IconDatadog } from '@inngest/components/icons/platforms/Datadog';
import { IconNetlify } from '@inngest/components/icons/platforms/Netlify';
import { IconVercel } from '@inngest/components/icons/platforms/Vercel';
import { RiExternalLinkLine } from '@remixicon/react';

import { useVercelIntegration } from './vercel/useVercelIntegration';

type Integration = {
  title: string;
  Icon: ReactNode;
  actionButton: (enabled: boolean, loading?: boolean) => ReactNode;
  description: string;
  upvoteId?: string;
};

const INTEGRATIONS: Integration[] = [
  {
    title: 'Vercel',
    Icon: <IconVercel className="text-onContrast h-6 w-6" />,
    actionButton: (enabled, loading) => (
      <NewButton
        kind="primary"
        appearance="solid"
        size="medium"
        loading={loading}
        href={enabled ? '/settings/integrations/vercel' : '/settings/integrations/vercel/connect'}
        label={enabled ? 'Manage' : 'Connect'}
      />
    ),
    description:
      'Host your Inngest functions on Vercel and automatically sync them every time you deploy code.',
  },
  {
    title: 'Netlify',
    Icon: <IconNetlify className="text-onContrast h-6 w-6" />,
    actionButton: () => (
      <NewButton
        icon={<RiExternalLinkLine />}
        iconSide="left"
        kind="secondary"
        appearance="outlined"
        size="medium"
        label="View docs"
        href="https://www.inngest.com/docs/deploy/netlify"
      />
    ),
    description:
      'Check out our docs to see how you can use Inngest with your applications deployed to Netlify.',
  },
  {
    title: 'Datadog',
    Icon: <IconDatadog className="text-onContrast h-6 w-6" />,
    actionButton: () => (
      <NewButton
        icon={<RiExternalLinkLine />}
        iconSide="left"
        kind="secondary"
        appearance="outlined"
        size="medium"
        label="Upvote"
        href="https://roadmap.inngest.com/roadmap?id=b9303313-0234-4ece-8376-df862364c16b"
        target="_blank"
      />
    ),
    description: 'Let us know if a Datadog integration is important to you by upvoting!',
  },
];

export default function IntegrationsList() {
  const { data: vercelData, fetching } = useVercelIntegration();
  return (
    <div className="mx-auto mt-16 flex w-[800px] flex-col">
      <div className="mb-7 w-full text-2xl font-medium">All integrations</div>
      <div className="grid w-[800px] grid-cols-2 gap-6">
        {INTEGRATIONS.map((i: Integration, n) => (
          <Card key={`integration-card-${n}`}>
            <div className="flex h-[189px] w-[388px] flex-col p-6">
              <div className="align-center flex flex-row items-center justify-between">
                <div className="bg-contrast flex h-12 w-12 items-center justify-center rounded">
                  {i.Icon}
                </div>
                {i.actionButton(
                  i.title === 'Vercel'
                    ? vercelData.enabled && vercelData.projects.length > 0
                    : false,
                  fetching
                )}
              </div>
              <div className="text-basis mt-[18px] text-lg font-medium">{i.title}</div>
              <div className="text-subtle mt-2 text-sm leading-tight">{i.description}</div>
            </div>
          </Card>
        ))}
        <Card>
          <div className="bg-canvasSubtle flex h-[189px] w-[388px] flex-col p-6">
            <div className="text-basis text-lg font-medium">Can&apos;t find what you need?</div>
            <div className="text-basis mt-3 text-sm leading-tight">
              Write to our team about the integration you are looking for and we will get back to
              you.
            </div>
            <div>
              <NewButton
                icon={<RiExternalLinkLine />}
                iconSide="left"
                kind="secondary"
                appearance="outlined"
                size="medium"
                label="Request integration"
                className="border-muted bg-subtle mt-5"
                href="https://roadmap.inngest.com/roadmap"
                target="_blank"
              />
            </div>
          </div>
        </Card>
      </div>
    </div>
  );
}
