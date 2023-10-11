import InngestLogo from '@/icons/InngestLogo';
import VercelIntegrationForm from './VercelIntegrationForm';
import createVercelIntegration from './createVercelIntegration';

type VercelIntegrationCallbackPageProps = {
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

export default async function VercelIntegrationCallbackPage({
  searchParams,
}: VercelIntegrationCallbackPageProps) {
  if (!searchParams.code) {
    throw new Error('Missing Vercel authorization code');
  }
  const vercelIntegration = await createVercelIntegration({
    vercelAuthorizationCode: searchParams.code,
  });

  return (
    <div className="mx-auto flex h-full max-w-3xl flex-col justify-center gap-8 px-6">
      <header className="space-y-1">
        <InngestLogo />
        <p className="text-sm">
          Let’s connect your Vercel account to Inngest! Select which Vercel projects to enable.
        </p>
      </header>
      <main className="space-y-6">
        <header className="space-y-2">
          <h2 className="font-medium text-slate-700">Projects</h2>
          <p className="max-w-lg text-sm">
            Toggle each project that you have Inngest functions. You can optionally specify a custom
            serve route (see docs) other than the default.
          </p>
        </header>
        <VercelIntegrationForm
          vercelIntegration={vercelIntegration}
          onSuccessRedirectURL={searchParams.next}
        />
      </main>
    </div>
  );
}

export const dynamic = 'force-dynamic';
