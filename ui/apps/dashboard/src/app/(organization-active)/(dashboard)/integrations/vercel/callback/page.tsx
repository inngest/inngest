import type { Route } from 'next';
import { Link } from '@inngest/components/Link/Link';
import { IconVercel } from '@inngest/components/icons/platforms/Vercel';

import VercelConnect from './connect';
import createVercelIntegration from './createVercelIntegration';

export type VercelCallbackProps = {
  searchParams: {
    // OAuth 2.0 authorization code issued by Vercel’s authorization server. This code is valid for
    // 30 minutes and can be only exchanged once for a long-lived access token.
    code: string;
    // The ID of the Vercel team that the user has selected. It is passed only if the user is
    // installing the integration on a team.
    teamId?: string;
    // The ID of the Vercel integration’s configuration.
    configurationId: string;
    // Encoded URL to redirect the user once the installation process is finished.
    next: string;
    // Source defines where the integration was installed from.
    source: string;
  };
};

export default async function VercelCallbackPage({ searchParams }: VercelCallbackProps) {
  if (!searchParams.code) {
    throw new Error('Missing Vercel authorization code');
  }
  const vercelIntegration = await createVercelIntegration({
    vercelAuthorizationCode: searchParams.code,
  });

  return (
    <div className="mx-auto mt-8 flex w-[800px] flex-col p-8">
      <div className="mb-7 flex h-12 w-12 items-center justify-center rounded bg-black">
        <IconVercel className="text-alwaysWhite h-6 w-6" />
      </div>
      <div className="text-basis mb-2 text-2xl leading-loose">Connect Vercel to Inngest</div>
      <div className="text-muted mb-7 text-base">
        Select the Vercel projects that have Inngest functions. You can optionally specify server
        route other than the default <span className="font-semibold">(`/api/inngest`)</span>.{' '}
        <Link size="medium" href={'/create-organization/set-up' as Route}>
          Learn more
        </Link>
      </div>
      <VercelConnect searchParams={searchParams} integrations={vercelIntegration} />
    </div>
  );
}

export const dynamic = 'force-dynamic';
