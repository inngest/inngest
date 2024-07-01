import InngestLogo from '@/icons/InngestLogo';
import type VercelIntegration from '../../../settings/integrations/vercel/VercelIntegration';
import VercelIntegrationForm from './VercelIntegrationForm';
import type { VercelCallbackProps } from './page';

export default async function VercelIntegrationCallbackPage({
  vercelIntegration,
  searchParams,
}: VercelCallbackProps & { vercelIntegration: VercelIntegration }) {
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
