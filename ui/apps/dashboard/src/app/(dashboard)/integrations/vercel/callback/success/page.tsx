import { type Route } from 'next';

import Button from '@/components/Button';
import InngestLogo from '@/icons/InngestLogo';

type VercelIntegrationCallbackSuccessPageProps = {
  searchParams: {
    onSuccessRedirectURL: string;
  };
};

export default function VercelIntegrationCallbackSuccessPage({
  searchParams,
}: VercelIntegrationCallbackSuccessPageProps) {
  return (
    <div className="mx-auto flex h-full max-w-2xl flex-col justify-center gap-8 px-6">
      <header className="space-y-1">
        <InngestLogo />
        <p className="text-sm">The Inngest integration has successfully been installed!</p>
      </header>
      <main className="space-y-6">
        <h2 className="font-medium text-slate-700">What Has Been Set Up</h2>
        <ol className="list-inside list-decimal space-y-4 text-sm">
          <li>
            Each Vercel project will have <code>INNGEST_SIGNING_KEY</code> and{' '}
            <code>INNGEST_EVENT_KEY</code> environment variables set
          </li>
          <li>
            The next time you deploy your project to Vercel your functions will automatically appear
            in the Inngest dashboard
          </li>
        </ol>
        <div className="flex justify-end">
          <Button href={searchParams.onSuccessRedirectURL as Route}>Continue to Vercel →</Button>
        </div>
      </main>
    </div>
  );
}
