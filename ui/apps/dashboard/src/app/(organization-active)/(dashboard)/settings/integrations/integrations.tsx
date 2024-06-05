'use client';

import type { ReactNode } from 'react';
import { Button } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card';
import { IconDatadog } from '@inngest/components/icons/platforms/Datadog';
import { IconNetlify } from '@inngest/components/icons/platforms/Netlify';
import { IconVercel } from '@inngest/components/icons/platforms/Vercel';

import { useVercelIntegration } from './vercel/useVercelIntegration';

type Integration = {
  title: string;
  Icon: ReactNode;
  actionButton: (enabled: boolean) => ReactNode;
  description: string;
  upvoteId?: string;
};

const INTEGRATIONS: Integration[] = [
  {
    title: 'Vercel',
    Icon: <IconVercel className="text-white" />,
    actionButton: (enabled: boolean) => (
      <Button
        kind="success"
        appearance="solid"
        href={enabled ? '/settings/integrations/vercel' : '/settings/integrations/vercel/connect'}
        label={enabled ? 'Manage' : 'Connect'}
      />
    ),
    description:
      'Host your Inngest functions on Vercel and automatically sync them every time you deploy code.',
  },
  {
    title: 'Netlify',
    Icon: <IconNetlify className="text-white" />,
    actionButton: () => (
      <Button
        kind="default"
        appearance="outlined"
        label="View documentation"
        href="https://www.inngest.com/docs/deploy/netlify"
      />
    ),
    description:
      'Check out our docs to see how you can use Inngest with your applications deployed to Netlify.',
  },
  {
    title: 'Datadog',
    Icon: <IconDatadog className="text-white" />,
    actionButton: () => <Button kind="default" appearance="outlined" label="Upvote" />,
    description: 'Let us know if a Datadog integration is important to you by upvoting!',
  },
];

export default function IntegrationsList() {
  const { data: vercelData, fetching, error } = useVercelIntegration();
  return (
    <div className="mx-auto mt-16 flex w-[800px] flex-col">
      <div className="mb-7 w-full text-2xl font-medium">All integrations</div>
      <div className="grid w-[800px] grid-cols-2 gap-4">
        {INTEGRATIONS.map((i: Integration) => (
          <Card>
            <div className="flex h-[175px] flex-col p-6">
              <div className="align-center flex flex-row justify-between">
                <div className="flex h-8 w-8 items-center justify-center rounded bg-black">
                  {i.Icon}
                </div>
                {i.actionButton(i.title === 'Vercel' ? vercelData.enabled : false)}
              </div>
              <div className="mt-[18px] text-lg font-medium">{i.title}</div>
              <div className="mt-2 text-sm text-slate-500">{i.description}</div>
            </div>
          </Card>
        ))}
        <Card>
          <div className="flex h-[175px] flex-col bg-slate-100 p-6">
            <div className="text-lg font-medium">Can't find what you need?</div>
            <div className="mt-3 text-sm text-slate-500">
              Write to our team about the integration you are looking for and we will get back to
              you.
            </div>
            <div>
              <Button
                kind="default"
                appearance="outlined"
                label="Request integration"
                className="mt-3 bg-slate-100"
              />
            </div>
          </div>
        </Card>
      </div>
    </div>
  );
}
