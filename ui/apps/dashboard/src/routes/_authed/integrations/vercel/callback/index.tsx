import { Connect } from '@/components/Integrations/Vercel/Connect';
import { createVercelIntegration } from '@/queries/server/integrations/vercel';
import { IconVercel } from '@inngest/components/icons/platforms/Vercel';
import { Link } from '@inngest/components/Link';
import { createFileRoute, type FileRouteTypes } from '@tanstack/react-router';

export type VercelCallbackProps = {
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

export const Route = createFileRoute('/_authed/integrations/vercel/callback/')({
  component: VercelCallbackComponent,
  validateSearch: (search) => search as VercelCallbackProps,
  loaderDeps: ({
    search: { code, teamId, configurationId, next, source },
  }) => ({
    code,
    teamId,
    configurationId,
    next,
    source,
  }),
  loader: async ({ deps: { code } }) => {
    if (!code) {
      throw new Error('Missing Vercel authorization code');
    }

    return await createVercelIntegration({
      data: { vercelAuthorizationCode: code },
    });
  },
});

function VercelCallbackComponent() {
  const searchParams = Route.useSearch();
  const vercelIntegration = Route.useLoaderData();
  if (!searchParams.code) {
    throw new Error('Missing Vercel authorization code');
  }

  return (
    <div className="mx-auto mt-8 flex w-[800px] flex-col p-8">
      <div className="mb-7 flex h-12 w-12 items-center justify-center rounded bg-black">
        <IconVercel className="text-alwaysWhite h-6 w-6" />
      </div>
      <div className="text-basis mb-2 text-2xl leading-loose">
        Connect Vercel to Inngest
      </div>
      <div className="text-muted mb-7 text-base">
        Select the Vercel projects that have Inngest functions. You can
        optionally specify server route other than the default{' '}
        <span className="font-semibold">(`/api/inngest`)</span>.
        <Link
          size="medium"
          to={'/create-organization/set-up' as FileRouteTypes['to']}
        >
          Learn more
        </Link>
      </div>
      <Connect searchParams={searchParams} integrations={vercelIntegration} />
    </div>
  );
}
